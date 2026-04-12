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

	timecache "github.com/agilira/go-timecache"
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

	// Cleanup orphan .tmp files from interrupted rotations (crash recovery)
	l.cleanupOrphanTmpFiles(filepath.Dir(sanitizedPath))

	file, err := l.openLogFile(sanitizedPath, fileMode, retryCount, retryDelay)
	if err != nil {
		return err
	}

	return l.initFileState(file, sanitizedPath)
}

// initSizeConfig initializes the size configuration with backward compatibility.
// Thread-safe: uses atomic.Int64 for maxSizeBytes.
// Idempotent: returns immediately if already initialized.
func (l *Logger) initSizeConfig() {
	if l.maxSizeBytes.Load() != 0 {
		return
	}

	if l.MaxSizeStr != "" {
		// Use new string-based configuration
		if size, err := ParseSize(l.MaxSizeStr); err == nil {
			l.maxSizeBytes.Store(size)
		} else {
			l.reportError("size_parse", fmt.Errorf("invalid MaxSizeStr %q: %v", l.MaxSizeStr, err))
		}
	} else if l.MaxSize > 0 {
		// Fallback to legacy MB-based configuration
		l.maxSizeBytes.Store(l.MaxSize * 1024 * 1024) // MB to bytes
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

// cleanupOrphanTmpFiles removes orphan .tmp files left from interrupted rotations.
// This provides crash recovery - if the process died mid-rotation, .tmp files
// may be left behind. We clean them up on startup to prevent disk space leaks.
func (l *Logger) cleanupOrphanTmpFiles(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Not critical - just log and continue
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".tmp") {
			tmpPath := filepath.Join(dir, name)
			// Only remove if file is old enough (at least 1 minute)
			// to avoid removing files from concurrent rotations
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if time.Since(info.ModTime()) > time.Minute {
				// WHY: Best-effort cleanup of orphan temp file; error is
				// intentionally not acted upon to avoid masking the original error.
				_ = os.Remove(tmpPath)
			}
		}
	}
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

	// WHY capture before closeAndRotateFile: bytesWritten is reset in
	// updateRotationState(), so we must snapshot it here for the
	// RotationEvent. This gives downstream consumers the exact size
	// of the sealed segment for anomaly detection (flood attacks).
	sealedBytes := l.bytesWritten.Load()

	if err := l.closeAndRotateFile(currentFile, backupName, retryCount, retryDelay, fileMode); err != nil {
		return err
	}

	l.updateRotationState()

	// Invoke OnRotate callback before scheduling background tasks.
	// WHY before: the callback must fire while the rotation is still
	// synchronous so that blackbox can record the event before
	// compression/cleanup may alter the sealed file.
	if l.OnRotate != nil {
		l.safeInvokeOnRotate(RotationEvent{
			Timestamp:    timecache.CachedTime(),
			PreviousFile: backupName,
			NewFile:      l.Filename,
			Sequence:     l.rotationSeq.Load(),
			BytesWritten: sealedBytes,
		})
	}

	l.scheduleBackgroundTasks(backupName)

	return nil
}

// safeInvokeOnRotate calls the OnRotate callback with panic recovery.
// WHY: the callback is user-provided code running in the rotation path.
// A panic here would leave the rotation flag set and block all future
// rotations (denial of service). We recover and report via ErrorCallback.
func (l *Logger) safeInvokeOnRotate(event RotationEvent) {
	defer func() {
		if r := recover(); r != nil {
			l.reportError("on_rotate_panic", fmt.Errorf("OnRotate callback panicked: %v", r))
		}
	}()
	l.OnRotate(event)
}

// generateBackupName creates a timestamped backup filename
func (l *Logger) generateBackupName() string {
	// WHY: Both writeSync and generateBackupName go through timeCacheOnce.Do
	// so that all reads of l.timeCache are synchronized through the same
	// sync.Once memory ordering guarantee. Direct reads without the Once
	// would race with the initialization write in writeSync (DATA RACE).
	l.timeCacheOnce.Do(func() {
		l.timeCache = timecache.NewWithResolution(time.Millisecond)
	})
	now := l.timeCache.CachedTime()
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
	var sourceCloseOnce sync.Once
	defer func() {
		sourceCloseOnce.Do(func() {
			if closeErr := source.Close(); closeErr != nil {
				// Only report if it's not "file already closed"
				if !isFileAlreadyClosedError(closeErr) {
					l.reportError("compress_source_close", closeErr)
				}
			}
		})
	}()

	// Use temporary file for crash consistency
	compressedName := filename + ".gz"
	tempName := compressedName + ".tmp"

	// Create temporary compressed file
	target, err := os.Create(tempName) // #nosec G304 -- tempName is internally generated, not user input
	if err != nil {
		l.reportError("compress_create", err)
		return
	}
	var targetCloseOnce sync.Once
	defer func() {
		targetCloseOnce.Do(func() {
			if closeErr := target.Close(); closeErr != nil {
				// Only report if it's not "file already closed"
				if !isFileAlreadyClosedError(closeErr) {
					l.reportError("compress_target_close", closeErr)
				}
			}
		})
	}()

	// Create gzip writer
	gzWriter := gzip.NewWriter(target)
	var gzCloseOnce sync.Once
	defer func() {
		gzCloseOnce.Do(func() {
			if closeErr := gzWriter.Close(); closeErr != nil {
				// Only report if it's not "file already closed"
				if !isFileAlreadyClosedError(closeErr) {
					l.reportError("compress_gzip_close", closeErr)
				}
			}
		})
	}()

	// Copy data with compression
	_, err = io.Copy(gzWriter, source)
	if err != nil {
		// Clean up failed compression - use sync.Once to avoid duplicate closes
		gzCloseOnce.Do(func() { _ = gzWriter.Close() })
		targetCloseOnce.Do(func() { _ = target.Close() })
		_ = os.Remove(tempName) // Ignore remove error during cleanup
		l.reportError("compress_copy", err)
		return
	}

	// Close gzip writer to finalize compression
	var finalizeErr error
	gzCloseOnce.Do(func() {
		finalizeErr = gzWriter.Close()
	})
	if finalizeErr != nil {
		_ = os.Remove(tempName) // Ignore remove error during cleanup
		l.reportError("compress_finalize", finalizeErr)
		return
	}

	// Close target file
	var closeErr error
	targetCloseOnce.Do(func() {
		closeErr = target.Close()
	})
	if closeErr != nil {
		_ = os.Remove(tempName) // Ignore remove error during cleanup
		l.reportError("compress_close", closeErr)
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

	// Condition variable for efficient waitForCompletion
	taskCond *sync.Cond
	condMu   sync.Mutex
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
	bg.taskCond = sync.NewCond(&bg.condMu)

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
	defer func() {
		bg.activeTasks.Add(-1)
		// Signal any waiters that a task completed
		bg.taskCond.Broadcast()
	}()

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
// Uses condition variable instead of busy-wait polling
func (bg *BackgroundWorkers) waitForCompletion() {
	bg.condMu.Lock()
	defer bg.condMu.Unlock()

	for bg.activeTasks.Load() > 0 {
		bg.taskCond.Wait()
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
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// Only report if it's not "file already closed"
			if !isFileAlreadyClosedError(closeErr) {
				l.reportError("checksum_file_close", closeErr)
			}
		}
	}()

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

// isFileAlreadyClosedError checks if the error indicates the file is already closed
func isFileAlreadyClosedError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "file already closed") ||
		strings.Contains(errStr, "use of closed file") ||
		strings.Contains(errStr, "bad file descriptor")
}
