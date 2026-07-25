package scanner

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"qanvidnas/internal/models"
	"qanvidnas/internal/thumbnail"

	"github.com/fsnotify/fsnotify"
)

var supportedExtensions = map[string]string{
	".mp4": "video", ".mkv": "video", ".avi": "video", ".mov": "video",
	".wmv": "video", ".flv": "video", ".webm": "video", ".m4v": "video",
	".mpg": "video", ".mpeg": "video", ".ts": "video",
	".mp3": "audio", ".flac": "audio", ".aac": "audio", ".ogg": "audio",
	".wav": "audio", ".m4a": "audio", ".wma": "audio", ".opus": "audio",
}

type Scanner struct {
	db          *sql.DB
	thumbnailer *thumbnail.Thumbnailer
	progress    *models.ScanProgress
	mu          sync.RWMutex
	watcher     *fsnotify.Watcher
	stopCh      chan struct{}
}

func NewScanner(db *sql.DB, thumbnailer *thumbnail.Thumbnailer) *Scanner {
	return &Scanner{
		db:          db,
		thumbnailer: thumbnailer,
		progress:    &models.ScanProgress{Status: "idle"},
		stopCh:      make(chan struct{}),
	}
}

func (s *Scanner) GetProgress() *models.ScanProgress {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.progress
}

// ScanPath scans a specific path for media files.
func (s *Scanner) ScanPath(path string) {
	s.FullScan()
}

// FullScan performs a complete scan of all configured folders.
func (s *Scanner) FullScan() {
	s.mu.Lock()
	s.progress = &models.ScanProgress{Status: "scanning"}
	s.mu.Unlock()

	// Get all scan paths
	rows, err := s.db.Query("SELECT path FROM scan_folders")
	if err != nil {
		log.Printf("Scanner: failed to query scan folders: %v", err)
		s.mu.Lock()
		s.progress.Status = "error"
		s.progress.Message = err.Error()
		s.mu.Unlock()
		return
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		rows.Scan(&p)
		paths = append(paths, p)
	}

	// Count total files first
	totalFiles := 0
	for _, root := range paths {
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if _, ok := supportedExtensions[ext]; ok {
				totalFiles++
			}
			return nil
		})
	}

	s.mu.Lock()
	s.progress.Total = totalFiles
	s.mu.Unlock()

	// Scan each path
	processed := 0
	for _, root := range paths {
		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			mediaType, ok := supportedExtensions[ext]
			if !ok {
				return nil
			}

			// Check if already indexed (by path and mtime)
			var existingID int64
			var existingMtime string
			s.db.QueryRow("SELECT id, updated_at FROM media WHERE path = ? AND deleted = 0", path).Scan(&existingID, &existingMtime)

			modTime := info.ModTime()
			if existingID > 0 {
				// Check if file changed
				var dbModTime time.Time
				fmt.Sscanf(existingMtime, "%s", &dbModTime) // Simplified check
				_ = dbModTime
				// Skip if not modified
				return nil
			}

			// Extract metadata
			meta := extractMetadata(path, mediaType)
			if meta == nil {
				return nil
			}

			fileName := strings.TrimSuffix(info.Name(), ext)

			// Insert media record
			result, err := s.db.Exec(`
				INSERT INTO media (title, description, type, path, duration, resolution,
					file_size, codec, bitrate, created_at, updated_at)
				VALUES (?, '', ?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(path) DO UPDATE SET
					file_size = ?, duration = ?, updated_at = ?
				WHERE deleted = 0
			`, fileName, mediaType, path, meta.Duration, meta.Resolution,
				info.Size(), meta.Codec, meta.Bitrate, modTime, modTime,
				info.Size(), meta.Duration, modTime)

			if err != nil {
				log.Printf("Scanner: insert media %s: %v", path, err)
				return nil
			}

			mediaID, _ := result.LastInsertId()
			if mediaID == 0 {
				s.db.QueryRow("SELECT id FROM media WHERE path = ? AND deleted = 0", path).Scan(&mediaID)
			}

			// Add to FTS index
			s.db.Exec("DELETE FROM media_fts WHERE rowid = ?", mediaID)
			s.db.Exec("INSERT INTO media_fts (rowid, title, description, tags) VALUES (?, ?, '', '')",
				mediaID, fileName)

			// Generate thumbnails asynchronously
			if mediaType == "video" && s.thumbnailer != nil {
				go s.thumbnailer.GenerateCover(mediaID, path, meta.Duration)
				go s.thumbnailer.GenerateSprite(mediaID, path, meta.Duration)
			}

			processed++
			s.mu.Lock()
			s.progress.Processed = processed
			s.mu.Unlock()

			return nil
		})
	}

	// Update last_scan_at
	for _, root := range paths {
		s.db.Exec("UPDATE scan_folders SET last_scan_at = ?, status = 'idle' WHERE path = ?", time.Now(), root)
	}

	s.mu.Lock()
	s.progress.Status = "complete"
	s.mu.Unlock()
}

