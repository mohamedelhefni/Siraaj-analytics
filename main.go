package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
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

	// API endpoints
	mux.HandleFunc("/api/track", eventHandler.TrackEvent)
	mux.HandleFunc("/api/track/batch", eventHandler.TrackBatchEvents)
	mux.HandleFunc("/api/stats", eventHandler.GetStats)
	mux.HandleFunc("/api/events", eventHandler.GetEvents)
	mux.HandleFunc("/api/online", eventHandler.GetOnlineUsers)
	mux.HandleFunc("/api/projects", eventHandler.GetProjects)
	mux.HandleFunc("/api/funnel", eventHandler.GetFunnelAnalysis)
	mux.HandleFunc("/api/health", eventHandler.Health)
	mux.HandleFunc("/api/geo", eventHandler.GeoTest)

	// New focused stats endpoints
	mux.HandleFunc("/api/stats/overview", eventHandler.GetTopStats)
	mux.HandleFunc("/api/stats/timeline", eventHandler.GetTimeline)
	mux.HandleFunc("/api/stats/pages", eventHandler.GetTopPagesHandler)
	mux.HandleFunc("/api/stats/pages/entry-exit", eventHandler.GetEntryExitPagesHandler)
	mux.HandleFunc("/api/stats/countries", eventHandler.GetTopCountriesHandler)
	mux.HandleFunc("/api/stats/sources", eventHandler.GetTopSourcesHandler)
	mux.HandleFunc("/api/stats/events", eventHandler.GetTopEventsHandler)
	mux.HandleFunc("/api/stats/devices", eventHandler.GetBrowsersDevicesOSHandler)

	// Channel analytics
	mux.HandleFunc("/api/channels", eventHandler.GetChannelsHandler)

	// Debug endpoint to show all events
	mux.HandleFunc("/api/debug/events", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, timestamp, event_name, user_id FROM events ORDER BY timestamp DESC LIMIT 50")
		if err != nil {
			log.Printf("Error querying events: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		defer func() {
			if err := rows.Close(); err != nil {
				log.Printf("Warning: failed to close rows: %v", err)
			}
		}()

		events := []map[string]any{}
		for rows.Next() {
			var id uint64
			var timestamp time.Time
			var eventName, userID string
			if err := rows.Scan(&id, &timestamp, &eventName, &userID); err != nil {
				continue
			}
			events = append(events, map[string]any{
				"id":         id,
				"timestamp":  timestamp.Format(time.RFC3339),
				"event_name": eventName,
				"user_id":    userID,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"events": events,
			"count":  len(events),
		}); err != nil {
			log.Printf("Error encoding debug events: %v", err)
		}
	})

	// Database stats endpoint
	mux.HandleFunc("/api/debug/storage", func(w http.ResponseWriter, r *http.Request) {
		var tableSize int64
		err := db.QueryRow("SELECT COUNT(*) FROM events").Scan(&tableSize)
		if err != nil {
			log.Printf("Error getting table size: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"total_events":  tableSize,
			"storage_type":  "DuckDB Native",
			"database_path": dbPath,
		}); err != nil {
			log.Printf("Error encoding storage stats: %v", err)
		}
	})

	// Serve dashboard (SvelteKit app) with optional BasicAuth
	dashboardFS, err := fs.Sub(dashboardFiles, "ui/dashboard")
	if err != nil {
		log.Printf("Warning: Could not load dashboard: %v", err)
	} else {
		dashboardHandler := http.StripPrefix("/dashboard", http.FileServer(http.FS(dashboardFS)))
		// Apply BasicAuth middleware to dashboard routes
		mux.Handle("/dashboard/", middleware.BasicAuth(dashboardHandler))
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
	if os.Getenv("DASHBOARD_USERNAME") != "" && os.Getenv("DASHBOARD_PASSWORD") != "" {
		fmt.Println("🔒 Dashboard protected with Basic Authentication")
	} else {
		fmt.Println("⚠️  Dashboard is publicly accessible (set DASHBOARD_USERNAME and DASHBOARD_PASSWORD to enable auth)")
	}
	if geoService != nil {
		fmt.Println("✓ Geolocation service enabled")
	} else {
		fmt.Println("⚠️  Geolocation service disabled")
	}
	fmt.Println("✓ Clean Architecture implemented")
	fmt.Printf("✓ DuckDB native storage: %s\n", dbPath)
	fmt.Println()

	// Apply middleware: CORS and Logging
	httpHandler := middleware.CORS(middleware.Logging(mux))
	log.Fatal(http.ListenAndServe(":"+port, httpHandler))
}
