// Lethe Framework Integration Tests
// Tests for integration with popular Go logging frameworks
// Copyright 2025 AGILira
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agilira/lethe"
)

// generateTestFile creates a unique test file name
func generateTestFile(framework string) string {
	timestamp := time.Now().UnixNano()
	return filepath.Join(".", fmt.Sprintf("test_%s_%d.log", framework, timestamp))
}

// cleanupTestFile removes test files and related files
func cleanupTestFile(testFile string) {
	// Remove main file
	os.Remove(testFile)

	// Remove potential backup files
	pattern := testFile + "*"
	matches, _ := filepath.Glob(pattern)
	for _, match := range matches {
		os.Remove(match)
	}
}

// TestStandardLibraryIntegration tests integration with Go's standard log package
func TestStandardLibraryIntegration(t *testing.T) {
	testFile := generateTestFile("stdlib")
	defer cleanupTestFile(testFile)

	// Create Lethe logger with minimal configuration
	logger := &lethe.Logger{
		Filename: testFile,
		MaxSize:  1,     // Use old format to ensure compatibility
		Async:    false, // Force sync mode for testing
	}

	// Force initialization by writing directly first
	logger.Write([]byte("Initialization test\n"))

	// Set as output for standard library logger
	originalOutput := log.Writer()
	log.SetOutput(logger)
	defer log.SetOutput(originalOutput) // Restore default

	// Test logging
	log.Println("Standard library integration test message 1")
	log.Printf("Standard library integration test message %d", 2)
	log.Print("Standard library integration test message 3")

	// Force some data to be written
	data := strings.Repeat("Standard library test data ", 50) // ~1.3KB
	log.Println(data)

	// Allow time for writes and close logger to ensure sync
	time.Sleep(50 * time.Millisecond)
	logger.Close()
	time.Sleep(50 * time.Millisecond)

	// Verify file was created and contains data
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatalf("Log file was not created: %s", testFile)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	t.Logf("Debug: File content length: %d, content: %q", len(content), string(content))

	if len(content) == 0 {
		t.Error("Log file is empty - possible sync issue")
		return
	}

	if !strings.Contains(string(content), "Standard library integration test message") {
		t.Errorf("Log file does not contain expected content. Got: %q", string(content))
	} else {
		t.Logf("✅ Standard library integration successful - log file size: %d bytes", len(content))
	}
}

// TestCustomLoggerIntegration tests integration with a custom logger that uses io.Writer
func TestCustomLoggerIntegration(t *testing.T) {
	testFile := generateTestFile("custom")
	defer cleanupTestFile(testFile)

	// Create Lethe logger
	rotator := &lethe.Logger{
		Filename:   testFile,
		MaxSizeStr: "2KB",
		MaxBackups: 2,
		Async:      true, // Test MPSC mode
		BufferSize: 128,
	}

	// Custom logger implementation
	type CustomLogger struct {
		writer *lethe.Logger
		prefix string
		level  string
	}

	customLogger := &CustomLogger{
		writer: rotator,
		prefix: "[CUSTOM]",
		level:  "INFO",
	}

	// Custom log method
	logMethod := func(level, message string) {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		logLine := fmt.Sprintf("%s %s %s: %s\n", timestamp, customLogger.prefix, level, message)
		customLogger.writer.Write([]byte(logLine))
	}

	// Test various log levels
	logMethod("INFO", "Custom logger integration test - info message")
	logMethod("WARN", "Custom logger integration test - warning message")
	logMethod("ERROR", "Custom logger integration test - error message")
	logMethod("DEBUG", "Custom logger integration test - debug message")

	// Test with larger payload to trigger rotation
	largeData := strings.Repeat("Custom logger test with large payload ", 100) // ~3.4KB
	logMethod("INFO", largeData)

	// Allow MPSC to flush
	time.Sleep(100 * time.Millisecond)
	rotator.Close()

	// Verify files were created (should have rotated)
	pattern := testFile + "*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Failed to glob files: %v", err)
	}

	if len(matches) < 1 {
		t.Fatalf("Expected at least one log file, got %d", len(matches))
	}

	// Read and verify content
	var totalContent bytes.Buffer
	for _, match := range matches {
		content, err := os.ReadFile(match)
		if err != nil {
			t.Logf("Could not read file %s: %v", match, err)
			continue
		}
		totalContent.Write(content)
	}

	contentStr := totalContent.String()
	if !strings.Contains(contentStr, "[CUSTOM]") {
		t.Error("Log content does not contain custom prefix")
	}
	if !strings.Contains(contentStr, "Custom logger integration test") {
		t.Error("Log content does not contain expected messages")
	}

	t.Logf("✅ Custom logger integration successful - created %d files, total content: %d bytes",
		len(matches), totalContent.Len())
}

