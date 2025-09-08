// demo_test.go: Live demonstration of hot reload functionality
//
// This test shows hot reload working in real-time and can be used
// for documentation and demonstration purposes.
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

func TestHotReloadDemo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping hot reload demo in short mode")
	}

	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "demo.log")
	configFile := filepath.Join(tempDir, "demo-config.json")

	t.Logf("   Starting Lethe Hot Reload Demo")
	t.Logf("   Log file: %s", logFile)
	t.Logf("   Config file: %s", configFile)

	// Create initial configuration
	initialConfig := LoggerConfig{
		Filename:           logFile,
		MaxSizeStr:         "5KB", // Small for demo
		MaxBackups:         3,
		Compress:           false,
		LocalTime:          true,
		BackpressurePolicy: "fallback",
	}

	// Write initial config
	configData, err := json.MarshalIndent(initialConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal initial config: %v", err)
	}
	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	t.Logf(" Created initial config: MaxBackups=%d, Compress=%t",
		initialConfig.MaxBackups, initialConfig.Compress)

	// Create logger
	logger, err := NewWithDefaults(logFile)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Enable hot reload
	watcher, err := EnableDynamicConfig(logger, configFile)
	if err != nil {
		t.Fatalf("Failed to enable dynamic config: %v", err)
	}
	defer func() {
		if err := watcher.Stop(); err != nil {
			t.Logf("Warning: Failed to stop watcher: %v", err)
		}
	}()

	t.Logf("✅ Hot reload enabled and running")

	// Wait for initial config to be applied
	time.Sleep(200 * time.Millisecond)

	// Verify initial state
	if logger.MaxBackups != 3 {
		t.Errorf("Initial MaxBackups should be 3, got %d", logger.MaxBackups)
	}
	if logger.Compress != false {
		t.Errorf("Initial Compress should be false, got %t", logger.Compress)
	}

	t.Logf("✅ Initial configuration verified")

	// Write some test data using WriteOwned for optimal performance
	for i := 1; i <= 3; i++ {
		msg := fmt.Sprintf("Initial test message #%d - time: %s\n",
			i, time.Now().Format("15:04:05"))
		if _, err := logger.WriteOwned([]byte(msg)); err != nil {
			t.Errorf("Failed to write initial test data: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Logf(" Wrote initial test messages")

	// Update configuration - simulate user editing the config file
	updatedConfig := LoggerConfig{
		Filename:           logFile,
		MaxSizeStr:         "10KB", // Increased size limit
		MaxBackups:         7,      // More backups
		Compress:           true,   // Enable compression
		LocalTime:          false,  // Change timezone
		BackpressurePolicy: "adaptive",
	}

	updatedConfigData, err := json.MarshalIndent(updatedConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal updated config: %v", err)
	}

	t.Logf(" Updating configuration...")
	t.Logf("   MaxBackups: %d -> %d", initialConfig.MaxBackups, updatedConfig.MaxBackups)
	t.Logf("   Compress: %t -> %t", initialConfig.Compress, updatedConfig.Compress)
	t.Logf("   MaxSizeStr: %s -> %s", initialConfig.MaxSizeStr, updatedConfig.MaxSizeStr)

	// Write updated config (simulates user editing the file)
	if err := os.WriteFile(configFile, updatedConfigData, 0644); err != nil {
		t.Fatalf("Failed to write updated config: %v", err)
	}

	// Wait for hot reload to detect and apply changes
	t.Logf(" Waiting for hot reload to detect changes...")
	time.Sleep(3 * time.Second)

	// Verify configuration was updated
	if logger.MaxBackups != 7 {
		t.Errorf("Updated MaxBackups should be 7, got %d", logger.MaxBackups)
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

	t.Logf("✅ Configuration updated successfully!")
	t.Logf("   New MaxBackups: %d", logger.MaxBackups)
	t.Logf("   New Compress: %t", logger.Compress)
	t.Logf("   New BackpressurePolicy: %s", logger.BackpressurePolicy)

	// Write more test data with new settings using WriteOwned
	for i := 4; i <= 6; i++ {
		msg := fmt.Sprintf("Updated test message #%d - time: %s\n",
			i, time.Now().Format("15:04:05"))
		if _, err := logger.WriteOwned([]byte(msg)); err != nil {
			t.Errorf("Failed to write updated test data: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Logf(" Wrote test messages with updated configuration")

	// Verify last config is accessible
	lastConfig := watcher.GetLastConfig()
	if lastConfig == nil {
		t.Error("Last config should not be nil")
	} else {
		t.Logf(" Last applied config: MaxBackups=%d, Compress=%t",
			lastConfig.MaxBackups, lastConfig.Compress)
	}

	// Final stats
	stats := logger.Stats()
	t.Logf(" Final Statistics:")
	t.Logf("   Total writes: %d", stats.WriteCount)
	t.Logf("   Current file size: %d bytes", stats.CurrentFileSize)
	t.Logf("   Rotations performed: %d", stats.RotationCount)
	t.Logf("   Average latency: %d ns", stats.AvgLatencyNs)

	// Check created files
	files, err := filepath.Glob(filepath.Join(tempDir, "*"))
	if err == nil {
		t.Logf(" Created files:")
		for _, file := range files {
			info, _ := os.Stat(file)
			if info != nil {
				t.Logf("   %s (%d bytes)", filepath.Base(file), info.Size())
			}
		}
	}

	t.Logf(" Hot reload demo completed successfully!")
}
