// rotation.go: Core rotation logic and file operations
//
// Copyright (c) 2025 AGILira
// Series: an AGILira fragment
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// initFile creates and opens the initial log file
// Called lazily on first write
func (l *Logger) initFile() error {
	l.initSizeConfig()
	retryCount, retryDelay, fileMode := l.getRetryConfig()

	sanitizedPath, err := l.validateAndSanitizePath()
	if err != nil {
		return err
	}

	if err := l.createLogDirectory(sanitizedPath, retryCount, retryDelay); err != nil {
		return err
	}

	file, err := l.openLogFile(sanitizedPath, fileMode, retryCount, retryDelay)
	if err != nil {
		return err
	}

	return l.initFileState(file, sanitizedPath)
}

// initSizeConfig initializes the size configuration with backward compatibility
func (l *Logger) initSizeConfig() {
	if l.maxSizeBytes != 0 {
		return
	}

	if l.MaxSizeStr != "" {
		// Use new string-based configuration
		if size, err := ParseSize(l.MaxSizeStr); err == nil {
			l.maxSizeBytes = size
		} else {
			l.reportError("size_parse", fmt.Errorf("invalid MaxSizeStr %q: %v", l.MaxSizeStr, err))
		}
	} else if l.MaxSize > 0 {
		// Fallback to legacy MB-based configuration
		l.maxSizeBytes = l.MaxSize * 1024 * 1024 // MB to bytes
	}
}

// validateAndSanitizePath validates and sanitizes the log file path
func (l *Logger) validateAndSanitizePath() (string, error) {
	if err := ValidatePathLength(l.Filename); err != nil {
		return "", fmt.Errorf("invalid log file path: %v", err)
	}

	// Sanitize filename for cross-platform compatibility
	dir := filepath.Dir(l.Filename)
	base := filepath.Base(l.Filename)
	sanitizedBase := SanitizeFilename(base)
	return filepath.Join(dir, sanitizedBase), nil
}

// createLogDirectory creates the log directory if needed
func (l *Logger) createLogDirectory(sanitizedPath string, retryCount int, retryDelay time.Duration) error {
	dir := filepath.Dir(sanitizedPath)
	if dir == "." {
		return nil
	}

	err := RetryFileOperation(func() error {
		return os.MkdirAll(dir, 0750) // More secure permissions
	}, retryCount, retryDelay)

	if err != nil {
		l.reportError("directory_creation", fmt.Errorf("failed to create log directory %q: %v (check permissions and disk space)", dir, err))
		return fmt.Errorf("failed to create log directory: %v", err)
	}
	return nil
}

// openLogFile opens or creates the log file with retry
func (l *Logger) openLogFile(sanitizedPath string, fileMode os.FileMode, retryCount int, retryDelay time.Duration) (*os.File, error) {
	var file *os.File
	err := RetryFileOperation(func() error {
		var err error
		file, err = os.OpenFile(sanitizedPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, fileMode) // #nosec G304 -- sanitizedPath validated by SanitizeFilename above
		return err
	}, retryCount, retryDelay)

	if err != nil {
		l.reportError("file_open", fmt.Errorf("failed to open log file %q: %v (check permissions and disk space)", sanitizedPath, err))
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}
	return file, nil
}

// initFileState initializes the file state after successful file creation
func (l *Logger) initFileState(file *os.File, sanitizedPath string) error {
	// Get current size with detailed error reporting
	info, err := file.Stat()
	if err != nil {
		_ = file.Close() // Ignore close error during cleanup
		l.reportError("file_stat", fmt.Errorf("failed to stat log file %q: %v", sanitizedPath, err))
		return fmt.Errorf("failed to stat log file: %v", err)
	}

	// Update the filename to the sanitized version
	l.Filename = sanitizedPath

	// Store file, size and creation time atomically
	l.currentFile.Store(file)

	// Safely convert file size, handling negative values
	size := info.Size()
	if size < 0 {
		size = 0 // Treat negative size as 0
	}
	l.bytesWritten.Store(uint64(size)) // #nosec G115 -- size checked for negative values above

	// Use cached time for better performance
	if l.timeCache != nil {
		l.fileCreated.Store(l.timeCache.CachedTime().Unix())
	} else {
		l.fileCreated.Store(time.Now().Unix())
	}

	return nil
}