// TestBufferedWriterIntegration tests integration with buffered writers
func TestBufferedWriterIntegration(t *testing.T) {
	testFile := generateTestFile("buffered")
	defer cleanupTestFile(testFile)

	// Create Lethe logger
	rotator := &lethe.Logger{
		Filename:   testFile,
		MaxSizeStr: "1KB",
		MaxBackups: 3,
		Async:      false, // Test sync mode for this test
	}

	// Simulate buffered writer pattern (common in many frameworks)
	type BufferedWriter struct {
		underlying *lethe.Logger
		buffer     bytes.Buffer
		threshold  int
	}

	buffered := &BufferedWriter{
		underlying: rotator,
		threshold:  256, // Flush every 256 bytes
	}

	// Buffered write method
	writeBuffered := func(data []byte) error {
		buffered.buffer.Write(data)
		if buffered.buffer.Len() >= buffered.threshold {
			// Flush to underlying writer
			_, err := buffered.underlying.Write(buffered.buffer.Bytes())
			buffered.buffer.Reset()
			return err
		}
		return nil
	}

	// Force flush method
	flush := func() error {
		if buffered.buffer.Len() > 0 {
			_, err := buffered.underlying.Write(buffered.buffer.Bytes())
			buffered.buffer.Reset()
			return err
		}
		return nil
	}

	// Test buffered writes
	for i := 0; i < 10; i++ {
		message := fmt.Sprintf("Buffered write test message %d - ", i)
		data := bytes.Repeat([]byte(message), 5) // ~150 bytes per write
		if err := writeBuffered(data); err != nil {
			t.Errorf("Buffered write %d failed: %v", i, err)
		}
	}

	// Final flush
	if err := flush(); err != nil {
		t.Errorf("Final flush failed: %v", err)
	}

	rotator.Close()

	// Verify file was created and contains data
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatalf("Log file was not created: %s", testFile)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "Buffered write test message") {
		t.Error("Log file does not contain expected content")
	}

	t.Logf("✅ Buffered writer integration successful - log file size: %d bytes", len(content))
}

// TestZeroCopyIntegration tests the WriteOwned zero-copy API
func TestZeroCopyIntegration(t *testing.T) {
	testFile := generateTestFile("zerocopy")
	defer cleanupTestFile(testFile)

	rotator := &lethe.Logger{
		Filename:   testFile,
		MaxSizeStr: "10KB",
		Async:      true,
		BufferSize: 128,
	}
	defer rotator.Close()

	// Test ownership transfer
	for i := 0; i < 20; i++ {
		// Create data buffer (simulating framework that can transfer ownership)
		data := fmt.Sprintf("Zero-copy message %d with content for ownership transfer\n", i)
		buffer := make([]byte, len(data))
		copy(buffer, data)

		// Transfer ownership to Lethe
		n, err := rotator.WriteOwned(buffer)
		if err != nil {
			t.Errorf("WriteOwned %d failed: %v", i, err)
		}
		if n != len(buffer) {
			t.Errorf("Expected %d bytes written, got %d", len(buffer), n)
		}
	}

	// Allow processing
	time.Sleep(50 * time.Millisecond)

	// Verify stats
	stats := rotator.Stats()
	if stats.WriteCount == 0 {
		t.Error("Expected non-zero write count")
	}

	t.Logf("✅ Zero-copy integration successful - %d writes processed", stats.WriteCount)
}

