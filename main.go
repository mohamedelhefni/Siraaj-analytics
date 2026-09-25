package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"database/sql/driver"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	duckdb "github.com/duckdb/duckdb-go/v2"
	"github.com/mohamedelhefni/siraaj/geolocation"
	"github.com/mohamedelhefni/siraaj/internal/handler"
	"github.com/mohamedelhefni/siraaj/internal/middleware"
	"github.com/mohamedelhefni/siraaj/internal/migrations"
	"github.com/mohamedelhefni/siraaj/internal/repository"
	"github.com/mohamedelhefni/siraaj/internal/service"
)

//go:embed all:ui/dashboard
var dashboardFiles embed.FS

//go:embed ui/landing/index.html
var landingPage string

type cleanURLFS struct {
	fs.FS
}

func (f cleanURLFS) Open(name string) (fs.File, error) {
	file, err := f.FS.Open(name)
	if errors.Is(err, fs.ErrNotExist) && path.Ext(name) == "" {
		return f.FS.Open(name + ".html")
	}
	return file, err
}

// initDatabase initializes the database connection and runs migrations
func initDatabase(dbPath string) (*duckdb.Connector, *sql.DB, driver.Conn, error) {
	memoryLimit := os.Getenv("DUCKDB_MEMORY_LIMIT")
	if memoryLimit == "" {
		memoryLimit = "4GB"
	}
	threads := os.Getenv("DUCKDB_THREADS")
	if threads == "" {
		threads = "4"
	}

	connector, err := duckdb.NewConnector(dbPath, func(execer driver.ExecerContext) error {
		bootQueries := []string{
			fmt.Sprintf("PRAGMA memory_limit='%s'", memoryLimit),
			fmt.Sprintf("PRAGMA threads=%s", threads),
			"SET enable_object_cache=true",
			"SET preserve_insertion_order=false",
			"SET temp_directory='/tmp/duckdb_temp'",
			"SET enable_http_metadata_cache=true",
		}
		for _, q := range bootQueries {
			if _, err := execer.ExecContext(context.Background(), q, nil); err != nil {
				log.Printf("Warning: Could not apply %s: %v", q, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create connector: %v", err)
	}

	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		connector.Close()
		return nil, nil, nil, fmt.Errorf("failed to ping database: %v", err)
	}

	log.Printf("✓ DuckDB memory limit set to: %s", memoryLimit)
	log.Printf("✓ DuckDB threads set to: %s", threads)

	// Run migrations
	if err := migrations.Migrate(db); err != nil {
		connector.Close()
		return nil, nil, nil, fmt.Errorf("failed to run migrations: %v", err)
	}

	// Create a dedicated connection for the Appender
	appenderConn, err := connector.Connect(context.Background())
	if err != nil {
		connector.Close()
		return nil, nil, nil, fmt.Errorf("failed to create appender connection: %v", err)
	}

	return connector, db, appenderConn, nil
}

func main() {
	// Initialize geolocation service
	geoService, err := geolocation.NewService()
	if err != nil {
		log.Printf("⚠️  Warning: Geolocation service unavailable: %v", err)
		log.Println("⚠️  Continuing without geolocation support...")
		geoService = nil
	}
	if geoService != nil {
		defer func() {
			if err := geoService.Close(); err != nil {
				log.Printf("Warning: failed to close geolocation service: %v", err)
			}
		}()
	}

	// Initialize database first (needed for Parquet storage)
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/analytics.db"
	}

	connector, db, appenderConn, err := initDatabase(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Warning: failed to close database: %v", err)
		}
		if err := connector.Close(); err != nil {
			log.Printf("Warning: failed to close connector: %v", err)
		}
	}()

	log.Println("✓ DuckDB initialized successfully")

	// Initialize repository with DuckDB + dedicated Appender connection
	baseRepo := repository.NewEventRepository(db, appenderConn)
	defer func() {
		if err := baseRepo.Close(); err != nil {
			log.Printf("Warning: failed to close repository: %v", err)
		}
	}()

	eventService := service.NewEventService(baseRepo)
	eventHandler := handler.NewEventHandler(eventService, geoService)
	linkRepo := repository.NewLinkRepository(db)
	linkService := service.NewLinkService(linkRepo)
	linkHandler := handler.NewLinkHandler(linkService, geoService)
	authRepo := repository.NewAuthRepository(db)
	authService := service.NewAuthService(authRepo, authSecret(), authTokenTTL())
	authHandler := handler.NewAuthHandler(authService)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n🛑 Shutting down gracefully...")

		// Close repository first to flush any pending data
		if err := baseRepo.Close(); err != nil {
			log.Printf("Error closing repository: %v", err)
		}

		// Close other resources
		if geoService != nil {
			if err := geoService.Close(); err != nil {
				log.Printf("Error closing geolocation service: %v", err)
			}
		}

		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}

		os.Exit(0)
	}()

	// Setup HTTP routes
	mux := http.NewServeMux()
	access := func(next http.HandlerFunc) http.Handler { return middleware.AccessAuth(authService, next) }
	tracking := func(next http.HandlerFunc) http.Handler { return middleware.TrackingAuth(authService, next) }
	loginLimiter := middleware.NewRateLimiter(10, 15*time.Minute)
	signupLimiter := middleware.NewRateLimiter(5, time.Hour)

	// Authentication and user management
	mux.Handle("/api/auth/bootstrap", signupLimiter.Limit(http.HandlerFunc(authHandler.Bootstrap)))
	mux.Handle("/api/auth/login", loginLimiter.Limit(http.HandlerFunc(authHandler.Login)))
	mux.Handle("/api/auth/signup", signupLimiter.Limit(http.HandlerFunc(authHandler.Signup)))
	mux.Handle("/api/auth/me", access(authHandler.Me))
	mux.Handle("/api/users", access(authHandler.Users))
	mux.Handle("/api/tracking-tokens", access(authHandler.TrackingTokens))

	// Event ingestion requires a project-scoped tracking token. Analytics reads require a user session.
	mux.Handle("/api/track", tracking(eventHandler.TrackEvent))
	mux.Handle("/api/track/batch", tracking(eventHandler.TrackBatchEvents))
	mux.Handle("/api/stats", access(eventHandler.GetStats))
	mux.Handle("/api/events", access(eventHandler.GetEvents))
	mux.Handle("/api/online", access(eventHandler.GetOnlineUsers))
	mux.Handle("/api/projects", access(authHandler.Projects))
	mux.Handle("/api/funnel", access(eventHandler.GetFunnelAnalysis))
	mux.HandleFunc("/api/health", eventHandler.Health)
	mux.Handle("/api/geo", access(eventHandler.GeoTest))
	mux.Handle("/api/links", access(linkHandler.Links))
	mux.Handle("/api/links/stats", access(linkHandler.Stats))
	mux.HandleFunc("/s/", linkHandler.Redirect)

	mux.Handle("/api/stats/overview", access(eventHandler.GetTopStats))
	mux.Handle("/api/stats/timeline", access(eventHandler.GetTimeline))
	mux.Handle("/api/stats/pages", access(eventHandler.GetTopPagesHandler))
	mux.Handle("/api/stats/pages/entry-exit", access(eventHandler.GetEntryExitPagesHandler))
	mux.Handle("/api/stats/countries", access(eventHandler.GetTopCountriesHandler))
	mux.Handle("/api/stats/sources", access(eventHandler.GetTopSourcesHandler))
	mux.Handle("/api/stats/events", access(eventHandler.GetTopEventsHandler))
	mux.Handle("/api/stats/devices", access(eventHandler.GetBrowsersDevicesOSHandler))

	// Channel analytics
	mux.Handle("/api/channels", access(eventHandler.GetChannelsHandler))

	// Serve the dashboard shell publicly so users can reach the login screen. All data APIs are protected above.
	dashboardFS, err := fs.Sub(dashboardFiles, "ui/dashboard")
	if err != nil {
		log.Printf("Warning: Could not load dashboard: %v", err)
	} else {
		dashboardHandler := http.StripPrefix("/dashboard", http.FileServer(http.FS(cleanURLFS{dashboardFS})))
		mux.Handle("/dashboard/", dashboardHandler)
	}

	// Serve landing page at root
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write([]byte(landingPage)); err != nil {
			log.Printf("Error serving landing page: %v", err)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 Analytics Server")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("🎨 Dashboard:  http://localhost:%s/dashboard/\n", port)
	fmt.Printf("📡 API Track:  http://localhost:%s/api/track\n", port)
	fmt.Printf("📦 API Batch:  http://localhost:%s/api/track/batch\n", port)
	fmt.Printf("📈 API Stats:  http://localhost:%s/api/stats\n", port)
	fmt.Printf("🌍 Geo Test:   http://localhost:%s/api/geo\n", port)
	fmt.Printf("❤️  Health:    http://localhost:%s/api/health\n", port)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✓ Server ready - Using official DuckDB Go driver")
	fmt.Println("✓ Svelte Dashboard embedded and ready")
	fmt.Println("🔒 Dashboard APIs protected with user authentication")
	fmt.Println("🔑 Event ingestion requires a project tracking token")
	if geoService != nil {
		fmt.Println("✓ Geolocation service enabled")
	} else {
		fmt.Println("⚠️  Geolocation service disabled")
	}
	fmt.Println("✓ Clean Architecture implemented")
	fmt.Printf("✓ DuckDB native storage: %s\n", dbPath)
	fmt.Println()

	// Apply middleware: CORS and Logging
	httpHandler := middleware.SecurityHeaders(middleware.CORS(middleware.Logging(mux)))
	log.Fatal(http.ListenAndServe(":"+port, httpHandler))
}

func authSecret() []byte {
	if configured := os.Getenv("AUTH_SECRET"); configured != "" {
		if len(configured) < 32 {
			log.Fatal("AUTH_SECRET must be at least 32 characters")
		}
		return []byte(configured)
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		log.Fatalf("failed to generate authentication secret: %v", err)
	}
	log.Print("⚠️  AUTH_SECRET is unset; using an ephemeral secret. Sessions will expire on restart.")
	return secret
}

func authTokenTTL() time.Duration {
	configured := os.Getenv("AUTH_TOKEN_TTL")
	if configured == "" {
		return 24 * time.Hour
	}
	ttl, err := time.ParseDuration(configured)
	if err != nil || ttl <= 0 {
		log.Fatal("AUTH_TOKEN_TTL must be a positive Go duration such as 24h")
	}
	return ttl
}