// performRotation does the actual file rotation
func (l *Logger) performRotation() error {
	currentFile := l.currentFile.Load()
	if currentFile == nil {
		return fmt.Errorf("no current file to rotate")
	}

	backupName := l.generateBackupName()
	retryCount, retryDelay, fileMode := l.getRetryConfig()

	if err := l.closeAndRotateFile(currentFile, backupName, retryCount, retryDelay, fileMode); err != nil {
		return err
	}

	l.updateRotationState()
	l.scheduleBackgroundTasks(backupName)

	return nil
}

// generateBackupName creates a timestamped backup filename
func (l *Logger) generateBackupName() string {
	var now time.Time
	if l.timeCache != nil {
		now = l.timeCache.CachedTime()
	} else {
		now = time.Now()
	}
	if !l.LocalTime {
		now = now.UTC()
	}
	return fmt.Sprintf("%s.%s", l.Filename, now.Format("2006-01-02-15-04-05"))
}

// getRetryConfig returns retry configuration with defaults
func (l *Logger) getRetryConfig() (int, time.Duration, os.FileMode) {
	retryCount := l.RetryCount
	if retryCount == 0 {
		retryCount = 3
	}

	retryDelay := l.RetryDelay
	if retryDelay == 0 {
		retryDelay = 10 * time.Millisecond
	}

	fileMode := l.FileMode
	if fileMode == 0 {
		fileMode = GetDefaultFileMode()
	}

	return retryCount, retryDelay, fileMode
}

// closeAndRotateFile handles the file rotation operation
func (l *Logger) closeAndRotateFile(currentFile *os.File, backupName string, retryCount int, retryDelay time.Duration, fileMode os.FileMode) error {
	// Close current file with retry
	err := RetryFileOperation(func() error {
		return currentFile.Close()
	}, retryCount, retryDelay)
	if err != nil {
		return fmt.Errorf("failed to close current file: %v", err)
	}

	// Rename current file to backup with retry
	err = RetryFileOperation(func() error {
		return os.Rename(l.Filename, backupName)
	}, retryCount, retryDelay)
	if err != nil {
		return fmt.Errorf("failed to rename log file: %v", err)
	}

	// Small delay to ensure file handles are released (Windows)
	time.Sleep(retryDelay)

	// Create new file with retry
	var newFile *os.File
	err = RetryFileOperation(func() error {
		var err error
		newFile, err = os.OpenFile(l.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, fileMode) // #nosec G304 -- l.Filename is controlled by application, not user input
		return err
	}, retryCount, retryDelay)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %v", err)
	}

	// Update atomic pointer to new file
	l.currentFile.Store(newFile)
	return nil
}

// updateRotationState updates internal rotation state
func (l *Logger) updateRotationState() {
	l.bytesWritten.Store(0)
	if l.timeCache != nil {
		l.fileCreated.Store(l.timeCache.CachedTime().Unix())
	} else {
		l.fileCreated.Store(time.Now().Unix())
	}
	l.rotationSeq.Add(1)
}

// scheduleBackgroundTasks submits background tasks for cleanup, compression, etc.
func (l *Logger) scheduleBackgroundTasks(backupName string) {
	// Initialize background workers if needed
	if l.bgWorkers.Load() == nil {
		workers := newBackgroundWorkers(2)
		l.bgWorkers.Store(workers)
	}

	workers := l.bgWorkers.Load()
	if workers == nil {
		return
	}

	// Submit cleanup task if needed (least intrusive)
	if l.MaxBackups > 0 {
		l.safeSubmitTask(BackgroundTask{
			TaskType: "cleanup",
			Logger:   l,
		})
	}

	// Submit checksum task if enabled (read-only, safer)
	if l.Checksum {
		l.safeSubmitTask(BackgroundTask{
			TaskType: "checksum",
			FilePath: backupName,
			Logger:   l,
		})
	}

	// Submit compression task if enabled
	if l.Compress {
		l.safeSubmitTask(BackgroundTask{
			TaskType: "compress",
			FilePath: backupName,
			Logger:   l,
		})
	}
}

// safeSubmitTask submits a task only if workers are still active
func (l *Logger) safeSubmitTask(task BackgroundTask) {
	workers := l.bgWorkers.Load()
	if workers == nil {
		return // Workers shut down
	}

	// Check if context is cancelled first
	select {
	case <-workers.ctx.Done():
		return // Workers are shutting down
	default:
	}

	// Use non-blocking submit to avoid panics
	select {
	case workers.taskQueue <- task:
		// Task submitted successfully
	case <-workers.ctx.Done():
		// Workers shut down while we were trying to submit
		return
	default:
		// Queue is full, skip task
	}
}

