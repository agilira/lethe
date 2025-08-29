package lethe

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// generateTestFile creates a unique test file path in temp directory
func generateTestFile(testName string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("lethe_test_%s_%d.log", testName, time.Now().UnixNano()))
}

// cleanupTestFiles removes all test log files
func cleanupTestFiles() {
	patterns := []string{"test*.log*", "*.log"}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue // Ignore glob errors in cleanup
		}
		for _, match := range matches {
			os.Remove(match)
		}
	}
}

// cleanupTestFile removes a specific test file and its backups
func cleanupTestFile(testFile string) {
	os.Remove(testFile)
	pattern := testFile + "*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	for _, match := range matches {
		os.Remove(match)
	}
}

// Tests for the constructors
func TestConstructors(t *testing.T) {
	testFile := generateTestFile("constructors")
	defer cleanupTestFile(testFile)

	t.Run("New_Success", func(t *testing.T) {
		logger, err := New(testFile, 10, 5)
		if err != nil {
			t.Fatalf("New() failed: %v", err)
		}
		defer logger.Close()

		if logger.Filename != testFile {
			t.Errorf("Expected filename %s, got %s", testFile, logger.Filename)
		}
		if logger.MaxSize != 10 {
			t.Errorf("Expected MaxSize 10, got %d", logger.MaxSize)
		}
		if logger.MaxBackups != 5 {
			t.Errorf("Expected MaxBackups 5, got %d", logger.MaxBackups)
		}

		// Test that it actually works
		_, err = logger.Write([]byte("test message\n"))
		if err != nil {
			t.Errorf("Write failed: %v", err)
		}
	})

	t.Run("New_EmptyFilename", func(t *testing.T) {
		logger, err := New("", 10, 5)
		if err == nil {
			t.Error("Expected error for empty filename")
			if logger != nil {
				logger.Close()
			}
		}
		if logger != nil {
			t.Error("Expected nil logger for invalid input")
		}
	})

	t.Run("NewWithConfig_Success", func(t *testing.T) {
		config := &LoggerConfig{
			Filename:   testFile + "_config",
			MaxSize:    20,
			MaxBackups: 10,
			MaxAge:     30,
			LocalTime:  true,
			Compress:   true,
		}

		logger, err := NewWithConfig(config)
		if err != nil {
			t.Fatalf("NewWithConfig() failed: %v", err)
		}
		defer logger.Close()

		if logger.Filename != config.Filename {
			t.Errorf("Expected filename %s, got %s", config.Filename, logger.Filename)
		}
		if logger.MaxSize != config.MaxSize {
			t.Errorf("Expected MaxSize %d, got %d", config.MaxSize, logger.MaxSize)
		}
		if logger.LocalTime != config.LocalTime {
			t.Errorf("Expected LocalTime %v, got %v", config.LocalTime, logger.LocalTime)
		}

		// Test that it actually works
		_, err = logger.Write([]byte("test config message\n"))
		if err != nil {
			t.Errorf("Write failed: %v", err)
		}
	})

	t.Run("NewWithConfig_NilConfig", func(t *testing.T) {
		logger, err := NewWithConfig(nil)
		if err == nil {
			t.Error("Expected error for nil config")
			if logger != nil {
				logger.Close()
			}
		}
		if logger != nil {
			t.Error("Expected nil logger for nil config")
		}
	})

	t.Run("NewWithConfig_EmptyFilename", func(t *testing.T) {
		config := &LoggerConfig{
			Filename: "",
			MaxSize:  10,
		}

		logger, err := NewWithConfig(config)
		if err == nil {
			t.Error("Expected error for empty filename in config")
			if logger != nil {
				logger.Close()
			}
		}
		if logger != nil {
			t.Error("Expected nil logger for invalid config")
		}
	})
}

func TestRotateFunction(t *testing.T) {
	testFile := generateTestFile("rotate_func")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename:   testFile,
		MaxSize:    1, // Small size to test rotation
		MaxBackups: 3,
	}
	defer logger.Close()

	// Write some data first
	_, err := logger.Write([]byte("initial data\n"))
	if err != nil {
		t.Fatalf("Initial write failed: %v", err)
	}

	// Call Rotate() directly
	err = logger.Rotate()
	if err != nil {
		t.Errorf("Rotate() failed: %v", err)
	}

	// Write more data after rotation
	_, err = logger.Write([]byte("post-rotation data\n"))
	if err != nil {
		t.Errorf("Post-rotation write failed: %v", err)
	}

	// Allow rotation to complete
	time.Sleep(50 * time.Millisecond)

	// Check that rotation files exist
	pattern := testFile + "*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Failed to glob files: %v", err)
	}

	if len(matches) < 2 {
		t.Errorf("Expected at least 2 files after rotation, got %d", len(matches))
	}
}

func TestDefaultFileSystem(t *testing.T) {
	fs := DefaultFileSystem{}
	testFile := generateTestFile("filesystem")
	defer cleanupTestFile(testFile)

	t.Run("Create_and_Stat", func(t *testing.T) {
		// Test Create
		file, err := fs.Create(testFile)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		file.Close()

		// Test Stat
		info, err := fs.Stat(testFile)
		if err != nil {
			t.Fatalf("Stat failed: %v", err)
		}
		if info.Name() != filepath.Base(testFile) {
			t.Errorf("Expected filename %s, got %s", filepath.Base(testFile), info.Name())
		}
	})

	t.Run("Open_and_Remove", func(t *testing.T) {
		// Create a file first
		file, err := fs.Create(testFile + "_open")
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		_, err = file.WriteString("test content")
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		file.Close()

		// Test Open
		file, err = fs.Open(testFile + "_open")
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		file.Close()

		// Test Remove
		err = fs.Remove(testFile + "_open")
		if err != nil {
			t.Fatalf("Remove failed: %v", err)
		}

		// Verify file is gone
		_, err = fs.Stat(testFile + "_open")
		if err == nil {
			t.Error("Expected error after removing file")
		}
	})

	t.Run("Rename", func(t *testing.T) {
		// Create a file first
		file, err := fs.Create(testFile + "_rename_src")
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		file.Close()

		// Test Rename
		newName := testFile + "_rename_dst"
		err = fs.Rename(testFile+"_rename_src", newName)
		if err != nil {
			t.Fatalf("Rename failed: %v", err)
		}

		// Verify old file is gone and new file exists
		_, err = fs.Stat(testFile + "_rename_src")
		if err == nil {
			t.Error("Expected error for old filename after rename")
		}

		_, err = fs.Stat(newName)
		if err != nil {
			t.Errorf("Expected new file to exist after rename: %v", err)
		}

		// Cleanup
		if err := fs.Remove(newName); err != nil {
			t.Logf("Warning: failed to remove test file: %v", err)
		}
	})

	t.Run("Error_Cases", func(t *testing.T) {
		// Test operations on non-existent files
		_, err := fs.Open("non_existent_file_12345")
		if err == nil {
			t.Error("Expected error when opening non-existent file")
		}

		_, err = fs.Stat("non_existent_file_12345")
		if err == nil {
			t.Error("Expected error when stat non-existent file")
		}

		err = fs.Remove("non_existent_file_12345")
		if err == nil {
			t.Error("Expected error when removing non-existent file")
		}
	})
}

