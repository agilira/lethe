// hot_reload_test.go: Tests for dynamic configuration hot reload
//
// Copyright (c) 2025 AGILira
// Series: an AGILira fragment
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDynamicConfigWatcher(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	configFile := filepath.Join(tempDir, "test-config.json")

	// Create initial config
	initialConfig := LoggerConfig{
		Filename:   logFile,
		MaxSizeStr: "1MB",
		MaxAgeStr:  "1h",
		MaxBackups: 5,
		Compress:   false,
		LocalTime:  true,
	}

	// Write initial config to file
	configData, err := json.MarshalIndent(initialConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal initial config: %v", err)
	}

	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Create logger
	logger, err := NewWithDefaults(logFile)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Create watcher
	watcher, err := NewDynamicConfigWatcher(configFile, logger)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer func() {
		if err := watcher.Stop(); err != nil {
			t.Logf("Warning: Failed to stop watcher: %v", err)
		}
	}()

	// Start watcher
	if err := watcher.Start(); err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}

	// Verify initial state
	if !watcher.IsRunning() {
		t.Fatal("Watcher should be running")
	}

	// Check initial configuration was applied
	time.Sleep(100 * time.Millisecond) // Give time for initial config to apply
	if logger.MaxBackups != 5 {
		t.Errorf("Expected MaxBackups to be 5, got %d", logger.MaxBackups)
	}
	if logger.Compress != false {
		t.Errorf("Expected Compress to be false, got %t", logger.Compress)
	}

	// Update configuration
	updatedConfig := LoggerConfig{
		Filename:   logFile,
		MaxSizeStr: "2MB",
		MaxAgeStr:  "2h",
		MaxBackups: 10,
		Compress:   true,
		LocalTime:  false,
	}

	updatedConfigData, err := json.MarshalIndent(updatedConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal updated config: %v", err)
	}

	// Write updated config
	if err := os.WriteFile(configFile, updatedConfigData, 0644); err != nil {
		t.Fatalf("Failed to write updated config: %v", err)
	}

	// Wait for the change to be detected and applied
	time.Sleep(3 * time.Second) // Give Argus time to detect and apply changes

	// Verify configuration was updated
	if logger.MaxBackups != 10 {
		t.Errorf("Expected MaxBackups to be updated to 10, got %d", logger.MaxBackups)
	}
	if logger.Compress != true {
		t.Errorf("Expected Compress to be updated to true, got %t", logger.Compress)
	}
	if logger.LocalTime != false {
		t.Errorf("Expected LocalTime to be updated to false, got %t", logger.LocalTime)
	}

	// Check that maxSizeBytes was updated correctly
	expectedSize := int64(2 * 1024 * 1024) // 2MB in bytes
	if logger.maxSizeBytes != expectedSize {
		t.Errorf("Expected maxSizeBytes to be %d, got %d", expectedSize, logger.maxSizeBytes)
	}

	// Verify last config is stored
	lastConfig := watcher.GetLastConfig()
	if lastConfig == nil {
		t.Fatal("Last config should not be nil")
	}
	if lastConfig.MaxBackups != 10 {
		t.Errorf("Expected last config MaxBackups to be 10, got %d", lastConfig.MaxBackups)
	}
}

func TestDynamicConfigWatcherErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	configFile := filepath.Join(tempDir, "nonexistent-config.json")

	// Create logger
	logger, err := NewWithDefaults(logFile)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Try to create watcher with non-existent config file
	_, err = NewDynamicConfigWatcher(configFile, logger)
	if err == nil {
		t.Fatal("Expected error when config file doesn't exist")
	}
}

func TestCreateSampleConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "sample-config.json")

	// Create sample config
	if err := CreateSampleConfig(configFile); err != nil {
		t.Fatalf("Failed to create sample config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configFile); err != nil {
		t.Fatalf("Sample config file was not created: %v", err)
	}

	// Verify file content contains expected JSON
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read sample config: %v", err)
	}

	content := string(data)
	if !contains(content, "max_size_str") || !contains(content, "100MB") {
		t.Error("Sample config should contain max_size_str with 100MB")
	}
	if !contains(content, "max_age_str") || !contains(content, "7d") {
		t.Error("Sample config should contain max_age_str with 7d")
	}
}

func TestEnableDynamicConfig(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	configFile := filepath.Join(tempDir, "test-config.json")

	// Create config file
	config := LoggerConfig{
		Filename:   logFile,
		MaxSizeStr: "5MB",
		MaxBackups: 3,
		Compress:   true,
	}

	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create logger
	logger, err := NewWithDefaults(logFile)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Enable dynamic config (convenience function)
	watcher, err := EnableDynamicConfig(logger, configFile)
	if err != nil {
		t.Fatalf("Failed to enable dynamic config: %v", err)
	}
	defer func() {
		if err := watcher.Stop(); err != nil {
			t.Logf("Warning: Failed to stop watcher: %v", err)
		}
	}()

	// Verify watcher is running
	if !watcher.IsRunning() {
		t.Fatal("Watcher should be running after EnableDynamicConfig")
	}

	// Wait a bit for config to be applied
	time.Sleep(100 * time.Millisecond)

	// Verify config was applied
	if logger.MaxBackups != 3 {
		t.Errorf("Expected MaxBackups to be 3, got %d", logger.MaxBackups)
	}
	if logger.Compress != true {
		t.Errorf("Expected Compress to be true, got %t", logger.Compress)
	}
}

func TestWatcherStartStop(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")
	configFile := filepath.Join(tempDir, "test-config.json")

	// Create config file
	if err := CreateSampleConfig(configFile); err != nil {
		t.Fatalf("Failed to create sample config: %v", err)
	}

	// Create logger
	logger, err := NewWithDefaults(logFile)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Create watcher
	watcher, err := NewDynamicConfigWatcher(configFile, logger)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Initially not running
	if watcher.IsRunning() {
		t.Fatal("Watcher should not be running initially")
	}

	// Start watcher
	if err := watcher.Start(); err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}

	// Should be running now
	if !watcher.IsRunning() {
		t.Fatal("Watcher should be running after start")
	}

	// Try to start again (should fail)
	if err := watcher.Start(); err == nil {
		t.Fatal("Starting an already running watcher should fail")
	}

	// Stop watcher
	if err := watcher.Stop(); err != nil {
		t.Fatalf("Failed to stop watcher: %v", err)
	}

	// Should not be running now
	if watcher.IsRunning() {
		t.Fatal("Watcher should not be running after stop")
	}

	// Try to stop again (should fail)
	if err := watcher.Stop(); err == nil {
		t.Fatal("Stopping an already stopped watcher should fail")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexByte(s, substr) >= 0)
}

func indexByte(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