// fileInfo holds file information for sorting
type fileInfo struct {
	name    string
	modTime time.Time
}

// cleanupOldFiles removes old backup files based on MaxBackups and MaxFileAge settings
func (l *Logger) cleanupOldFiles() {
	// Find all backup files using proper filepath operations
	pattern := l.Filename + ".*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	// Get file info for all backup files
	var files []fileInfo
	var now time.Time
	if l.timeCache != nil {
		now = l.timeCache.CachedTime()
	} else {
		now = time.Now()
	}

	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue // Skip files we can't stat
		}

		// Check age-based cleanup first
		if l.MaxFileAge > 0 {
			fileAge := now.Sub(info.ModTime())
			if fileAge > l.MaxFileAge {
				// File is too old, remove it
				err := os.Remove(match)
				if err != nil {
					l.reportError("age_cleanup", fmt.Errorf("failed to remove old file %s (age: %v): %v", match, fileAge, err))
				}
				continue // Don't include in files list since it's removed
			}
		}

		files = append(files, fileInfo{
			name:    match,
			modTime: info.ModTime(),
		})
	}

	// Apply count-based cleanup (MaxBackups)
	if l.MaxBackups <= 0 || len(files) <= l.MaxBackups {
		return // Nothing to clean up by count
	}

	// Sort by modification time (oldest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	// Remove oldest files beyond MaxBackups
	filesToRemove := len(files) - l.MaxBackups
	for i := 0; i < filesToRemove; i++ {
		err := os.Remove(files[i].name)
		if err != nil {
			l.reportError("count_cleanup", fmt.Errorf("failed to remove excess backup file %s: %v", files[i].name, err))
		}
	}
}

// compressFile compresses a rotated log file using gzip with crash consistency
func (l *Logger) compressFile(filename string) {
	// Open source file with retry (file might be in use during high-frequency rotation)
	var source *os.File
	err := RetryFileOperation(func() error {
		var err error
		source, err = os.Open(filename) // #nosec G304 -- filename is internal backup file path, not user input
		return err
	}, 3, 10*time.Millisecond)

	if err != nil {
		l.reportError("compress_open", err)
		return
	}
	defer source.Close()

	// Use temporary file for crash consistency
	compressedName := filename + ".gz"
	tempName := compressedName + ".tmp"

	// Create temporary compressed file
	target, err := os.Create(tempName) // #nosec G304 -- tempName is internally generated, not user input
	if err != nil {
		l.reportError("compress_create", err)
		return
	}
	defer target.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(target)
	defer gzWriter.Close()

	// Copy data with compression
	_, err = io.Copy(gzWriter, source)
	if err != nil {
		// Clean up failed compression
		_ = target.Close()      // Ignore close error during cleanup
		_ = os.Remove(tempName) // Ignore remove error during cleanup
		l.reportError("compress_copy", err)
		return
	}

	// Close gzip writer to finalize compression
	err = gzWriter.Close()
	if err != nil {
		_ = target.Close()      // Ignore close error during cleanup
		_ = os.Remove(tempName) // Ignore remove error during cleanup
		l.reportError("compress_finalize", err)
		return
	}

	// Close target file
	err = target.Close()
	if err != nil {
		_ = os.Remove(tempName) // Ignore remove error during cleanup
		l.reportError("compress_close", err)
		return
	}

	// Atomically rename temporary file to final name
	// This ensures crash consistency - either compression is complete or it failed
	err = os.Rename(tempName, compressedName)
	if err != nil {
		_ = os.Remove(tempName) // Ignore remove error during cleanup
		l.reportError("compress_rename", fmt.Errorf("failed to rename %s to %s: %v", tempName, compressedName, err))
		return
	}

	// Remove original file only after successful compression and rename
	if err := os.Remove(filename); err != nil {
		l.reportError("compress_cleanup", err)
	}
}

// FileSystem interface for cross-platform abstraction
type FileSystem interface {
	Create(name string) (*os.File, error)
	Open(name string) (*os.File, error)
	Rename(oldname, newname string) error
	Remove(name string) error
	Stat(name string) (os.FileInfo, error)
}

// DefaultFileSystem implements FileSystem using standard os package
type DefaultFileSystem struct{}

func (fs DefaultFileSystem) Create(name string) (*os.File, error) {
	return os.Create(name) // #nosec G304 -- name is controlled by application via filesystem interface
}