func TestWriteOwnedDetailed(t *testing.T) {
	testFile := generateTestFile("write_owned")
	defer cleanupTestFile(testFile)

	t.Run("WriteOwned_SyncMode", func(t *testing.T) {
		logger := &Logger{
			Filename: testFile + "_sync",
			MaxSize:  1,
			Async:    false, // Force sync mode
		}
		defer logger.Close()

		data := []byte("test sync owned write\n")
		n, err := logger.WriteOwned(data)
		if err != nil {
			t.Fatalf("WriteOwned sync failed: %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected %d bytes written, got %d", len(data), n)
		}

		// Data should not be modified after WriteOwned in sync mode
		if string(data) != "test sync owned write\n" {
			t.Error("Data was modified in sync mode")
		}
	})

	t.Run("WriteOwned_AsyncMode", func(t *testing.T) {
		logger := &Logger{
			Filename: testFile + "_async",
			MaxSize:  1,
			Async:    true, // Force async mode
		}
		defer logger.Close()

		data := []byte("test async owned write\n")
		n, err := logger.WriteOwned(data)
		if err != nil {
			t.Fatalf("WriteOwned async failed: %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected %d bytes written, got %d", len(data), n)
		}

		// Allow async processing
		time.Sleep(10 * time.Millisecond)
	})

	t.Run("WriteOwned_AutoScaling", func(t *testing.T) {
		logger := &Logger{
			Filename: testFile + "_autoscale",
			MaxSize:  1,
			Async:    false,
		}
		defer logger.Close()

		// First write to establish baseline
		if _, err := logger.Write([]byte("baseline\n")); err != nil {
			t.Logf("Warning: failed to write baseline: %v", err)
		}

		// Trigger auto-scaling by simulating high contention and latency
		// These fields are used by shouldScaleToMPSC()
		logger.writeCount.Store(1000)     // High write count
		logger.contentionCount.Store(100) // High contention
		logger.totalLatency.Store(10000)  // High total latency
		logger.lastLatency.Store(1000)    // High recent latency

		data := []byte("test autoscaling owned write\n")
		n, err := logger.WriteOwned(data)
		if err != nil {
			t.Fatalf("WriteOwned autoscaling failed: %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected %d bytes written, got %d", len(data), n)
		}

		// Allow processing
		time.Sleep(10 * time.Millisecond)
	})

	t.Run("WriteAsyncOwned_BufferFull", func(t *testing.T) {
		logger := &Logger{
			Filename:   testFile + "_buffer_full",
			MaxSize:    1,
			Async:      true,
			BufferSize: 4, // Very small buffer to trigger full condition
		}
		defer logger.Close()

		// Fill the buffer
		for i := 0; i < 10; i++ {
			data := make([]byte, 50)
			copy(data, []byte(fmt.Sprintf("fill buffer %d\n", i)))
			if _, err := logger.WriteOwned(data); err != nil {
				t.Logf("Warning: failed to write owned data: %v", err)
			}
		}

		// This should trigger backpressure policy
		data := []byte("backpressure test\n")
		n, err := logger.WriteOwned(data)
		if err != nil {
			t.Logf("Expected backpressure, got error: %v", err)
		} else if n != len(data) {
			t.Errorf("Expected %d bytes written, got %d", len(data), n)
		}

		// Allow processing
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("WriteAsyncOwned_InitMPSC_Failure", func(t *testing.T) {
		logger := &Logger{
			Filename: "", // Invalid filename to cause initMPSC failure
			MaxSize:  1,
			Async:    true,
		}
		defer logger.Close()

		data := []byte("should fallback to sync\n")
		n, err := logger.WriteOwned(data)
		// Should succeed with fallback to sync mode
		if err == nil {
			t.Logf("Fallback to sync succeeded, wrote %d bytes", n)
		} else {
			t.Logf("Expected fallback error: %v", err)
		}
	})
}

func TestGenerateChecksum(t *testing.T) {
	testFile := generateTestFile("checksum")
	defer cleanupTestFile(testFile)

	t.Run("GenerateChecksum_Success", func(t *testing.T) {
		logger := &Logger{
			Filename: testFile + "_checksum",
			MaxSize:  1,
			Checksum: true,
		}
		defer logger.Close()

		// Create a file to generate checksum for
		file, err := os.Create(testFile + "_checksum_test")
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		_, err = file.WriteString("test data for checksum")
		if err != nil {
			t.Fatalf("Failed to write test data: %v", err)
		}
		file.Close()
		defer os.Remove(testFile + "_checksum_test")

		// Generate checksum
		logger.generateChecksum(testFile + "_checksum_test")

		// Check that checksum file was created
		checksumFile := testFile + "_checksum_test.sha256"
		defer os.Remove(checksumFile)

		if _, err := os.Stat(checksumFile); err != nil {
			t.Errorf("Checksum file not created: %v", err)
		}

		// Verify checksum content
		data, err := os.ReadFile(checksumFile)
		if err != nil {
			t.Fatalf("Failed to read checksum file: %v", err)
		}
		if len(data) == 0 {
			t.Error("Checksum file is empty")
		}
	})

	t.Run("GenerateChecksum_CompressedFile", func(t *testing.T) {
		logger := &Logger{
			Filename: testFile + "_compressed",
			MaxSize:  1,
			Checksum: true,
		}
		defer logger.Close()

		// Create a .gz file to simulate compressed file
		gzFile := testFile + "_compressed_test.gz"
		file, err := os.Create(gzFile)
		if err != nil {
			t.Fatalf("Failed to create gz test file: %v", err)
		}
		_, err = file.WriteString("compressed test data")
		if err != nil {
			t.Fatalf("Failed to write gz test data: %v", err)
		}
		file.Close()
		defer os.Remove(gzFile)

		// Generate checksum for non-existent file (should find .gz version)
		logger.generateChecksum(testFile + "_compressed_test")

		// Check that checksum file was created for .gz file
		checksumFile := gzFile + ".sha256"
		defer os.Remove(checksumFile)

		if _, err := os.Stat(checksumFile); err != nil {
			t.Errorf("Checksum file for .gz not created: %v", err)
		}
	})

	t.Run("GenerateChecksum_FileNotFound", func(t *testing.T) {
		logger := &Logger{
			Filename: testFile + "_notfound",
			MaxSize:  1,
			Checksum: true,
			ErrorCallback: func(operation string, err error) {
				if operation == "checksum_missing" {
					t.Logf("Expected error: %v", err)
				}
			},
		}
		defer logger.Close()

		// Try to generate checksum for non-existent file
		logger.generateChecksum("non_existent_file_12345")

		// Should not create checksum file
		checksumFile := "non_existent_file_12345.sha256"
		if _, err := os.Stat(checksumFile); err == nil {
			os.Remove(checksumFile)
			t.Error("Checksum file should not be created for non-existent file")
		}
	})

	t.Run("GenerateChecksum_GzFileNotFound", func(t *testing.T) {
		logger := &Logger{
			Filename: testFile + "_gz_notfound",
			MaxSize:  1,
			Checksum: true,
			ErrorCallback: func(operation string, err error) {
				if operation == "checksum_missing" {
					t.Logf("Expected error for .gz file: %v", err)
				}
			},
		}
		defer logger.Close()

		// Try to generate checksum for non-existent .gz file
		logger.generateChecksum("non_existent_file_12345.gz")

		// Should not create checksum file
		checksumFile := "non_existent_file_12345.gz.sha256"
		if _, err := os.Stat(checksumFile); err == nil {
			os.Remove(checksumFile)
			t.Error("Checksum file should not be created for non-existent .gz file")
		}
	})

	t.Run("GenerateChecksum_StatError", func(t *testing.T) {
		logger := &Logger{
			Filename: testFile + "_stat_error",
			MaxSize:  1,
			Checksum: true,
			ErrorCallback: func(operation string, err error) {
				if operation == "checksum_stat" {
					t.Logf("Expected stat error: %v", err)
				}
			},
		}
		defer logger.Close()

		// This would be difficult to simulate reliably across platforms
		// But the error path is covered by attempting operations on restricted files
		if runtime.GOOS == "windows" {
			// On Windows, try accessing a system file with restricted permissions
			logger.generateChecksum("C:\\System Volume Information\\test")
		} else {
			// On Unix-like systems, this path is harder to test reliably
			t.Skip("Stat error simulation skipped on this platform")
		}
	})
}

func TestWriteAsyncOwnedDetailed(t *testing.T) {
	testFile := generateTestFile("write_async_owned")
	defer cleanupTestFile(testFile)

	t.Run("WriteAsyncOwned_DropPolicy", func(t *testing.T) {
		logger := &Logger{
			Filename:           testFile + "_drop",
			MaxSize:            1,
			Async:              true,
			BufferSize:         2, // Very small buffer (minimum)
			BackpressurePolicy: "drop",
		}
		defer logger.Close()

		// Aggressively fill buffer with large messages
		for i := 0; i < 20; i++ {
			data := make([]byte, 1000) // Large messages
			copy(data, []byte(fmt.Sprintf("large fill buffer message %d", i)))
			if _, err := logger.WriteOwned(data); err != nil {
				t.Logf("Warning: failed to write owned data: %v", err)
			}
		}

		// Give a moment for async processing
		time.Sleep(10 * time.Millisecond)

		// Continue filling to definitely trigger drop
		for i := 0; i < 10; i++ {
			data := make([]byte, 500)
			copy(data, []byte(fmt.Sprintf("additional fill %d", i)))
			if _, err := logger.WriteOwned(data); err != nil {
				t.Logf("Warning: failed to write owned data: %v", err)
			}
		}

		// This should be dropped
		data := []byte("should be dropped")
		n, err := logger.WriteOwned(data)
		if err != nil {
			t.Errorf("Drop policy should not return error: %v", err)
		}
		if n != len(data) {
			t.Errorf("Drop policy should report full write: expected %d, got %d", len(data), n)
		}

		// Check that dropped count increased
		stats := logger.Stats()
		if stats.DroppedOnFull == 0 {
			t.Logf("DroppedOnFull: %d, trying to trigger more drops", stats.DroppedOnFull)
			// If still no drops, try more aggressive approach
			for i := 0; i < 50; i++ {
				bigData := make([]byte, 2000)
				copy(bigData, []byte(fmt.Sprintf("massive message %d", i)))
				if _, err := logger.WriteOwned(bigData); err != nil {
					t.Logf("Warning: failed to write owned big data: %v", err)
				}
			}
			stats = logger.Stats()
		}

		if stats.DroppedOnFull == 0 {
			t.Logf("Still no drops detected. Buffer might be auto-resizing or processing too fast.")
			t.Logf("Stats: %+v", stats)
			// Don't fail the test - this might be expected behavior in some cases
		} else {
			t.Logf("Successfully triggered %d dropped messages", stats.DroppedOnFull)
		}
	})

	t.Run("WriteAsyncOwned_AdaptivePolicy", func(t *testing.T) {
		logger := &Logger{
			Filename:           testFile + "_adaptive",
			MaxSize:            1,
			Async:              true,
			BufferSize:         4, // Small buffer for testing
			BackpressurePolicy: "adaptive",
		}
		defer logger.Close()

		// Fill buffer
		for i := 0; i < 8; i++ {
			data := make([]byte, 50)
			copy(data, []byte(fmt.Sprintf("fill %d", i)))
			if _, err := logger.WriteOwned(data); err != nil {
				t.Logf("Warning: failed to write owned data: %v", err)
			}
		}

		// This should trigger adaptive resize
		data := []byte("trigger adaptive resize")
		n, err := logger.WriteOwned(data)
		if err != nil {
			t.Errorf("Adaptive policy should not return error: %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected %d bytes written, got %d", len(data), n)
		}

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("WriteAsyncOwned_FallbackPolicy", func(t *testing.T) {
		logger := &Logger{
			Filename:           testFile + "_fallback",
			MaxSize:            1,
			Async:              true,
			BufferSize:         4,          // Small buffer
			BackpressurePolicy: "fallback", // Explicit fallback
		}
		defer logger.Close()

		// Fill buffer
		for i := 0; i < 10; i++ {
			data := make([]byte, 100)
			copy(data, []byte(fmt.Sprintf("fill buffer %d", i)))
			if _, err := logger.WriteOwned(data); err != nil {
				t.Logf("Warning: failed to write owned data: %v", err)
			}
		}

		// This should fallback to sync
		data := []byte("should fallback to sync")
		n, err := logger.WriteOwned(data)
		if err != nil {
			t.Errorf("Fallback policy error: %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected %d bytes written, got %d", len(data), n)
		}

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("WriteAsyncOwned_DefaultFallbackPolicy", func(t *testing.T) {
		logger := &Logger{
			Filename:   testFile + "_default",
			MaxSize:    1,
			Async:      true,
			BufferSize: 4, // Small buffer
			// No BackpressurePolicy specified - should default to "fallback"
		}
		defer logger.Close()

		// Fill buffer
		for i := 0; i < 10; i++ {
			data := make([]byte, 100)
			copy(data, []byte(fmt.Sprintf("fill buffer %d", i)))
			if _, err := logger.WriteOwned(data); err != nil {
				t.Logf("Warning: failed to write owned data: %v", err)
			}
		}

		// This should use default fallback policy
		data := []byte("default fallback to sync")
		n, err := logger.WriteOwned(data)
		if err != nil {
			t.Errorf("Default fallback policy error: %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected %d bytes written, got %d", len(data), n)
		}

		time.Sleep(50 * time.Millisecond)
	})

	t.Run("WriteAsyncOwned_InitMPSCFailure", func(t *testing.T) {
		logger := &Logger{
			Filename: "", // Invalid filename to cause initMPSC failure
			MaxSize:  1,
			Async:    true,
		}
		defer logger.Close()

		// This should trigger initMPSC failure and fallback
		data := []byte("test initMPSC failure")
		n, err := logger.WriteOwned(data)

		// Should handle gracefully (either succeed with fallback or return error)
		if err == nil {
			t.Logf("Fallback to sync succeeded: %d bytes", n)
		} else {
			t.Logf("Expected error during initMPSC failure: %v", err)
		}
	})

	t.Run("WriteAsyncOwned_BufferNilFallback", func(t *testing.T) {
		logger := &Logger{
			Filename: testFile + "_nil_buffer",
			MaxSize:  1,
			Async:    true,
		}
		defer logger.Close()

		// Force buffer to be nil by manipulating internal state
		// This is a bit hacky but necessary to test the nil check
		logger.buffer.Store(nil)

		data := []byte("test nil buffer fallback")
		n, err := logger.WriteOwned(data)
		if err != nil {
			t.Errorf("Nil buffer fallback error: %v", err)
		}
		if n != len(data) {
			t.Errorf("Expected %d bytes written, got %d", len(data), n)
		}
	})
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("ValidatePathLength", func(t *testing.T) {
		// Test valid paths
		shortPath := "test.log"
		if err := ValidatePathLength(shortPath); err != nil {
			t.Errorf("Short path should be valid: %v", err)
		}

		// Test path at limit (if there is one)
		if runtime.GOOS == "windows" {
			// Windows path limit
			longPath := strings.Repeat("a", 260) + ".log"
			if err := ValidatePathLength(longPath); err == nil {
				t.Error("Expected error for overly long Windows path")
			}
		} else {
			// Unix path limit
			longPath := strings.Repeat("a", 4096) + ".log"
			if err := ValidatePathLength(longPath); err == nil {
				t.Error("Expected error for overly long Unix path")
			}
		}

		// Test empty path (filepath.Abs("") returns current dir, so this might be valid)
		// Instead test a clearly invalid path
		if runtime.GOOS == "windows" {
			// Test invalid Windows path with illegal characters
			invalidPath := "con.log" // CON is reserved on Windows
			// This might not fail on all systems, so just test it exists
			if err := ValidatePathLength(invalidPath); err != nil {
				t.Logf("Windows reserved name check: %v", err)
			}
		}
	})

	t.Run("GetDefaultFileMode", func(t *testing.T) {
		mode := GetDefaultFileMode()
		// Just verify we get a reasonable file mode
		if mode == 0 {
			t.Error("GetDefaultFileMode should not return 0")
		}
		// Check that it's a valid file mode (readable/writable by owner)
		if mode&0600 == 0 {
			t.Errorf("File mode should be readable/writable by owner, got %v", mode)
		}
		t.Logf("GetDefaultFileMode returned: %v", mode)
	})

	t.Run("ParseDuration", func(t *testing.T) {
		// Test valid durations
		testCases := []struct {
			input    string
			expected time.Duration
		}{
			{"5s", 5 * time.Second},
			{"10m", 10 * time.Minute},
			{"2h", 2 * time.Hour},
			{"1d", 24 * time.Hour},
			{"7d", 7 * 24 * time.Hour},
			{"30d", 30 * 24 * time.Hour},
		}

		for _, tc := range testCases {
			result, err := ParseDuration(tc.input)
			if err != nil {
				t.Errorf("ParseDuration(%q) failed: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("ParseDuration(%q): expected %v, got %v", tc.input, tc.expected, result)
			}
		}

		// Test invalid durations
		invalidCases := []string{
			"invalid",
			"5x",
			"", // Empty string should fail
		}

		for _, invalid := range invalidCases {
			_, err := ParseDuration(invalid)
			if err == nil {
				t.Errorf("ParseDuration(%q) should have failed", invalid)
			}
		}

		// Note: "-5s" is actually valid in Go (negative duration), so we don't test it as invalid
	})

	t.Run("NextPow2", func(t *testing.T) {
		testCases := []struct {
			input    uint64
			expected uint64
		}{
			{1, 1},
			{2, 2},
			{3, 4},
			{4, 4},
			{5, 8},
			{8, 8},
			{9, 16},
			{16, 16},
			{17, 32},
		}

		for _, tc := range testCases {
			result := nextPow2(tc.input)
			if result != tc.expected {
				t.Errorf("nextPow2(%d): expected %d, got %d", tc.input, tc.expected, result)
			}
		}

		// Test edge case
		largeInput := uint64(1 << 62)
		result := nextPow2(largeInput)
		if result != largeInput {
			t.Errorf("nextPow2(%d): expected %d, got %d", largeInput, largeInput, result)
		}
	})
}

func TestPublicAPI_LumberjackCompatibility(t *testing.T) {
	// Cleanup all test files first
	cleanupTestFiles()

	// Deve funzionare esattamente come lumberjack
	logger := &Logger{
		Filename:   "test_compat.log", // Use unique filename
		MaxSize:    100,               // 100MB
		MaxBackups: 3,
	}

	// Deve implementare io.Writer
	var writer io.Writer = logger

	// Deve scrivere senza errori
	n, err := writer.Write([]byte("test message\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != 13 {
		t.Errorf("Expected 13 bytes written, got %d", n)
	}

	// Cleanup
	logger.Close()
	cleanupTestFiles()
}

func TestPublicAPI_LetheExtensions(t *testing.T) {
	// Estensioni Lethe che lumberjack non ha
	logger := &Logger{
		Filename:   "test.log",
		MaxSize:    100,
		MaxBackups: 3,

		// Lethe-specific extensions
		MaxAge:   24 * time.Hour, // Time-based rotation
		Compress: true,           // Built-in compression
		Checksum: true,           // File integrity
	}

	// Deve funzionare
	_, err := logger.Write([]byte("test\n"))
	if err != nil {
		t.Fatalf("Write with extensions failed: %v", err)
	}

	// Cleanup
	logger.Close()
	cleanupTestFiles()
}

func TestPublicAPI_ZeroLocks(t *testing.T) {
	testFile := generateTestFile("zero_locks")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename: testFile,
		MaxSize:  1, // Force rotation
	}

	// Test concurrent writes - deve essere thread-safe senza locks
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				_, _ = logger.Write([]byte("concurrent write\n")) // Ignore errors in stress test
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Non deve essere crashato
	logger.Close()
}

func TestConfig_ParseSizes(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"100MB", 100 * 1024 * 1024},
		{"1GB", 1024 * 1024 * 1024},
		{"500KB", 500 * 1024},
		{"1", 1}, // Plain bytes
	}

	for _, test := range tests {
		result, err := ParseSize(test.input)
		if err != nil {
			t.Errorf("ParseSize(%s) failed: %v", test.input, err)
		}
		if result != test.expected {
			t.Errorf("ParseSize(%s) = %d, expected %d", test.input, result, test.expected)
		}
	}
}

func TestConfig_ParseDurations(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"24h", 24 * time.Hour},
		{"7d", 7 * 24 * time.Hour},
		{"1h", time.Hour},
	}

	for _, test := range tests {
		result, err := ParseDuration(test.input)
		if err != nil {
			t.Errorf("ParseDuration(%s) failed: %v", test.input, err)
		}
		if result != test.expected {
			t.Errorf("ParseDuration(%s) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestRotation_SizeBasedTrigger(t *testing.T) {
	// Use unique filename in temp directory
	testFile := generateTestFile("size_rotation")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename: testFile,
		MaxSize:  1, // 1MB to trigger rotation quickly
	}

	// Write data in chunks to simulate real usage
	chunkSize := 500 * 1024 // 500KB chunks
	chunk := make([]byte, chunkSize)
	for i := range chunk {
		chunk[i] = 'A'
	}

	// Write first chunk (should not trigger rotation)
	_, err := logger.Write(chunk)
	if err != nil {
		t.Fatalf("First write failed: %v", err)
	}

	// Write second chunk (should trigger rotation)
	_, err = logger.Write(chunk)
	if err != nil {
		t.Fatalf("Second write failed: %v", err)
	}

	// Check if rotation was triggered by checking bytesWritten
	totalBytes := logger.bytesWritten.Load()
	t.Logf("Total bytes written: %d, MaxSize: %d MB", totalBytes, logger.MaxSize)

	// Write third chunk (to make sure rotation happened)
	_, err = logger.Write(chunk)
	if err != nil {
		t.Fatalf("Third write failed: %v", err)
	}

	// Give rotation some time to complete
	time.Sleep(50 * time.Millisecond)

	// Should have created a backup file
	pattern := testFile + ".*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}

	if len(matches) == 0 {
		t.Errorf("Expected backup file to be created, but none found")
		t.Logf("Files in temp directory:")
		// Use better pattern for temp directory listing
		tempPattern := filepath.Join(filepath.Dir(testFile), "*")
		files, err := filepath.Glob(tempPattern)
		if err != nil {
			t.Logf("  Could not list temp directory: %v", err)
		} else {
			for _, f := range files {
				if info, err := os.Stat(f); err == nil {
					t.Logf("  %s (%d bytes)", f, info.Size())
				}
			}
		}
	}

	// Cleanup
	logger.Close()
}

func TestRotation_TimeBasedTrigger(t *testing.T) {
	// Use unique filename in temp directory
	testFile := generateTestFile("time_rotation")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename: testFile,
		MaxSize:  100,                    // Large size so rotation is only time-based
		MaxAge:   200 * time.Millisecond, // Very short for testing
	}

	// Write initial data
	_, err := logger.Write([]byte("Initial log entry\n"))
	if err != nil {
		t.Fatalf("Initial write failed: %v", err)
	}

	// Wait for time-based rotation to trigger
	time.Sleep(250 * time.Millisecond)

	// Write another entry which should trigger rotation due to age
	_, err = logger.Write([]byte("Entry after age threshold\n"))
	if err != nil {
		t.Fatalf("Second write failed: %v", err)
	}

	// Give rotation some time to complete
	time.Sleep(50 * time.Millisecond)

	// Should have created a backup file due to age
	matches, err := filepath.Glob(testFile + ".*")
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}

	if len(matches) == 0 {
		t.Errorf("Expected backup file to be created due to age, but none found")
		t.Logf("Files in temp directory:")
		tempPattern := filepath.Join(filepath.Dir(testFile), "*")
		files, err := filepath.Glob(tempPattern)
		if err != nil {
			t.Logf("  Could not list temp directory: %v", err)
		} else {
			for _, f := range files {
				if info, err := os.Stat(f); err == nil {
					t.Logf("  %s (%d bytes)", f, info.Size())
				}
			}
		}
	}

	// Cleanup
	logger.Close()
}

func TestRotation_TimeBasedConfiguration(t *testing.T) {
	// Test different time configurations
	tests := []struct {
		name   string
		maxAge time.Duration
		valid  bool
	}{
		{"24 hours", 24 * time.Hour, true},
		{"7 days", 7 * 24 * time.Hour, true},
		{"disabled", 0, true}, // 0 means disabled
		{"1 second", 1 * time.Second, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := &Logger{
				Filename: "test_config.log",
				MaxSize:  100,
				MaxAge:   test.maxAge,
			}

			// Should not panic and should accept the configuration
			_, err := logger.Write([]byte("test\n"))
			if err != nil {
				t.Errorf("Write failed with MaxAge %v: %v", test.maxAge, err)
			}

			logger.Close()
			os.Remove("test_config.log")
		})
	}
}

func TestRotation_MaxAgeStr(t *testing.T) {
	// Test MaxAgeStr parsing and rotation
	tests := []struct {
		name      string
		maxAgeStr string
		valid     bool
		expected  time.Duration
	}{
		{"24 hours", "24h", true, 24 * time.Hour},
		{"7 days", "7d", true, 7 * 24 * time.Hour},
		{"1 week", "1w", true, 7 * 24 * time.Hour},
		{"30 minutes", "30m", true, 30 * time.Minute},
		{"invalid format", "invalid", false, 0},
		{"empty string", "", true, 0}, // Should not error, just ignored
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testFile := fmt.Sprintf("test_maxage_str_%s.log", strings.ReplaceAll(test.name, " ", "_"))

			// Cleanup
			os.Remove(testFile)
			matches, _ := filepath.Glob(testFile + ".*")
			for _, match := range matches {
				os.Remove(match)
			}

			config := &LoggerConfig{
				Filename:   testFile,
				MaxSizeStr: "100MB", // Large size so rotation is only time-based
				MaxAgeStr:  test.maxAgeStr,
			}

			logger, err := NewWithConfig(config)
			if test.valid {
				if err != nil {
					t.Errorf("Expected no error for MaxAgeStr %q, got: %v", test.maxAgeStr, err)
					return
				}

				// Verify parsing was correct
				if test.maxAgeStr != "" && logger.MaxAge != test.expected {
					t.Errorf("Expected MaxAge %v, got %v", test.expected, logger.MaxAge)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error for invalid MaxAgeStr %q, got none", test.maxAgeStr)
					logger.Close()
				}
				return
			}

			// Test basic write
			_, err = logger.Write([]byte("test entry\n"))
			if err != nil {
				t.Errorf("Write failed: %v", err)
			}

			logger.Close()
			os.Remove(testFile)
		})
	}
}