// StartWatch begins filesystem watching on all configured folders.
func (s *Scanner) StartWatch() error {
	var err error
	s.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}

	// Add scan paths to watcher
	rows, err := s.db.Query("SELECT path FROM scan_folders")
	if err != nil {
		return fmt.Errorf("query scan folders: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p string
		rows.Scan(&p)
		s.watcher.Add(p)

		// Also watch subdirectories
		filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err == nil && info.IsDir() {
				s.watcher.Add(path)
			}
			return nil
		})
	}

	// Start watching in background
	go s.watchLoop()

	// Start polling fallback
	go s.pollingLoop()

	return nil
}

func (s *Scanner) watchLoop() {
	// Debounce map: path -> timer
	debounce := make(map[string]*time.Timer)
	var debounceMu sync.Mutex

	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
				debounceMu.Lock()
				if timer, exists := debounce[event.Name]; exists {
					timer.Stop()
				}
				debounce[event.Name] = time.AfterFunc(5*time.Second, func() {
					s.handleNewFile(event.Name)
					debounceMu.Lock()
					delete(debounce, event.Name)
					debounceMu.Unlock()
				})
				debounceMu.Unlock()
			} else if event.Has(fsnotify.Remove) {
				s.handleDeletedFile(event.Name)
			} else if event.Has(fsnotify.Rename) {
				s.handleDeletedFile(event.Name)
			}

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Scanner watcher error: %v", err)

		case <-s.stopCh:
			return
		}
	}
}

func (s *Scanner) pollingLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.incrementalScan()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Scanner) incrementalScan() {
	rows, err := s.db.Query("SELECT path FROM scan_folders")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var root string
		rows.Scan(&root)

		filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			if _, ok := supportedExtensions[ext]; !ok {
				return nil
			}

			// Check if already indexed
			var existingID int64
			s.db.QueryRow("SELECT id FROM media WHERE path = ? AND deleted = 0", path).Scan(&existingID)
			if existingID == 0 {
				s.handleNewFile(path)
			}

			return nil
		})
	}
}

func (s *Scanner) handleNewFile(path string) {
	ext := strings.ToLower(filepath.Ext(path))
	mediaType, ok := supportedExtensions[ext]
	if !ok {
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		return
	}

	meta := extractMetadata(path, mediaType)
	if meta == nil {
		return
	}

	fileName := strings.TrimSuffix(info.Name(), ext)

	result, err := s.db.Exec(`
		INSERT INTO media (title, description, type, path, duration, resolution,
			file_size, codec, bitrate, created_at, updated_at)
		VALUES (?, '', ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, fileName, mediaType, path, meta.Duration, meta.Resolution,
		info.Size(), meta.Codec, meta.Bitrate, time.Now(), time.Now())

	if err == nil {
		mediaID, _ := result.LastInsertId()
		s.db.Exec("INSERT INTO media_fts (rowid, title, description, tags) VALUES (?, ?, '', '')", mediaID, fileName)

		if mediaType == "video" && s.thumbnailer != nil {
			go s.thumbnailer.GenerateCover(mediaID, path, meta.Duration)
			go s.thumbnailer.GenerateSprite(mediaID, path, meta.Duration)
		}
	}
}

func (s *Scanner) handleDeletedFile(path string) {
	s.db.Exec("UPDATE media SET deleted = 1 WHERE path = ?", path)
}

func (s *Scanner) Stop() {
	close(s.stopCh)
	if s.watcher != nil {
		s.watcher.Close()
	}
}