func (fs DefaultFileSystem) Open(name string) (*os.File, error) {
	return os.Open(name) // #nosec G304 -- name is controlled by application via filesystem interface
}

func (fs DefaultFileSystem) Rename(oldname, newname string) error {
	return os.Rename(oldname, newname)
}

func (fs DefaultFileSystem) Remove(name string) error {
	return os.Remove(name)
}

func (fs DefaultFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

// BackgroundTask represents a task for the worker pool
type BackgroundTask struct {
	TaskType string // "cleanup", "compress", or "checksum"
	FilePath string
	Logger   *Logger
}

// BackgroundWorkers manages a pool of workers for background operations
type BackgroundWorkers struct {
	ctx         context.Context
	cancel      context.CancelFunc
	taskQueue   chan BackgroundTask
	wg          sync.WaitGroup
	workers     int
	activeTasks atomic.Int64 // Track active tasks for synchronization
	stopOnce    sync.Once    // Ensure stop is called only once
}

// newBackgroundWorkers creates a new worker pool
func newBackgroundWorkers(numWorkers int) *BackgroundWorkers {
	ctx, cancel := context.WithCancel(context.Background())

	bg := &BackgroundWorkers{
		ctx:       ctx,
		cancel:    cancel,
		taskQueue: make(chan BackgroundTask, 100), // Buffered channel
		workers:   numWorkers,
	}

	// Start workers
	for i := 0; i < numWorkers; i++ {
		bg.wg.Add(1)
		go bg.worker()
	}

	return bg
}

// worker processes background tasks
func (bg *BackgroundWorkers) worker() {
	defer bg.wg.Done()

	for {
		select {
		case <-bg.ctx.Done():
			return
		case task := <-bg.taskQueue:
			bg.processTask(task)
		}
	}
}

// processTask executes a background task
func (bg *BackgroundWorkers) processTask(task BackgroundTask) {
	// Increment active task counter
	bg.activeTasks.Add(1)
	defer bg.activeTasks.Add(-1)

	switch task.TaskType {
	case "cleanup":
		task.Logger.cleanupOldFiles()
	case "compress":
		task.Logger.compressFile(task.FilePath)
	case "checksum":
		task.Logger.generateChecksum(task.FilePath)
	}
}

// stop gracefully shuts down the worker pool
func (bg *BackgroundWorkers) stop() {
	bg.stopOnce.Do(func() {
		bg.cancel()
		close(bg.taskQueue)
		bg.wg.Wait()
	})
}

// waitForCompletion waits for all active tasks to complete
func (bg *BackgroundWorkers) waitForCompletion() {
	// Wait until no active tasks remain
	for bg.activeTasks.Load() > 0 {
		time.Sleep(1 * time.Millisecond)
	}
}

// generateChecksum creates a SHA-256 checksum sidecar file for the given file
// Called in background worker pool for rotated files
func (l *Logger) generateChecksum(filename string) {
	// Check if the file exists
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		// File might have been compressed - try .gz version
		if !strings.HasSuffix(filename, ".gz") {
			gzFilename := filename + ".gz"
			if _, err := os.Stat(gzFilename); err == nil {
				filename = gzFilename
			} else {
				l.reportError("checksum_missing", fmt.Errorf("file not found for checksum: %s", filename))
				return
			}
		} else {
			l.reportError("checksum_missing", fmt.Errorf("file not found for checksum: %s", filename))
			return
		}
	} else if err != nil {
		l.reportError("checksum_stat", fmt.Errorf("failed to stat file for checksum %s: %v", filename, err))
		return
	}

	// Open the file
	file, err := os.Open(filename) // #nosec G304 -- filename is internal backup file path, not user input
	if err != nil {
		l.reportError("checksum_open", fmt.Errorf("failed to open file for checksum %s: %v", filename, err))
		return
	}
	defer file.Close()

	// Calculate SHA-256 hash
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		l.reportError("checksum_read", fmt.Errorf("failed to read file for checksum %s: %v", filename, err))
		return
	}

	// Generate hex string
	hashHex := fmt.Sprintf("%x", hash.Sum(nil))

	// Create checksum sidecar file
	checksumFile := filename + ".sha256"
	content := fmt.Sprintf("%s  %s\n", hashHex, filepath.Base(filename))

	err = os.WriteFile(checksumFile, []byte(content), 0600) // More secure permissions
	if err != nil {
		l.reportError("checksum_write", fmt.Errorf("failed to write checksum file %s: %v", checksumFile, err))
		return
	}
}