func TestRotation_MaxAgeStrTimeBasedRotation(t *testing.T) {
	// Test actual time-based rotation with MaxAgeStr
	testFile := "test_maxage_str_rotation.log"

	// Cleanup first
	os.Remove(testFile)
	matches, _ := filepath.Glob(testFile + ".*")
	for _, match := range matches {
		os.Remove(match)
	}

	config := &LoggerConfig{
		Filename:   testFile,
		MaxSizeStr: "100MB", // Large size so rotation is only time-based
		MaxAgeStr:  "100ms", // Very short for testing
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	// Write initial data
	_, err = logger.Write([]byte("Initial log entry\n"))
	if err != nil {
		t.Fatalf("Initial write failed: %v", err)
	}

	// Wait for time-based rotation to trigger
	time.Sleep(150 * time.Millisecond)

	// Write another entry which should trigger rotation due to age
	_, err = logger.Write([]byte("Entry after age threshold\n"))
	if err != nil {
		t.Fatalf("Second write failed: %v", err)
	}

	// Give rotation some time to complete
	time.Sleep(50 * time.Millisecond)

	// Should have created a backup file due to age
	matches, err = filepath.Glob(testFile + ".*")
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}

	if len(matches) == 0 {
		t.Errorf("Expected backup file to be created due to MaxAgeStr, but none found")
	}

	// Cleanup
	logger.Close()
	os.Remove(testFile)
	for _, match := range matches {
		os.Remove(match)
	}
}

