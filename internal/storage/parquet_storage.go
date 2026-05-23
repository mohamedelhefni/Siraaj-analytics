package storage

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mohamedelhefni/siraaj/internal/domain"
)

const (
	// Default buffer size before flushing to disk
	DefaultBufferSize = 10000
	// Default flush interval
	DefaultFlushInterval = 30 * time.Second
	// Parquet file directory
	DefaultParquetDir = "data/events"
	// Temp CSV path for buffering
	TempCSVFile = "data/events_buffer.csv"
	// Max files before triggering merge
	MaxFilesBeforeMerge = 100
	// Merge check interval
	MergeCheckInterval = 5 * time.Minute
)

// ParquetStorage handles buffered writes to Parquet files using DuckDB COPY
// Uses append-only partitioned files for scalability
type ParquetStorage struct {
	db            *sql.DB
	dataDir       string // Directory containing partition files
	tempCSVPath   string
	buffer        []domain.Event
	bufferSize    int
	flushInterval time.Duration
	mu            sync.Mutex
	flushMu       sync.Mutex // Separate mutex for flush operations
	mergeMu       sync.Mutex // Separate mutex for merge operations
	stopChan      chan struct{}
	flushChan     chan struct{}
	wg            sync.WaitGroup
	idCounter     uint64
	fileCounter   int64 // Counter for generating unique filenames
}

// NewParquetStorage creates a new Parquet storage with buffering
// Uses partitioned append-only files for better scalability
func NewParquetStorage(db *sql.DB, dataDir string, bufferSize int, flushInterval time.Duration) (*ParquetStorage, error) {
	if dataDir == "" {
		dataDir = DefaultParquetDir
	}
	if bufferSize <= 0 {
		bufferSize = DefaultBufferSize
	}
	if flushInterval <= 0 {
		flushInterval = DefaultFlushInterval
	}

	// Ensure directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	ps := &ParquetStorage{
		db:            db,
		dataDir:       dataDir,
		tempCSVPath:   TempCSVFile,
		buffer:        make([]domain.Event, 0, bufferSize),
		bufferSize:    bufferSize,
		flushInterval: flushInterval,
		stopChan:      make(chan struct{}),
		flushChan:     make(chan struct{}, 1),
		idCounter:     1,
		fileCounter:   time.Now().Unix(), // Initialize with timestamp
	}

	// Start background flusher
	ps.wg.Add(1)
	go ps.backgroundFlusher()

	// Start background merger
	ps.wg.Add(1)
	go ps.backgroundMerger()

	log.Printf("✓ Parquet storage initialized: dir=%s, buffer_size=%d, flush_interval=%v",
		dataDir, bufferSize, flushInterval)

	return ps, nil
}

// GetNextID returns the next ID for event insertion
func (ps *ParquetStorage) GetNextID() uint64 {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	id := ps.idCounter
	ps.idCounter++
	return id
}

// Write adds an event to the buffer
func (ps *ParquetStorage) Write(event domain.Event) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.buffer = append(ps.buffer, event)

	// Check if buffer is full
	if len(ps.buffer) >= ps.bufferSize {
		log.Printf("📦 Buffer full (%d events), triggering flush...", len(ps.buffer))
		// Trigger flush without blocking
		select {
		case ps.flushChan <- struct{}{}:
		default:
			// Flush already pending
		}
	}

	return nil
}

// WriteBatch adds multiple events to the buffer
func (ps *ParquetStorage) WriteBatch(events []domain.Event) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.buffer = append(ps.buffer, events...)

	// Check if buffer is full
	if len(ps.buffer) >= ps.bufferSize {
		log.Printf("📦 Buffer full (%d events), triggering flush...", len(ps.buffer))
		// Trigger flush without blocking
		select {
		case ps.flushChan <- struct{}{}:
		default:
			// Flush already pending
		}
	}

	return nil
}

// backgroundFlusher runs in a goroutine and flushes buffer periodically
func (ps *ParquetStorage) backgroundFlusher() {
	defer ps.wg.Done()

	ticker := time.NewTicker(ps.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ps.stopChan:
			// Final flush before shutdown
			if err := ps.Flush(); err != nil {
				log.Printf("❌ Error during final flush: %v", err)
			}
			return

		case <-ticker.C:
			// Periodic flush
			if err := ps.Flush(); err != nil {
				log.Printf("❌ Error during periodic flush: %v", err)
			}

		case <-ps.flushChan:
			// Manual flush triggered by full buffer
			if err := ps.Flush(); err != nil {
				log.Printf("❌ Error during manual flush: %v", err)
			}
		}
	}
}