// BenchmarkFrameworkPatterns benchmarks different integration patterns
func BenchmarkFrameworkPatterns(b *testing.B) {
	testFile := generateTestFile("benchmark")
	defer cleanupTestFile(testFile)

	rotator := &lethe.Logger{
		Filename:   testFile,
		MaxSizeStr: "100MB", // Large enough to avoid rotation during benchmark
		Async:      true,
		BufferSize: 1024,
	}
	defer rotator.Close()

	message := []byte("Benchmark integration test message with moderate length content\n")

	b.Run("DirectWrite", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				rotator.Write(message)
			}
		})
	})

	b.Run("WriteOwned", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				// Simulate ownership transfer
				buffer := make([]byte, len(message))
				copy(buffer, message)
				rotator.WriteOwned(buffer)
			}
		})
	})

	b.Run("StandardLog", func(b *testing.B) {
		// Test overhead of standard library integration
		originalOutput := log.Writer()
		log.SetOutput(rotator)
		defer log.SetOutput(originalOutput)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			log.Print("Benchmark message via standard library")
		}
	})
}

func TestMaxAgeStrIntegration(t *testing.T) {
	// Test MaxAgeStr in a realistic integration scenario
	tempFile := filepath.Join(os.TempDir(), "maxagestr_integration_test.log")

	// Cleanup first
	os.Remove(tempFile)
	matches, _ := filepath.Glob(tempFile + ".*")
	for _, match := range matches {
		os.Remove(match)
	}

	// Create logger with string-based age configuration
	config := &lethe.LoggerConfig{
		Filename:   tempFile,
		MaxSizeStr: "10MB",  // Large enough to not trigger size rotation
		MaxAgeStr:  "200ms", // Very short for testing - normally would be "7d" or "24h"
		MaxBackups: 3,
		Compress:   false, // Disable for easier testing
		Async:      false, // Synchronous for predictable testing
	}

	logger, err := lethe.NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Write initial log entry
	_, err = logger.Write([]byte("Initial entry - should be in main file\n"))
	if err != nil {
		t.Fatalf("Failed to write initial entry: %v", err)
	}

	// Wait for MaxAge to trigger rotation
	time.Sleep(250 * time.Millisecond)

	// Write second entry, which should trigger rotation due to age
	_, err = logger.Write([]byte("Second entry - should trigger rotation due to MaxAgeStr\n"))
	if err != nil {
		t.Fatalf("Failed to write second entry: %v", err)
	}

	// Allow some time for background rotation to complete
	time.Sleep(100 * time.Millisecond)

	// Check that rotation occurred
	matches, err = filepath.Glob(tempFile + ".*")
	if err != nil {
		t.Fatalf("Failed to glob backup files: %v", err)
	}

	if len(matches) == 0 {
		t.Errorf("Expected at least one backup file due to MaxAgeStr rotation, but found none")
		t.Logf("Checking if main file exists:")
		if info, err := os.Stat(tempFile); err == nil {
			t.Logf("Main file exists with size: %d bytes", info.Size())
		} else {
			t.Logf("Main file does not exist: %v", err)
		}
	} else {
		t.Logf("Successfully created %d backup file(s) due to MaxAgeStr rotation", len(matches))
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil {
				t.Logf("Backup file: %s (size: %d bytes)", match, info.Size())
			}
		}
	}

	// Verify main file still exists (it may be empty if the last write triggered rotation)
	if _, err := os.Stat(tempFile); err != nil {
		t.Errorf("Main log file should still exist after rotation: %v", err)
	}

	// Test with more realistic configuration
	t.Run("realistic_config", func(t *testing.T) {
		realisticFile := filepath.Join(os.TempDir(), "realistic_maxagestr_test.log")
		defer os.Remove(realisticFile)

		realisticConfig := &lethe.LoggerConfig{
			Filename:   realisticFile,
			MaxSizeStr: "100MB", // 100 megabytes
			MaxAgeStr:  "7d",    // 7 days - realistic retention period
			MaxBackups: 10,      // Keep 10 backup files
			Compress:   true,    // Enable compression for space efficiency
			Async:      true,    // Enable async for better performance
		}

		realisticLogger, err := lethe.NewWithConfig(realisticConfig)
		if err != nil {
			t.Fatalf("Failed to create realistic logger: %v", err)
		}

		// Verify MaxAge was parsed correctly (7 days = 168 hours)
		expectedDuration := 7 * 24 * time.Hour
		if realisticLogger.MaxAge != expectedDuration {
			t.Errorf("Expected MaxAge to be %v, got %v", expectedDuration, realisticLogger.MaxAge)
		}

		// Write a few entries to verify it works
		for i := 0; i < 5; i++ {
			msg := fmt.Sprintf("Realistic log entry #%d with timestamp %s\n",
				i+1, time.Now().Format(time.RFC3339))
			_, err = realisticLogger.Write([]byte(msg))
			if err != nil {
				t.Errorf("Failed to write entry %d: %v", i+1, err)
			}
		}

		realisticLogger.Close()
	})

	// Cleanup
	logger.Close()
	os.Remove(tempFile)
	for _, match := range matches {
		os.Remove(match)
	}
}