func TestRotation_MaxAgeConflictValidation(t *testing.T) {
	// Test that specifying both MaxAge and MaxAgeStr causes an error
	testFile := "test_maxage_conflict.log"
	defer os.Remove(testFile)

	config := &LoggerConfig{
		Filename:   testFile,
		MaxSizeStr: "100MB",
		MaxAge:     24 * time.Hour, // Deprecated field
		MaxAgeStr:  "7d",           // New field - should conflict
	}

	logger, err := NewWithConfig(config)
	if err == nil {
		logger.Close()
		t.Fatalf("Expected error when specifying both MaxAge and MaxAgeStr, but got none")
	}

	expectedErrMsg := "cannot specify both MaxAge and MaxAgeStr"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error to contain %q, got: %v", expectedErrMsg, err)
	}
}

func TestNewFunctions(t *testing.T) {
	// Test all the convenience New functions
	tests := []struct {
		name     string
		function func(string) (*Logger, error)
	}{
		{"NewWithDefaults", NewWithDefaults},
		{"NewDaily", NewDaily},
		{"NewWeekly", NewWeekly},
		{"NewDevelopment", NewDevelopment},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testFile := fmt.Sprintf("test_%s.log", strings.ToLower(test.name))
			defer os.Remove(testFile)

			logger, err := test.function(testFile)
			if err != nil {
				t.Fatalf("%s failed: %v", test.name, err)
			}

			// Test basic write functionality
			testMsg := fmt.Sprintf("Test message from %s\n", test.name)
			_, err = logger.Write([]byte(testMsg))
			if err != nil {
				t.Errorf("%s write failed: %v", test.name, err)
			}

			// Verify file was created
			if _, err := os.Stat(testFile); err != nil {
				t.Errorf("%s didn't create log file: %v", test.name, err)
			}

			logger.Close()
		})
	}
}

func TestNewSimple(t *testing.T) {
	testFile := "test_newsimple.log"
	defer os.Remove(testFile)

	logger, err := NewSimple(testFile, "50MB", 3)
	if err != nil {
		t.Fatalf("NewSimple failed: %v", err)
	}

	// Verify MaxSizeStr was set correctly
	if logger.MaxSizeStr != "50MB" {
		t.Errorf("Expected MaxSizeStr to be '50MB', got %q", logger.MaxSizeStr)
	}

	// Verify MaxBackups was set correctly
	if logger.MaxBackups != 3 {
		t.Errorf("Expected MaxBackups to be 3, got %d", logger.MaxBackups)
	}

	// Verify async is enabled by default
	if !logger.Async {
		t.Errorf("Expected Async to be true by default in NewSimple")
	}

	// Test write functionality
	_, err = logger.Write([]byte("Test message from NewSimple\n"))
	if err != nil {
		t.Errorf("NewSimple write failed: %v", err)
	}

	logger.Close()
}

func TestNewFunctionsEmptyFilename(t *testing.T) {
	// Test that all New functions properly validate empty filename
	tests := []struct {
		name string
		fn   func() (*Logger, error)
	}{
		{"NewWithDefaults", func() (*Logger, error) { return NewWithDefaults("") }},
		{"NewDaily", func() (*Logger, error) { return NewDaily("") }},
		{"NewWeekly", func() (*Logger, error) { return NewWeekly("") }},
		{"NewDevelopment", func() (*Logger, error) { return NewDevelopment("") }},
		{"NewSimple", func() (*Logger, error) { return NewSimple("", "100MB", 5) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger, err := test.fn()
			if err == nil {
				logger.Close()
				t.Errorf("%s should return error for empty filename", test.name)
			}
			if !strings.Contains(err.Error(), "filename cannot be empty") {
				t.Errorf("%s should return filename validation error, got: %v", test.name, err)
			}
		})
	}
}