// Flush writes buffered events to a new Parquet file (append-only, no merge)
func (ps *ParquetStorage) Flush() error {
	// Use separate mutex to prevent concurrent flushes
	ps.flushMu.Lock()
	defer ps.flushMu.Unlock()

	ps.mu.Lock()
	if len(ps.buffer) == 0 {
		ps.mu.Unlock()
		return nil
	}

	// Copy buffer and clear it
	eventsToWrite := make([]domain.Event, len(ps.buffer))
	copy(eventsToWrite, ps.buffer)
	ps.buffer = ps.buffer[:0]
	ps.mu.Unlock()

	start := time.Now()
	log.Printf("💾 Flushing %d events to Parquet file...", len(eventsToWrite))

	// Write events to temporary CSV file
	csvFile, err := os.Create(ps.tempCSVPath)
	if err != nil {
		return fmt.Errorf("failed to create temp CSV: %w", err)
	}
	defer func() {
		if err := csvFile.Close(); err != nil {
			log.Printf("Warning: failed to close CSV file: %v", err)
		}
		if err := os.Remove(ps.tempCSVPath); err != nil {
			log.Printf("Warning: failed to remove temp CSV file: %v", err)
		}
	}()

	// Write CSV data
	if _, err := fmt.Fprintf(csvFile, "id,timestamp,event_name,user_id,session_id,session_duration,url,referrer,user_agent,ip,country,browser,os,device,is_bot,project_id,channel\n"); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}
	for _, event := range eventsToWrite {
		// Format timestamp as ISO8601 string for DuckDB
		timestampStr := event.Timestamp.UTC().Format("2006-01-02 15:04:05.000000")
		if _, err := fmt.Fprintf(csvFile, "%d,%s,%s,%s,%s,%d,%s,%s,%s,%s,%s,%s,%s,%s,%t,%s,%s\n",
			event.ID,
			timestampStr,
			escapeCsv(event.EventName),
			escapeCsv(event.UserID),
			escapeCsv(event.SessionID),
			event.SessionDuration,
			escapeCsv(event.URL),
			escapeCsv(event.Referrer),
			escapeCsv(event.UserAgent),
			escapeCsv(event.IP),
			escapeCsv(event.Country),
			escapeCsv(event.Browser),
			escapeCsv(event.OS),
			escapeCsv(event.Device),
			event.IsBot,
			escapeCsv(event.ProjectID),
			escapeCsv(event.Channel),
		); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	if err := csvFile.Close(); err != nil {
		return fmt.Errorf("failed to close CSV file: %w", err)
	}

	// Generate unique filename using timestamp and counter
	// This allows for append-only writes without merging
	ps.fileCounter++
	timestamp := time.Now().UTC().Format("20060102_150405")
	outputFile := fmt.Sprintf("%s/events_%s_%d.parquet", ps.dataDir, timestamp, ps.fileCounter)

	// Convert CSV to Parquet with ZSTD compression
	// Each file is independent and sorted by timestamp
	copyQuery := fmt.Sprintf(`
		COPY (
			SELECT 
				id,
				timestamp,
				date_trunc('hour', timestamp)  AS date_hour,
				date_trunc('day', timestamp)   AS date_day,
				date_trunc('month', timestamp) AS date_month,
				event_name,
				user_id,
				session_id,
				session_duration,
				url,
				referrer,
				user_agent,
				ip,
				country,
				browser,
				os,
				device,
				is_bot,
				project_id,
				channel
			FROM read_csv('%s', 
				AUTO_DETECT=TRUE,
				header=true,
				timestampformat='%%Y-%%m-%%d %%H:%%M:%%S.%%f'
			)
			ORDER BY timestamp
		) TO '%s' (FORMAT 'PARQUET', CODEC 'ZSTD', ROW_GROUP_SIZE 100000)
	`, ps.tempCSVPath, outputFile)

	_, err = ps.db.Exec(copyQuery)
	if err != nil {
		return fmt.Errorf("failed to create Parquet file: %w", err)
	}

	duration := time.Since(start)
	log.Printf("✅ Flushed %d events to %s in %v (%.0f events/sec)",
		len(eventsToWrite), outputFile, duration, float64(len(eventsToWrite))/duration.Seconds())

	return nil
}

// escapeCsv escapes CSV fields
func escapeCsv(s string) string {
	if s == "" {
		return ""
	}
	// Simple CSV escaping - quote fields with commas or quotes
	needsQuote := false
	for _, c := range s {
		if c == ',' || c == '"' || c == '\n' || c == '\r' {
			needsQuote = true
			break
		}
	}
	if needsQuote {
		// Escape quotes by doubling them
		var escaped strings.Builder
		for _, c := range s {
			if c == '"' {
				escaped.WriteString("\"\"")
			} else {
				escaped.WriteString(string(c))
			}
		}
		return "\"" + escaped.String() + "\""
	}
	return s
}