// TestCoverageImprovement tests specific functions to improve coverage
func TestCoverageImprovement(t *testing.T) {
	t.Run("ValidatePathLength_EdgeCases", func(t *testing.T) {
		// Test path validation edge cases
		tests := []struct {
			name        string
			path        string
			expectError bool
		}{
			{"ValidShortPath", "test.log", false},
			{"ValidMediumPath", filepath.Join(strings.Repeat("dir", 20), "test.log"), false},
			{"EmptyPath", "", false}, // Empty path might be valid in some contexts
		}

		// Add OS-specific tests
		if runtime.GOOS == "windows" {
			tests = append(tests, struct {
				name        string
				path        string
				expectError bool
			}{
				"WindowsLongPath",
				strings.Repeat("a", 300), // Exceeds Windows 260 char limit
				true,
			})
		} else {
			tests = append(tests, struct {
				name        string
				path        string
				expectError bool
			}{
				"UnixLongPath",
				strings.Repeat("a", 5000), // Exceeds Unix 4096 char limit
				true,
			})
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := lethe.ValidatePathLength(tt.path)
				if tt.expectError && err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}
				if !tt.expectError && err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}
			})
		}
	})

	t.Run("GetDefaultFileMode_OSSpecific", func(t *testing.T) {
		mode := lethe.GetDefaultFileMode()
		if mode == 0 {
			t.Error("File mode should not be zero")
		}
		t.Logf("Default file mode: %o", mode)
	})

	t.Run("SanitizeFilename_EdgeCases", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"normal.log", "normal.log"},
			{"test<>:\"|?*.log", "test_______.log"},
			{"", ""},
			{"valid-name.log", "valid-name.log"},
		}

		for _, tt := range tests {
			result := lethe.SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		}
	})

	t.Run("ParseSize_EdgeCases", func(t *testing.T) {
		tests := []struct {
			input    string
			expected int64
		}{
			{"1KB", 1024},
			{"1MB", 1024 * 1024},
			{"1GB", 1024 * 1024 * 1024},
			{"500", 500}, // Just number without suffix
			{"", 0},
			{"invalid", 0},
		}

		for _, tt := range tests {
			result, err := lethe.ParseSize(tt.input)
			if err != nil && tt.expected != 0 {
				t.Errorf("ParseSize(%q) returned error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("ParseSize(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		}
	})

	t.Run("ParseDuration_EdgeCases", func(t *testing.T) {
		tests := []struct {
			input    string
			expected time.Duration
		}{
			{"1s", time.Second},
			{"1m", time.Minute},
			{"1h", time.Hour},
			{"1d", 24 * time.Hour},
			{"7d", 7 * 24 * time.Hour},
			{"", 0},
			{"invalid", 0},
		}

		for _, tt := range tests {
			result, err := lethe.ParseDuration(tt.input)
			if err != nil && tt.expected != 0 {
				t.Errorf("ParseDuration(%q) returned error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("ParseDuration(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		}
	})
}

// TestAdvancedRotationScenarios tests advanced rotation scenarios for better coverage
func TestAdvancedRotationScenarios(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("CompressionAndCleanup", func(t *testing.T) {
		config := &lethe.LoggerConfig{
			Filename:   filepath.Join(tempDir, "compression_test.log"),
			MaxSizeStr: "1KB",
			MaxBackups: 2,
			MaxAgeStr:  "1s",
			Compress:   true,
			Async:      true,
		}

		logger, err := lethe.NewWithConfig(config)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer logger.Close()

		// Write data to trigger rotation, compression, and cleanup
		for i := 0; i < 20; i++ {
			msg := fmt.Sprintf("Compression test message %d with some content to fill the buffer\n", i)
			logger.Write([]byte(msg))
		}

		// Wait for background processing
		time.Sleep(200 * time.Millisecond)

		// Check for compressed files
		matches, err := filepath.Glob(filepath.Join(tempDir, "compression_test.log*"))
		if err != nil {
			t.Fatalf("Failed to glob files: %v", err)
		}

		t.Logf("Created %d files during compression test", len(matches))
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil {
				t.Logf("File: %s (size: %d bytes)", match, info.Size())
			}
		}
	})

	t.Run("BackpressurePolicies", func(t *testing.T) {
		policies := []string{"drop", "adaptive", "fallback"}

		for _, policy := range policies {
			t.Run(policy, func(t *testing.T) {
				config := &lethe.LoggerConfig{
					Filename:           filepath.Join(tempDir, fmt.Sprintf("backpressure_%s.log", policy)),
					MaxSizeStr:         "1KB",
					BackpressurePolicy: policy,
					BufferSize:         1, // Very small buffer to trigger backpressure
					Async:              true,
				}

				logger, err := lethe.NewWithConfig(config)
				if err != nil {
					t.Fatalf("Failed to create logger with policy %s: %v", policy, err)
				}
				defer logger.Close()

				// Write data rapidly to trigger backpressure
				for i := 0; i < 100; i++ {
					msg := fmt.Sprintf("Backpressure test message %d for policy %s\n", i, policy)
					logger.Write([]byte(msg))
				}

				// Wait for processing
				time.Sleep(100 * time.Millisecond)

				// Check stats
				stats := logger.Stats()
				t.Logf("Policy %s: %d writes", policy, stats.WriteCount)
			})
		}
	})

	t.Run("ConcurrentWrites", func(t *testing.T) {
		config := &lethe.LoggerConfig{
			Filename:   filepath.Join(tempDir, "concurrent_test.log"),
			MaxSizeStr: "10KB",
			Async:      true,
			BufferSize: 64,
		}

		logger, err := lethe.NewWithConfig(config)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}
		defer logger.Close()

		// Test concurrent writes
		var wg sync.WaitGroup
		numGoroutines := 10
		writesPerGoroutine := 50

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < writesPerGoroutine; j++ {
					msg := fmt.Sprintf("Concurrent write from goroutine %d, message %d\n", id, j)
					logger.Write([]byte(msg))
				}
			}(i)
		}

		wg.Wait()
		time.Sleep(100 * time.Millisecond)

		// Verify stats
		stats := logger.Stats()
		if stats.WriteCount == 0 {
			t.Error("Expected non-zero write count")
		}
		t.Logf("Concurrent test: %d total writes", stats.WriteCount)
	})
}

// TestErrorHandlingScenarios tests various error handling scenarios
func TestErrorHandlingScenarios(t *testing.T) {
	t.Run("InvalidConfigurations", func(t *testing.T) {
		// Test with empty filename
		config := &lethe.LoggerConfig{
			Filename: "",
		}
		_, err := lethe.NewWithConfig(config)
		if err == nil {
			t.Error("Expected error with empty filename")
		}

		// Test with invalid file mode
		config = &lethe.LoggerConfig{
			Filename: filepath.Join(t.TempDir(), "test.log"),
			FileMode: 0,
		}
		logger, err := lethe.NewWithConfig(config)
		if err != nil {
			t.Fatalf("Failed to create logger with invalid file mode: %v", err)
		}
		defer logger.Close()

		// Test writing to logger with invalid config
		_, err = logger.Write([]byte("test"))
		if err != nil {
			t.Errorf("Write should handle invalid file mode gracefully: %v", err)
		}
	})

	t.Run("FileSystemErrors", func(t *testing.T) {
		// Test with invalid path (directory instead of file)
		invalidPath := t.TempDir() // This is a directory, not a file
		config := &lethe.LoggerConfig{
			Filename: invalidPath,
		}

		// This should handle the error gracefully
		logger, err := lethe.NewWithConfig(config)
		if err != nil {
			// Expected error for invalid path
			return
		}
		defer logger.Close()

		// Try to write - should handle gracefully
		_, err = logger.Write([]byte("test"))
		if err != nil {
			// This is expected for invalid paths
			return
		}
	})
}