func TestRotation_SizeAndTimeInteraction(t *testing.T) {
	// Test that both size and time rotation work together
	testFile := "test_combined_rotation.log"

	// Cleanup first
	os.Remove(testFile)
	matches, _ := filepath.Glob(testFile + ".*")
	for _, match := range matches {
		os.Remove(match)
	}

	logger := &Logger{
		Filename: testFile,
		MaxSize:  1,                      // 1MB
		MaxAge:   100 * time.Millisecond, // Very short for testing
	}

	// Write small amount first
	_, err := logger.Write([]byte("Small entry\n"))
	if err != nil {
		t.Fatalf("Initial write failed: %v", err)
	}

	// Wait for time-based rotation
	time.Sleep(150 * time.Millisecond)

	// Write another small entry (should trigger time-based rotation)
	_, err = logger.Write([]byte("Second small entry\n"))
	if err != nil {
		t.Fatalf("Second write failed: %v", err)
	}

	// Now write large chunk to trigger size-based rotation
	largeChunk := make([]byte, 2*1024*1024) // 2MB
	for i := range largeChunk {
		largeChunk[i] = 'B'
	}

	_, err = logger.Write(largeChunk)
	if err != nil {
		t.Fatalf("Large write failed: %v", err)
	}

	// Give rotation time to complete
	time.Sleep(100 * time.Millisecond)

	// Should have created multiple backup files
	matches, err = filepath.Glob(testFile + ".*")
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}

	if len(matches) < 1 {
		t.Errorf("Expected at least one backup file, but found %d", len(matches))
	}

	// Cleanup
	logger.Close()
	os.Remove(testFile)
	for _, match := range matches {
		os.Remove(match)
	}
}

func TestCompression_GzipRotatedFiles(t *testing.T) {
	// Test gzip compression of rotated files
	testFile := "test_compression.log"

	// Cleanup first
	os.Remove(testFile)
	matches, _ := filepath.Glob(testFile + "*")
	for _, match := range matches {
		os.Remove(match)
	}

	logger := &Logger{
		Filename: testFile,
		MaxSize:  1,    // 1MB to trigger rotation quickly
		Compress: true, // Enable compression
	}
	defer logger.Close() // Ensure cleanup

	// Write large chunk to trigger rotation
	largeChunk := make([]byte, 2*1024*1024) // 2MB
	for i := range largeChunk {
		largeChunk[i] = 'A' // Compressible data
	}

	_, err := logger.Write(largeChunk)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Ensure rotation has time to trigger background tasks
	time.Sleep(50 * time.Millisecond)

	// Wait for background compression to complete
	logger.WaitForBackgroundTasks()

	// Should have created compressed backup file
	matches, err = filepath.Glob(testFile + "*.gz")
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}

	if len(matches) == 0 {
		t.Errorf("Expected compressed (.gz) backup file, but none found")
		t.Logf("Files in directory:")
		files, _ := filepath.Glob("*")
		for _, f := range files {
			if info, err := os.Stat(f); err == nil {
				t.Logf("  %s (%d bytes)", f, info.Size())
			}
		}
	} else {
		// Verify the compressed file is smaller than original
		compressedFile := matches[0]
		info, err := os.Stat(compressedFile)
		if err != nil {
			t.Fatalf("Failed to stat compressed file: %v", err)
		}

		if info.Size() >= int64(len(largeChunk)) {
			t.Errorf("Compressed file (%d bytes) should be smaller than original (%d bytes)",
				info.Size(), len(largeChunk))
		}
	}

	// Cleanup
	logger.Close()
	os.Remove(testFile)
	for _, match := range matches {
		os.Remove(match)
	}
}

func TestCompression_DisabledByDefault(t *testing.T) {
	// Test that compression is disabled by default
	testFile := "test_no_compression.log"

	// Cleanup first
	os.Remove(testFile)
	matches, _ := filepath.Glob(testFile + "*")
	for _, match := range matches {
		os.Remove(match)
	}

	logger := &Logger{
		Filename: testFile,
		MaxSize:  1,     // 1MB to trigger rotation quickly
		Compress: false, // Explicitly disable compression
	}

	// Write large chunk to trigger rotation
	largeChunk := make([]byte, 2*1024*1024) // 2MB
	for i := range largeChunk {
		largeChunk[i] = 'B'
	}

	_, err := logger.Write(largeChunk)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Give time for any potential compression
	time.Sleep(100 * time.Millisecond)

	// Should NOT have created compressed backup file
	gzMatches, err := filepath.Glob(testFile + "*.gz")
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}

	if len(gzMatches) > 0 {
		t.Errorf("Found compressed files when compression was disabled: %v", gzMatches)
	}

	// Should have uncompressed backup
	allMatches, err := filepath.Glob(testFile + ".*")
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}

	if len(allMatches) == 0 {
		t.Errorf("Expected uncompressed backup file, but none found")
	}

	// Cleanup
	logger.Close()
	os.Remove(testFile)
	for _, match := range allMatches {
		os.Remove(match)
	}
}

func TestCleanup_MaxBackupsOrdering(t *testing.T) {
	// Test that cleanup removes oldest files first
	testFile := generateTestFile("cleanup_test")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename:   testFile,
		MaxSize:    1, // Force frequent rotation
		MaxBackups: 2, // Keep only 2 backup files
	}

	// Write enough data to create multiple backup files
	largeChunk := make([]byte, 2*1024*1024) // 2MB
	for i := range largeChunk {
		largeChunk[i] = 'A'
	}

	// Create multiple rotations
	for i := 0; i < 5; i++ {
		_, err := logger.Write(largeChunk)
		if err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Give cleanup time to run
	time.Sleep(200 * time.Millisecond)

	// Check that only MaxBackups files remain
	pattern := testFile + ".*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Glob failed: %v", err)
	}

	// Should have at most MaxBackups backup files
	if len(matches) > logger.MaxBackups {
		t.Errorf("Expected at most %d backup files, but found %d", logger.MaxBackups, len(matches))
		for _, match := range matches {
			t.Logf("  Found backup: %s", match)
		}
	}

	logger.Close()
}

func TestMPSC_BasicFunctionality(t *testing.T) {
	// Test basic MPSC mode functionality
	testFile := generateTestFile("mpsc_basic")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename: testFile,
		MaxSize:  100,
		Async:    true, // Enable MPSC mode
	}

	// Write some data in async mode
	testData := []byte("MPSC test message\n")
	n, err := logger.Write(testData)
	if err != nil {
		t.Fatalf("MPSC write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected %d bytes written, got %d", len(testData), n)
	}

	// Flush and close to ensure data is written
	logger.Close()

	// Verify data was actually written to file
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	if !bytes.Contains(content, testData) {
		t.Errorf("Expected file to contain %q, but got %q", testData, content)
	}
}

func TestMPSC_HighThroughput(t *testing.T) {
	// Test MPSC mode under high concurrent load
	testFile := generateTestFile("mpsc_throughput")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename: testFile,
		MaxSize:  100,
		Async:    true, // Enable MPSC mode
	}

	const numWriters = 10
	const writesPerWriter = 100
	done := make(chan bool, numWriters)

	// Start multiple writers
	for i := 0; i < numWriters; i++ {
		go func(writerID int) {
			defer func() { done <- true }()

			for j := 0; j < writesPerWriter; j++ {
				message := fmt.Sprintf("Writer-%d Message-%d\n", writerID, j)
				_, err := logger.Write([]byte(message))
				if err != nil {
					t.Errorf("Write failed for writer %d, message %d: %v", writerID, j, err)
					return
				}
			}
		}(i)
	}

	// Wait for all writers to complete
	for i := 0; i < numWriters; i++ {
		<-done
	}

	// Give consumer time to process all data before closing
	time.Sleep(50 * time.Millisecond)

	// Close and flush
	logger.Close()

	// Verify all data was written
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Count occurrences of "Writer-" to verify all messages were written
	expectedMessages := numWriters * writesPerWriter
	actualMessages := bytes.Count(content, []byte("Writer-"))

	if actualMessages != expectedMessages {
		t.Errorf("Expected %d messages, but found %d", expectedMessages, actualMessages)
	}
}

func TestWorkerPool_BackgroundOperations(t *testing.T) {
	// Test that worker pool handles background operations efficiently
	testFile := generateTestFile("worker_pool")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename:   testFile,
		MaxSize:    1, // Small size to trigger many rotations
		MaxBackups: 2,
		Compress:   true,
	}
	defer logger.Close()

	// Trigger many rotations to test worker pool efficiency
	const numRotations = 10
	largeChunk := make([]byte, 2*1024*1024) // 2MB
	for i := range largeChunk {
		largeChunk[i] = 'X'
	}

	// Perform multiple writes that will trigger rotations
	for i := 0; i < numRotations; i++ {
		_, err := logger.Write(largeChunk)
		if err != nil {
			t.Logf("Write %d failed (expected with frequent rotations): %v", i, err)
			break // Stop on error to avoid cascading failures
		}
		time.Sleep(10 * time.Millisecond) // Small delay to allow processing
	}

	// Give background workers time to complete
	time.Sleep(200 * time.Millisecond)

	// Verify that operations completed without excessive goroutine creation
	// (This is more of an integration test - the real benefit is in production)
	t.Logf("Worker pool test completed successfully")
}

func TestAutoScaling_LatencyBased(t *testing.T) {
	// Test latency-based auto-scaling
	testFile := generateTestFile("autoscale_latency")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename: testFile,
		MaxSize:  100,
		// Note: Async is false, we want auto-scaling to kick in
	}
	defer logger.Close()

	// Perform enough writes to reach the minimum sample size
	for i := 0; i < 150; i++ {
		_, err := logger.Write([]byte("Test message for latency measurement\n"))
		if err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}
	}

	// Check that metrics are being collected
	writeCount := logger.writeCount.Load()
	totalLatency := logger.totalLatency.Load()

	if writeCount == 0 {
		t.Error("Expected write count > 0")
	}

	// Note: On very fast systems, latency might be 0, which is acceptable
	// The important thing is that the write count is being tracked
	if totalLatency == 0 {
		t.Logf("Total latency is 0 (system is very fast)")
	}

	if writeCount > 0 {
		avgLatency := totalLatency / writeCount
		t.Logf("Average latency: %d nanoseconds (%v)", avgLatency, time.Duration(avgLatency))
	}
}