// Close gracefully shuts down the storage, flushing any remaining data
func (ps *ParquetStorage) Close() error {
	log.Println("🛑 Shutting down Parquet storage...")

	// Stop background flusher
	close(ps.stopChan)

	// Wait for background flusher to complete
	ps.wg.Wait()

	log.Println("✓ Parquet storage shut down successfully")
	return nil
}

// GetFilePath returns the Parquet directory path pattern for DuckDB queries
// Use with read_parquet('data/events/*.parquet') to query all files
func (ps *ParquetStorage) GetFilePath() string {
	return fmt.Sprintf("%s/*.parquet", ps.dataDir)
}

// backgroundMerger runs periodically to merge small Parquet files when there are too many
func (ps *ParquetStorage) backgroundMerger() {
	defer ps.wg.Done()

	ticker := time.NewTicker(MergeCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ps.stopChan:
			return

		case <-ticker.C:
			// Check file count and merge if needed
			if err := ps.checkAndMergeFiles(); err != nil {
				log.Printf("❌ Error during file merge: %v", err)
			}
		}
	}
}

// checkAndMergeFiles checks the number of Parquet files and merges them if needed
func (ps *ParquetStorage) checkAndMergeFiles() error {
	ps.mergeMu.Lock()
	defer ps.mergeMu.Unlock()

	// List all parquet files in directory
	files, err := os.ReadDir(ps.dataDir)
	if err != nil {
		return fmt.Errorf("failed to read data directory: %w", err)
	}

	// Count parquet files
	parquetFiles := []string{}
	for _, file := range files {
		if !file.IsDir() && len(file.Name()) > 8 && file.Name()[len(file.Name())-8:] == ".parquet" {
			parquetFiles = append(parquetFiles, file.Name())
		}
	}

	fileCount := len(parquetFiles)
	if fileCount <= MaxFilesBeforeMerge {
		return nil // No merge needed
	}

	log.Printf("🔄 Found %d Parquet files (max: %d), starting merge...", fileCount, MaxFilesBeforeMerge)
	start := time.Now()

	// Generate merged filename with timestamp
	timestamp := time.Now().UTC().Format("20060102_150405")
	mergedFile := fmt.Sprintf("%s/events_merged_%s.parquet", ps.dataDir, timestamp)
	tempMergedFile := mergedFile + ".tmp"

	// Use DuckDB to merge all files into one
	// This is efficient as DuckDB handles the Parquet format natively
	mergeQuery := fmt.Sprintf(`
		COPY (
			SELECT 
				id,
				timestamp,
				date_trunc('hour', timestamp)  AS date_hour,
				date_trunc('day', timestamp)   AS date_day,
				date_trunc('month', timestamp) AS date_month,
				event_name,
				user_id,
				session_id,
				session_duration,
				url,
				referrer,
				user_agent,
				ip,
				country,
				browser,
				os,
				device,
				is_bot,
				project_id,
				channel
			FROM read_parquet('%s/*.parquet')
			ORDER BY timestamp
		) TO '%s' (FORMAT 'PARQUET', CODEC 'ZSTD', ROW_GROUP_SIZE 100000)
	`, ps.dataDir, tempMergedFile)

	_, err = ps.db.Exec(mergeQuery)
	if err != nil {
		// Clean up temp file on error
		if removeErr := os.Remove(tempMergedFile); removeErr != nil {
			log.Printf("Warning: failed to remove temp merged file: %v", removeErr)
		}
		return fmt.Errorf("failed to merge Parquet files: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempMergedFile, mergedFile); err != nil {
		removeErr := os.Remove(tempMergedFile)
		if removeErr != nil {
			log.Printf("Warning: failed to remove temp merged file after rename failure: %v", removeErr)
		}
		return fmt.Errorf("failed to rename merged file: %w", err)
	}

	// Get merged file size
	mergedFileInfo, err := os.Stat(mergedFile)
	if err != nil {
		return fmt.Errorf("failed to stat merged file: %w", err)
	}

	// Delete old files
	deletedCount := 0
	for _, fileName := range parquetFiles {
		filePath := fmt.Sprintf("%s/%s", ps.dataDir, fileName)
		if err := os.Remove(filePath); err != nil {
			log.Printf("⚠️  Warning: failed to delete old file %s: %v", fileName, err)
		} else {
			deletedCount++
		}
	}

	duration := time.Since(start)
	log.Printf("✅ Merged %d files into 1 file (%.2f MB) in %v",
		deletedCount, float64(mergedFileInfo.Size())/(1024*1024), duration)

	return nil
}

// GetFileCount returns the current number of Parquet files
func (ps *ParquetStorage) GetFileCount() (int, error) {
	files, err := os.ReadDir(ps.dataDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read data directory: %w", err)
	}

	count := 0
	for _, file := range files {
		if !file.IsDir() && len(file.Name()) > 8 && file.Name()[len(file.Name())-8:] == ".parquet" {
			count++
		}
	}

	return count, nil
}
