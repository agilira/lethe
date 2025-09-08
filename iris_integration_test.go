// iris_integration_test.go: Tests for Iris integration
//
// Copyright (c) 2025 AGILira
// Series: Lethe - Iris Integration Tests
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewIrisWriter(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test_iris.log")

	writer := NewIrisWriter(logFile, &Logger{
		MaxSizeStr: "10MB",
		MaxBackups: 3,
		Compress:   false,
	})

	if writer == nil {
		t.Fatal("NewIrisWriter returned nil")
	}

	// Test that it implements the enhanced interface
	if writer.GetOptimalBufferSize() <= 0 {
		t.Error("GetOptimalBufferSize should return positive value")
	}

	if !writer.SupportsHotReload() {
		t.Error("Should support hot reload")
	}

	// Test basic writing
	testData := []byte("Test Iris integration\n")
	n, err := writer.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write returned %d bytes, expected %d", n, len(testData))
	}

	// Test WriteOwned (zero-copy path)
	n, err = writer.WriteOwned(testData)
	if err != nil {
		t.Fatalf("WriteOwned failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("WriteOwned returned %d bytes, expected %d", n, len(testData))
	}

	// Test Sync
	if err := writer.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
	}

	// Close and verify file was created
	if err := writer.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify file exists and has content
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}

	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Error("Log file is empty")
	}

	t.Logf("Iris integration test successful, wrote %d bytes", len(content))
}

func TestQuickStart(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "quickstart.log")

	writer := QuickStart(logFile)
	if writer == nil {
		t.Fatal("QuickStart returned nil")
	}

	// Verify defaults
	logger := writer.GetLogger()
	if logger.MaxSizeStr != "100MB" {
		t.Errorf("Expected MaxSizeStr '100MB', got '%s'", logger.MaxSizeStr)
	}

	if logger.MaxBackups != 5 {
		t.Errorf("Expected MaxBackups 5, got %d", logger.MaxBackups)
	}

	if !logger.Compress {
		t.Error("Expected compression to be enabled")
	}

	if !logger.Async {
		t.Error("Expected async mode to be enabled")
	}

	// Test writing
	testData := []byte("QuickStart test\n")
	n, err := writer.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Write returned %d bytes, expected %d", n, len(testData))
	}

	writer.Close()
	t.Log("QuickStart test successful")
}

func TestIrisIntegrationInterface(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "interface_test.log")

	writer := QuickStart(logFile)
	defer writer.Close()

	// Test all interface methods

	// GetOptimalBufferSize
	bufferSize := writer.GetOptimalBufferSize()
	if bufferSize <= 0 {
		t.Errorf("GetOptimalBufferSize returned %d, expected positive", bufferSize)
	}

	// SupportsHotReload
	if !writer.SupportsHotReload() {
		t.Error("Should support hot reload")
	}

	// GetLogger
	logger := writer.GetLogger()
	if logger == nil {
		t.Error("GetLogger returned nil")
	}

	// Write
	data := []byte("Interface test\n")
	n, err := writer.Write(data)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d, expected %d", n, len(data))
	}

	// WriteOwned
	n, err = writer.WriteOwned(data)
	if err != nil {
		t.Errorf("WriteOwned failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("WriteOwned returned %d, expected %d", n, len(data))
	}

	// Sync
	if err := writer.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
	}

	t.Log("Interface test successful")
}

func TestIrisIntegrationPerformance(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "perf_test.log")

	writer := QuickStart(logFile)
	defer writer.Close()

	// Write many messages to test performance
	testData := []byte("Performance test message\n")
	startTime := time.Now()

	const numWrites = 1000
	for i := 0; i < numWrites; i++ {
		if i%2 == 0 {
			writer.Write(testData)
		} else {
			writer.WriteOwned(testData) // Test zero-copy path
		}
	}

	duration := time.Since(startTime)

	// Calculate throughput
	bytesWritten := int64(len(testData) * numWrites)
	throughput := float64(bytesWritten) / duration.Seconds() / 1024 / 1024 // MB/s

	t.Logf("Performance test: %d writes in %v (%.2f MB/s)",
		numWrites, duration, throughput)

	if throughput < 1.0 {
		t.Logf("Warning: Low throughput (%.2f MB/s), but test passes", throughput)
	}
}