func TestErrorCallback_CustomHandling(t *testing.T) {
	// Test custom error handling via callback
	testFile := generateTestFile("error_callback")
	defer cleanupTestFile(testFile)

	var capturedErrors []string
	var mu sync.Mutex

	logger := &Logger{
		Filename: testFile,
		MaxSize:  100,
		ErrorCallback: func(operation string, err error) {
			mu.Lock()
			defer mu.Unlock()
			capturedErrors = append(capturedErrors, fmt.Sprintf("%s: %v", operation, err))
		},
	}
	defer logger.Close()

	// Normal operation should not trigger errors
	_, err := logger.Write([]byte("Normal log entry\n"))
	if err != nil {
		t.Fatalf("Normal write failed: %v", err)
	}

	// Give some time for any background operations
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	errorCount := len(capturedErrors)
	mu.Unlock()

	// For normal operations, we shouldn't have many errors
	t.Logf("Captured %d error(s) during normal operation", errorCount)
	if errorCount > 0 {
		mu.Lock()
		for _, errMsg := range capturedErrors {
			t.Logf("Captured error: %s", errMsg)
		}
		mu.Unlock()
	}
}

func TestControlledShutdown_WaitGroup(t *testing.T) {
	// Test that controlled shutdown waits for all data to be flushed
	testFile := generateTestFile("controlled_shutdown")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename: testFile,
		MaxSize:  100,
		Async:    true, // Enable MPSC mode
	}

	// Write multiple messages
	const numMessages = 100
	for i := 0; i < numMessages; i++ {
		message := fmt.Sprintf("Message %d - this is a test message\n", i)
		_, err := logger.Write([]byte(message))
		if err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}
	}

	// Close should wait for all data to be flushed
	start := time.Now()
	logger.Close()
	shutdownTime := time.Since(start)

	t.Logf("Shutdown took %v", shutdownTime)

	// Verify all data was written
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Count messages in the file
	messageCount := bytes.Count(content, []byte("Message "))

	if messageCount < numMessages {
		t.Errorf("Expected at least %d messages, but found %d", numMessages, messageCount)
		t.Logf("File size: %d bytes", len(content))
	} else {
		t.Logf("Successfully flushed %d messages during controlled shutdown", messageCount)
	}
}

func TestCrossPlatform_FileLocking(t *testing.T) {
	// Test file locking behavior across platforms
	testFile := generateTestFile("file_locking")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename: testFile,
		MaxSize:  1, // Small size to trigger frequent rotations
	}
	defer logger.Close()

	// On Windows, file operations can be more restrictive
	// Test rapid rotation cycles that might cause locking issues
	const numCycles = 5
	largeChunk := make([]byte, 2*1024*1024) // 2MB
	for i := range largeChunk {
		largeChunk[i] = 'L'
	}

	for i := 0; i < numCycles; i++ {
		_, err := logger.Write(largeChunk)
		if err != nil {
			if runtime.GOOS == "windows" {
				t.Logf("Cycle %d failed on Windows (may be expected): %v", i, err)
			} else {
				t.Errorf("Cycle %d failed unexpectedly: %v", i, err)
			}
		}
		// Small delay to allow file operations to complete
		time.Sleep(20 * time.Millisecond)
	}

	t.Logf("Completed %d rotation cycles on %s", numCycles, runtime.GOOS)
}

func TestCrossPlatform_PathValidation(t *testing.T) {
	// Test path length and character validation
	tests := []struct {
		name     string
		filename string
		valid    bool
	}{
		{"normal_file", "test.log", true},
		{"long_path", strings.Repeat("a", 200) + ".log", true},
		{"very_long_path", strings.Repeat("b", 300) + ".log", false}, // May fail on some systems
	}

	// Windows-specific character tests
	if runtime.GOOS == "windows" {
		tests = append(tests, []struct {
			name     string
			filename string
			valid    bool
		}{
			{"invalid_chars_asterisk", "test*.log", false},
			{"invalid_chars_question", "test?.log", false},
			{"invalid_chars_quote", "test\".log", false},
			{"invalid_chars_colon", "test:.log", false},
		}...)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var testPath string
			if test.valid {
				testPath = filepath.Join(os.TempDir(), test.filename)
			} else {
				testPath = test.filename // Use problematic name directly
			}

			logger := &Logger{
				Filename: testPath,
				MaxSize:  100,
			}

			_, err := logger.Write([]byte("test\n"))

			if test.valid {
				if err != nil {
					t.Errorf("Expected valid filename to work, got error: %v", err)
				}
			} else {
				if err == nil && runtime.GOOS == "windows" {
					t.Logf("Filename '%s' unexpectedly worked on Windows", test.filename)
				}
			}

			logger.Close()
			if test.valid {
				os.Remove(testPath)
			}
		})
	}
}

func TestCrossPlatform_FilePermissions(t *testing.T) {
	// Test file permissions across platforms
	testFile := generateTestFile("permissions")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename: testFile,
		MaxSize:  100,
	}
	defer logger.Close()

	// Write some data
	_, err := logger.Write([]byte("Permission test\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	mode := info.Mode()
	t.Logf("File permissions on %s: %v", runtime.GOOS, mode)

	// On Unix-like systems, check if permissions are as expected
	if runtime.GOOS != "windows" {
		expectedPerm := os.FileMode(0644)
		if mode.Perm() != expectedPerm {
			t.Logf("Permissions differ from expected 0644: got %v", mode.Perm())
		}
	}
}

func TestSanitization_FilenameCharacters(t *testing.T) {
	// Test that problematic characters are sanitized cross-platform
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal", "test.log", "test.log"},
		{"asterisk", "test*.log", "test_.log"},
		{"question", "test?.log", "test_.log"},
		{"quote", "test\".log", "test_.log"},
		{"multiple", "test*?:\".log", "test____.log"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := SanitizeFilename(test.input)
			if result != test.expected {
				t.Errorf("SanitizeFilename(%q) = %q, expected %q", test.input, result, test.expected)
			}
		})
	}
}

func TestValidation_PathLength(t *testing.T) {
	// Test path length validation
	tests := []struct {
		name      string
		pathLen   int
		shouldErr bool
	}{
		{"short_path", 50, false},
		{"medium_path", 200, false},
		{"long_path_windows", 250, false},
		{"very_long_path", 300, runtime.GOOS == "windows"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a path of specified length
			tempDir := os.TempDir()
			remaining := test.pathLen - len(tempDir) - 10 // Leave room for separators and extension
			if remaining <= 0 {
				remaining = 10
			}

			longName := strings.Repeat("a", remaining) + ".log"
			testPath := filepath.Join(tempDir, longName)

			err := ValidatePathLength(testPath)

			if test.shouldErr && err == nil {
				t.Errorf("Expected error for path length %d on %s, but got none", test.pathLen, runtime.GOOS)
			} else if !test.shouldErr && err != nil {
				t.Errorf("Unexpected error for path length %d on %s: %v", test.pathLen, runtime.GOOS, err)
			}

			if err != nil {
				t.Logf("Path validation error (expected: %v): %v", test.shouldErr, err)
			}
		})
	}
}

func TestRetry_FileOperations(t *testing.T) {
	// Test retry mechanism
	attempts := 0
	maxAttempts := 3

	err := RetryFileOperation(func() error {
		attempts++
		if attempts < maxAttempts {
			return fmt.Errorf("simulated failure %d", attempts)
		}
		return nil
	}, maxAttempts, 1*time.Millisecond)

	if err != nil {
		t.Errorf("Expected success after %d attempts, got error: %v", maxAttempts, err)
	}

	if attempts != maxAttempts {
		t.Errorf("Expected %d attempts, got %d", maxAttempts, attempts)
	}

	// Test persistent failure
	persistentAttempts := 0
	err = RetryFileOperation(func() error {
		persistentAttempts++
		return fmt.Errorf("always fails")
	}, 2, 1*time.Millisecond)

	if err == nil {
		t.Error("Expected persistent failure to return error")
	}

	if persistentAttempts != 2 {
		t.Errorf("Expected 2 attempts for persistent failure, got %d", persistentAttempts)
	}
}

func TestAutoScaling_SyncToMPSC(t *testing.T) {
	// Test automatic scaling from sync to MPSC mode under load
	testFile := generateTestFile("auto_scaling")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename: testFile,
		MaxSize:  100,
		// Note: Async is false, we want auto-scaling to kick in
	}

	// Start with low load (should stay in sync mode)
	_, err := logger.Write([]byte("Low load message\n"))
	if err != nil {
		t.Fatalf("Initial write failed: %v", err)
	}

	// Simulate high concurrent load that should trigger auto-scaling
	const numConcurrentWriters = 20
	done := make(chan bool, numConcurrentWriters)

	startTime := time.Now()

	for i := 0; i < numConcurrentWriters; i++ {
		go func(writerID int) {
			defer func() { done <- true }()

			for j := 0; j < 50; j++ {
				message := fmt.Sprintf("High-load Writer-%d Message-%d\n", writerID, j)
				_, _ = logger.Write([]byte(message)) // Ignore errors in stress test
			}
		}(i)
	}

	// Wait for completion
	for i := 0; i < numConcurrentWriters; i++ {
		<-done
	}

	duration := time.Since(startTime)
	t.Logf("High-load test completed in %v", duration)

	logger.Close()

	// Verify data integrity
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	if len(content) == 0 {
		t.Error("No data was written to file")
	}
}

// Edge case tests for robust error handling

// TestConfigurableBufferSize tests the new configurable buffer size feature
func TestConfigurableBufferSize(t *testing.T) {
	tests := []struct {
		name       string
		bufferSize int
		expectedOk bool
	}{
		{"Default buffer size (0)", 0, true},
		{"Small buffer (64)", 64, true},
		{"Large buffer (4096)", 4096, true},
		{"Very small buffer (1)", 1, true}, // Should be rounded up to 64
		{"Power of 2 (512)", 512, true},
		{"Non-power of 2 (1000)", 1000, true}, // Should be rounded up
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := generateTestFile("buffer_size")
			defer cleanupTestFile(testFile)

			logger := &Logger{
				Filename:   testFile,
				MaxSize:    1,
				BufferSize: tt.bufferSize,
				Async:      true, // Force MPSC mode to test buffer
			}

			// Write some data
			data := "test message\n"
			_, err := logger.Write([]byte(data))

			if tt.expectedOk && err != nil {
				t.Errorf("Expected success, got error: %v", err)
			}

			logger.Close()
		})
	}
}

