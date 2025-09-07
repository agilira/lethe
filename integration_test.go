// integration_test.go: Integration test for hot reload functionality
//
// This test demonstrates end-to-end hot reload functionality
//
// Copyright (c) 2025 AGILira
// Series: an AGILira fragment
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHotReloadIntegration(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "integration.log")
	configFile := filepath.Join(tempDir, "config.json")
	auditFile := filepath.Join(tempDir, "lethe-config-audit.jsonl")

	// Create initial configuration
	initialConfig := LoggerConfig{
		Filename:           logFile,
		MaxSizeStr:         "1KB", // Very small for testing rotation
		MaxBackups:         3,
		Compress:           false,
		LocalTime:          true,
		BackpressurePolicy: "fallback",
	}

	writeConfig := func(config LoggerConfig) {
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal config: %v", err)
		}
		if err := os.WriteFile(configFile, data, 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}
	}

	// Write initial config
	writeConfig(initialConfig)

	// Create logger
	logger, err := NewWithDefaults(logFile)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Track errors
	var errorCount int
	logger.ErrorCallback = func(operation string, err error) {
		t.Logf("Logger error (%s): %v", operation, err)
		errorCount++
	}

	// Enable hot reload
	watcher, err := EnableDynamicConfig(logger, configFile)
	if err != nil {
		t.Fatalf("Failed to enable dynamic config: %v", err)
	}
	defer watcher.Stop()

	t.Log("Initial setup complete")

	// Wait for initial config to be applied
	time.Sleep(100 * time.Millisecond)

	// Verify initial configuration
	if logger.MaxBackups != 3 {
		t.Errorf("Initial MaxBackups should be 3, got %d", logger.MaxBackups)
	}
	if logger.Compress != false {
		t.Errorf("Initial Compress should be false, got %t", logger.Compress)
	}

	// Write some data to test rotation
	testData := "This is a test log entry that should trigger rotation due to small size limit\n"
	for i := 0; i < 5; i++ {
		msg := fmt.Sprintf("[%d] %s", i, testData)
		if _, err := logger.Write([]byte(msg)); err != nil {
			t.Errorf("Failed to write test data: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Small delay to see rotation
	}

	t.Log("Initial writes complete")

	// Update configuration - change compression and backup count
	updatedConfig := LoggerConfig{
		Filename:           logFile,
		MaxSizeStr:         "2KB", // Slightly larger
		MaxBackups:         5,     // More backups
		Compress:           true,  // Enable compression
		LocalTime:          false, // Change timezone
		BackpressurePolicy: "adaptive",
	}

	t.Log("Updating configuration...")
	writeConfig(updatedConfig)

	// Wait for hot reload to detect and apply changes
	time.Sleep(3 * time.Second)

	// Verify configuration was updated
	if logger.MaxBackups != 5 {
		t.Errorf("Updated MaxBackups should be 5, got %d", logger.MaxBackups)
	}
	if logger.Compress != true {
		t.Errorf("Updated Compress should be true, got %t", logger.Compress)
	}
	if logger.LocalTime != false {
		t.Errorf("Updated LocalTime should be false, got %t", logger.LocalTime)
	}
	if logger.BackpressurePolicy != "adaptive" {
		t.Errorf("Updated BackpressurePolicy should be 'adaptive', got %s", logger.BackpressurePolicy)
	}

	t.Log("Configuration updated successfully")

	// Write more data to test new settings
	for i := 5; i < 10; i++ {
		msg := fmt.Sprintf("[%d] %s", i, testData)
		if _, err := logger.Write([]byte(msg)); err != nil {
			t.Errorf("Failed to write test data after config update: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Check if audit file was created
	if _, err := os.Stat(auditFile); err == nil {
		t.Log("Audit file created successfully")
		auditData, err := os.ReadFile(auditFile)
		if err == nil && len(auditData) > 0 {
			t.Logf("Audit file content: %s", string(auditData))
		}
	} else {
		t.Logf("Audit file not found (this may be normal): %v", err)
	}

	// Verify last config
	lastConfig := watcher.GetLastConfig()
	if lastConfig == nil {
		t.Error("Last config should not be nil")
	} else if lastConfig.MaxBackups != 5 {
		t.Errorf("Last config MaxBackups should be 5, got %d", lastConfig.MaxBackups)
	}

	// Final stats
	stats := logger.Stats()
	t.Logf("Final stats: Writes=%d, Size=%d, Rotations=%d, Errors=%d",
		stats.WriteCount, stats.CurrentFileSize, stats.RotationCount, errorCount)

	// Check that we have log files
	files, err := filepath.Glob(filepath.Join(tempDir, "*"))
	if err != nil {
		t.Errorf("Failed to list files: %v", err)
	} else {
		t.Logf("Created files: %v", files)
	}

	if errorCount > 0 {
		t.Logf("Warning: %d errors occurred during test", errorCount)
	}

	t.Log("Integration test completed successfully")
}