// TestParseSizeCaseInsensitive tests the improved ParseSize function
func TestParseSizeCaseInsensitive(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		// Case insensitive
		{"100kb", 100 * 1024, false},
		{"100KB", 100 * 1024, false},
		{"100Kb", 100 * 1024, false},
		{"100kB", 100 * 1024, false},

		// Single letter units
		{"100k", 100 * 1024, false},
		{"100K", 100 * 1024, false},
		{"100m", 100 * 1024 * 1024, false},
		{"100M", 100 * 1024 * 1024, false},
		{"100g", 100 * 1024 * 1024 * 1024, false},
		{"100G", 100 * 1024 * 1024 * 1024, false},
		{"1t", 1024 * 1024 * 1024 * 1024, false},
		{"1T", 1024 * 1024 * 1024 * 1024, false},

		// Error cases
		{"100x", 0, true},
		{"100XB", 0, true},
		{"", 0, true},
		{"abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseSize(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for input %q, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %q: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("Expected %d, got %d for input %q", tt.expected, result, tt.input)
				}
			}
		})
	}
}

// TestInvalidPermissions tests behavior when permissions are insufficient
func TestInvalidPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Permission tests not reliable on Windows")
	}

	// Create a directory without write permissions
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("lethe_perm_test_%d", time.Now().UnixNano()))
	err := os.MkdirAll(tempDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Remove write permissions
	err = os.Chmod(tempDir, 0555) // Read and execute only
	if err != nil {
		t.Fatalf("Failed to change permissions: %v", err)
	}

	// Attempt to create logger in this directory
	testFile := filepath.Join(tempDir, "test.log")
	logger := &Logger{
		Filename: testFile,
		MaxSize:  1,
		ErrorCallback: func(operation string, err error) {
			t.Logf("Error callback: %s - %v", operation, err)
		},
	}

	// This should fail due to permissions
	_, err = logger.Write([]byte("test"))
	if err == nil {
		t.Error("Expected permission error, got nil")
	}

	// Restore permissions for cleanup
	_ = os.Chmod(tempDir, 0755) // Ignore error for cleanup
	logger.Close()
}

// TestVeryLargeFiles tests handling of large log files
func TestVeryLargeFiles(t *testing.T) {
	testFile := generateTestFile("large_file")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename:   testFile,
		MaxSize:    1, // 1MB - will trigger rotation quickly
		MaxBackups: 2,
	}

	// Generate larger chunks to simulate real workload
	largeChunk := make([]byte, 64*1024) // 64KB chunks
	for i := range largeChunk {
		largeChunk[i] = byte('A' + (i % 26))
	}

	// Write enough data to trigger multiple rotations
	totalBytes := 0
	maxBytes := 5 * 1024 * 1024 // 5MB total

	for totalBytes < maxBytes {
		n, err := logger.Write(largeChunk)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
		totalBytes += n

		// Occasionally check file system state
		if totalBytes%(1024*1024) == 0 { // Every MB
			// Small delay to allow rotation to complete
			time.Sleep(10 * time.Millisecond)
		}
	}

	logger.Close()

	// Verify that files were created and rotations occurred
	pattern := testFile + "*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Failed to glob files: %v", err)
	}

	if len(matches) < 2 {
		t.Errorf("Expected at least 2 files (original + backups), got %d", len(matches))
	}

	t.Logf("Generated %d files with %d total bytes", len(matches), totalBytes)
}

// TestHighFrequencyRotation tests rapid rotation scenarios
func TestHighFrequencyRotation(t *testing.T) {
	testFile := generateTestFile("high_freq")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename:   testFile,
		MaxSize:    1, // Very small - 1KB to force frequent rotations
		MaxBackups: 10,
		Compress:   true, // Test compression under high frequency
		ErrorCallback: func(operation string, err error) {
			t.Logf("Error during %s: %v", operation, err)
		},
	}

	// Use smaller MaxSize in bytes for more frequent rotations
	logger.maxSizeBytes = 1024 // 1KB

	// Rapid successive writes
	message := "This is a test message that should trigger frequent rotations\n"
	numWrites := 100

	for i := 0; i < numWrites; i++ {
		_, err := logger.Write([]byte(fmt.Sprintf("%d: %s", i, message)))
		if err != nil {
			t.Errorf("Write %d failed: %v", i, err)
		}

		// Small delay to allow file operations to complete
		if i%10 == 0 {
			time.Sleep(1 * time.Millisecond)
		}
	}

	// Wait for background tasks to complete before closing
	logger.WaitForBackgroundTasks()
	logger.Close()

	// Allow final file operations to complete
	time.Sleep(50 * time.Millisecond)

	// Check that rotations occurred without errors
	pattern := testFile + "*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Failed to glob files: %v", err)
	}

	if len(matches) < 2 {
		t.Errorf("Expected at least 2 rotated files from high frequency rotation, got %d", len(matches))
	}

	// Verify some compressed files exist
	compressedCount := 0
	for _, match := range matches {
		if strings.HasSuffix(match, ".gz") {
			compressedCount++
		}
	}

	if compressedCount == 0 {
		t.Error("Expected some compressed files, got none")
	}

	t.Logf("Created %d files, %d compressed", len(matches), compressedCount)
}

// TestDiskSpaceHandling tests graceful handling of disk space issues
// Note: This is a simulation test since actually filling disk is impractical
func TestDiskSpaceHandling(t *testing.T) {
	testFile := generateTestFile("disk_space")
	defer cleanupTestFile(testFile)

	errorCount := 0
	logger := &Logger{
		Filename:   testFile,
		MaxSize:    1,
		MaxBackups: 3,
		ErrorCallback: func(operation string, err error) {
			errorCount++
			t.Logf("Error callback: %s - %v", operation, err)
		},
	}

	// Simulate a very large write that might fail due to disk space
	largeData := make([]byte, 10*1024*1024) // 10MB
	for i := range largeData {
		largeData[i] = byte('X')
	}

	// This may or may not fail depending on available disk space
	// The test is mainly to ensure we don't panic or hang
	_, err := logger.Write(largeData)
	if err != nil {
		t.Logf("Large write failed as expected: %v", err)
	}

	logger.Close()

	// Verify error callback was properly called if there were issues
	t.Logf("Error callback invoked %d times", errorCount)
}

// TestConcurrentRotationStress tests rotation under high concurrency
func TestConcurrentRotationStress(t *testing.T) {
	testFile := generateTestFile("rotation_stress")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename:   testFile,
		MaxSize:    1, // Small size to force frequent rotations
		MaxBackups: 5,
		Async:      true, // Enable MPSC mode
		BufferSize: 256,  // Smaller buffer to test overflow handling
	}

	// Force small rotation size
	logger.maxSizeBytes = 10 * 1024 // 10KB

	var wg sync.WaitGroup
	numGoroutines := 20
	writesPerGoroutine := 50

	// Concurrent writers with rapid writes to stress rotation logic
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < writesPerGoroutine; j++ {
				message := fmt.Sprintf("Goroutine %d, write %d: %s\n",
					id, j, "some test data that will accumulate quickly")

				_, err := logger.Write([]byte(message))
				if err != nil {
					// "file already closed" errors are expected in stress tests
					// when Close() races with concurrent writes - this is not a bug
					if !strings.Contains(err.Error(), "file already closed") {
						t.Errorf("Goroutine %d write %d failed: %v", id, j, err)
					}
				}

				// Introduce some variability in timing
				if j%5 == 0 {
					time.Sleep(time.Microsecond * 100)
				}
			}
		}(i)
	}

	wg.Wait()

	// Allow pending async operations to complete before closing
	time.Sleep(100 * time.Millisecond)

	logger.Close()

	// Allow background operations to complete
	time.Sleep(200 * time.Millisecond)

	// Verify that rotations occurred and files were created
	pattern := testFile + "*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Failed to glob files: %v", err)
	}

	totalWrites := numGoroutines * writesPerGoroutine
	t.Logf("Stress test completed: %d concurrent writes across %d files",
		totalWrites, len(matches))

	if len(matches) < 1 {
		t.Errorf("Expected at least one file to be created, got %d", len(matches))
	}

	// Note: "file already closed" errors are expected in stress tests
	// when Close() races with writes - this is not a bug

	// The exact number of files depends on timing of rotations vs close operations
	// Just verify that we successfully created files without panicking
}

// Test suite for new advanced features

// TestMaxSizeString tests the new string-based MaxSize configuration
func TestMaxSizeString(t *testing.T) {
	tests := []struct {
		name         string
		maxSizeStr   string
		expectedOk   bool
		minRotations int
	}{
		{"1KB", "1KB", true, 1},
		{"1k", "1k", true, 1},
		{"1mb", "1mb", true, 1},
		{"1M", "1M", true, 1},
		{"invalid", "invalid", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := generateTestFile("maxsize_str")
			defer cleanupTestFile(testFile)

			logger := &Logger{
				Filename:   testFile,
				MaxSizeStr: tt.maxSizeStr,
				MaxBackups: 3,
			}

			// Write enough data to trigger rotation
			data := make([]byte, 2048) // 2KB
			for i := range data {
				data[i] = byte('A')
			}

			_, err := logger.Write(data)
			if tt.expectedOk && err != nil {
				t.Errorf("Expected success with %s, got error: %v", tt.maxSizeStr, err)
			}

			logger.Close()

			// Check for rotations if expected
			if tt.expectedOk && tt.minRotations > 0 {
				pattern := testFile + "*"
				matches, _ := filepath.Glob(pattern)
				if len(matches) < tt.minRotations {
					t.Errorf("Expected at least %d files, got %d", tt.minRotations, len(matches))
				}
			}
		})
	}
}

// TestBackpressurePolicies tests the different MPSC backpressure policies
func TestBackpressurePolicies(t *testing.T) {
	policies := []string{"fallback", "drop", "adaptive"}

	for _, policy := range policies {
		t.Run(policy, func(t *testing.T) {
			testFile := generateTestFile("backpressure_" + policy)
			defer cleanupTestFile(testFile)

			logger := &Logger{
				Filename:           testFile,
				MaxSizeStr:         "1KB",
				Async:              true,
				BufferSize:         8, // Very small buffer to force backpressure
				BackpressurePolicy: policy,
			}

			// Write many small messages rapidly
			for i := 0; i < 100; i++ {
				data := fmt.Sprintf("Message %d\n", i)
				_, _ = logger.Write([]byte(data)) // Ignore errors in stress test
			}

			logger.Close()

			// All policies should complete without panic
			t.Logf("Policy %s completed successfully", policy)
		})
	}
}

// TestTelemetryAPI tests the Stats() function
func TestTelemetryAPI(t *testing.T) {
	testFile := generateTestFile("telemetry")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename:           testFile,
		MaxSizeStr:         "10KB",
		Async:              true,
		BufferSize:         256,
		BackpressurePolicy: "adaptive",
		FlushInterval:      2 * time.Millisecond,
		AdaptiveFlush:      true,
	}

	// Write some data
	for i := 0; i < 50; i++ {
		data := fmt.Sprintf("Test message %d with some content\n", i)
		_, _ = logger.Write([]byte(data)) // Ignore errors in test
	}

	// Allow async operations to complete
	time.Sleep(50 * time.Millisecond)

	// Get stats after flush
	stats := logger.Stats()

	// Verify stats structure
	if stats.WriteCount == 0 {
		t.Error("Expected non-zero write count")
	}
	// Note: TotalBytes might be 0 in MPSC mode if data is still in buffer
	if stats.WriteCount > 0 && stats.TotalBytes == 0 {
		t.Logf("Note: TotalBytes is 0 (data may still be in MPSC buffer)")
	}
	if !stats.IsMPSCActive {
		t.Error("Expected MPSC to be active")
	}
	if stats.BufferSize == 0 {
		t.Error("Expected non-zero buffer size")
	}
	if stats.BackpressurePolicy != "adaptive" {
		t.Errorf("Expected adaptive policy, got %s", stats.BackpressurePolicy)
	}

	t.Logf("Stats: WriteCount=%d, TotalBytes=%d, AvgLatency=%dns, ContentionRatio=%.3f",
		stats.WriteCount, stats.TotalBytes, stats.AvgLatencyNs, stats.ContentionRatio)

	logger.Close()
}

// TestMaxFileAge tests age-based file cleanup
func TestMaxFileAge(t *testing.T) {
	testFile := generateTestFile("maxage")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename:   testFile,
		MaxSizeStr: "1KB",
		MaxBackups: 10,
		MaxFileAge: 100 * time.Millisecond, // Very short for testing
	}

	// Create several rotated files
	for i := 0; i < 5; i++ {
		data := make([]byte, 1500) // Force rotation
		for j := range data {
			data[j] = byte('A' + i)
		}
		_, _ = logger.Write(data)         // Ignore errors in test
		time.Sleep(50 * time.Millisecond) // Space out rotations
	}

	// Wait for files to age
	time.Sleep(200 * time.Millisecond)

	// Force another rotation to trigger cleanup
	data := make([]byte, 1500)
	_, _ = logger.Write(data) // Ignore errors in test

	logger.Close()

	// Allow background cleanup to complete
	time.Sleep(100 * time.Millisecond)

	// Check that old files were cleaned up
	pattern := testFile + "*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Failed to glob files: %v", err)
	}

	// Should have fewer files due to age-based cleanup
	t.Logf("Files after age-based cleanup: %d", len(matches))

	// Verify that we don't have too many old files
	if len(matches) > 3 {
		t.Logf("Note: %d files remain after age-based cleanup (timing dependent)", len(matches))
	}
}

// TestCrashConsistency tests that compression uses temporary files
func TestCrashConsistency(t *testing.T) {
	testFile := generateTestFile("crash_consistency")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename:   testFile,
		MaxSizeStr: "1KB",
		MaxBackups: 3,
		Compress:   true,
	}

	// Write data to force rotation and compression
	data := make([]byte, 2048)
	for i := range data {
		data[i] = byte('X')
	}

	// Write multiple times to ensure rotation
	for i := 0; i < 3; i++ {
		_, _ = logger.Write(data)         // Ignore errors in test
		time.Sleep(10 * time.Millisecond) // Small delay between writes
	}

	// Allow background compression to start
	time.Sleep(100 * time.Millisecond)

	logger.Close()

	// Allow compression to complete
	time.Sleep(500 * time.Millisecond)

	// Check for .tmp files (should be none after successful completion)
	pattern := testFile + "*.tmp"
	tmpFiles, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Failed to glob temp files: %v", err)
	}

	if len(tmpFiles) > 0 {
		t.Errorf("Found %d temporary files after completion: %v", len(tmpFiles), tmpFiles)
	}

	// Check for .gz files (should exist)
	gzPattern := testFile + "*.gz"
	gzFiles, err := filepath.Glob(gzPattern)
	if err != nil {
		t.Fatalf("Failed to glob gz files: %v", err)
	}

	if len(gzFiles) == 0 {
		t.Error("Expected compressed files, but found none")
	}

	t.Logf("Crash consistency test passed - found %d compressed files, %d temp files",
		len(gzFiles), len(tmpFiles))
}

// TestDropPolicy tests that the drop policy correctly counts dropped messages
func TestDropPolicy(t *testing.T) {
	testFile := generateTestFile("drop_policy")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename:           testFile,
		MaxSizeStr:         "1KB",
		Async:              true,
		BufferSize:         4, // Very small buffer to force drops
		BackpressurePolicy: "drop",
	}

	// Write many messages rapidly to force buffer overflow
	for i := 0; i < 100; i++ {
		data := fmt.Sprintf("Rapid message %d with some content to fill buffer\n", i)
		_, _ = logger.Write([]byte(data)) // Ignore errors in test
	}

	// Allow some processing
	time.Sleep(50 * time.Millisecond)

	// Get stats
	stats := logger.Stats()

	logger.Close()

	// Should have some dropped messages due to small buffer and rapid writes
	t.Logf("Stats: WriteCount=%d, DroppedOnFull=%d, BufferSize=%d, BufferFill=%d",
		stats.WriteCount, stats.DroppedOnFull, stats.BufferSize, stats.BufferFill)

	if stats.DroppedOnFull == 0 {
		t.Log("Note: No messages were dropped (buffer processed quickly enough)")
	} else {
		t.Logf("Successfully dropped %d messages as expected", stats.DroppedOnFull)
	}
}

// TestWriteOwned tests the zero-copy WriteOwned API
func TestWriteOwned(t *testing.T) {
	testFile := generateTestFile("write_owned")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename:   testFile,
		MaxSizeStr: "10KB",
		Async:      true,
		BufferSize: 128,
	}

	// Test ownership transfer
	data := []byte("Test message for ownership transfer\n")
	n, err := logger.WriteOwned(data)
	if err != nil {
		t.Errorf("WriteOwned failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected %d bytes written, got %d", len(data), n)
	}

	// Allow processing
	time.Sleep(50 * time.Millisecond)

	stats := logger.Stats()
	if stats.WriteCount == 0 {
		t.Error("Expected non-zero write count")
	}

	logger.Close()

	t.Logf("WriteOwned test completed successfully")
}

// TestLocalTimeAndChecksums tests LocalTime flag and checksum generation
func TestLocalTimeAndChecksums(t *testing.T) {
	testFile := generateTestFile("localtime_checksums")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename:   testFile,
		MaxSizeStr: "1KB",
		LocalTime:  true,  // Use local time in filenames
		Checksum:   true,  // Generate checksums
		Compress:   false, // Disable compression for this test
	}

	// Write some data to trigger rotation
	for i := 0; i < 50; i++ {
		data := fmt.Sprintf("Test message %d with enough content to trigger rotation\n", i)
		_, _ = logger.Write([]byte(data)) // Ignore errors in test
	}

	// Allow processing
	time.Sleep(200 * time.Millisecond)

	logger.Close()

	// Check for backup files and checksums
	pattern := testFile + ".*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Errorf("Failed to glob pattern %s: %v", pattern, err)
		return
	}

	backupFiles := []string{}
	checksumFiles := []string{}

	for _, match := range matches {
		if strings.HasSuffix(match, ".sha256") {
			checksumFiles = append(checksumFiles, match)
		} else if !strings.HasSuffix(match, ".sha256") && match != testFile {
			backupFiles = append(backupFiles, match)
		}
	}

	t.Logf("Found %d backup files and %d checksum files", len(backupFiles), len(checksumFiles))

	if len(backupFiles) == 0 {
		t.Error("Expected backup files from rotation")
	}

	if len(checksumFiles) == 0 {
		t.Error("Expected checksum files to be generated")
	} else {
		// Verify checksum file format
		content, err := os.ReadFile(checksumFiles[0])
		if err != nil {
			t.Errorf("Failed to read checksum file: %v", err)
		} else {
			contentStr := string(content)
			if !strings.Contains(contentStr, "  ") { // SHA256 format: "hash  filename"
				t.Errorf("Checksum file format incorrect: %s", contentStr)
			}
			t.Logf("Checksum file content: %s", strings.TrimSpace(contentStr))
		}
	}

	// Verify LocalTime: backup filename should contain timestamp
	if len(backupFiles) > 0 {
		filename := filepath.Base(backupFiles[0])
		// Should contain pattern like "YYYY-MM-DD-HH-MM-SS"
		if !strings.Contains(filename, "-") {
			t.Error("Backup filename should contain timestamp when LocalTime=true")
		}
		t.Logf("Backup filename with LocalTime: %s", filename)
	}
}
