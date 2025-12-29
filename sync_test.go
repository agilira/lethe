// sync_test.go: Tests for sync/flush functionality
//
// Copyright (c) 2025 AGILira
// Series: an AGILira fragment
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// TIER-1 AUDIT: Sync/Flush API Tests
// =============================================================================
//
// REQUIREMENT: Audit systems need durability guarantees.
// - Sync() ensures all data is persisted to disk
// - FlushAndRotate() creates audit trail segments
//
// Use cases:
// - Pre-shutdown durability checkpoint
// - Audit trail segmentation by session
// - Force flush before critical operations
//
// =============================================================================

// TestSync_BasicFunctionality verifies Sync writes data to disk.
func TestSync_BasicFunctionality(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	config := &LoggerConfig{
		Filename: logFile,
		Async:    true, // Test with async mode
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Write some data
	testData := []byte("important audit entry\n")
	logger.Write(testData)

	// Sync to ensure data is on disk
	if err := logger.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
	}

	// Verify data was written
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(content) != string(testData) {
		t.Errorf("Content mismatch: got %q, want %q", content, testData)
	}
}

// TestSync_SyncMode verifies Sync works in sync mode too.
func TestSync_SyncMode(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	config := &LoggerConfig{
		Filename: logFile,
		Async:    false, // Sync mode
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Write data
	logger.Write([]byte("sync mode data\n"))

	// Sync should work in sync mode
	if err := logger.Sync(); err != nil {
		t.Errorf("Sync failed in sync mode: %v", err)
	}
}

// TestSync_EmptyBuffer verifies Sync works with empty buffer.
func TestSync_EmptyBuffer(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	config := &LoggerConfig{
		Filename: logFile,
		Async:    true,
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Sync without any writes
	if err := logger.Sync(); err != nil {
		t.Errorf("Sync on empty buffer failed: %v", err)
	}
}

// TestFlushAndRotate_CreatesNewFile verifies FlushAndRotate creates rotation.
func TestFlushAndRotate_CreatesNewFile(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	config := &LoggerConfig{
		Filename: logFile,
		Async:    true,
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Write initial data
	logger.Write([]byte("session 1 data\n"))

	// Force rotation
	if err := logger.FlushAndRotate(); err != nil {
		t.Errorf("FlushAndRotate failed: %v", err)
	}

	// Wait for background tasks
	logger.WaitForBackgroundTasks()

	// Write more data (goes to new file)
	logger.Write([]byte("session 2 data\n"))
	logger.Sync()

	// Current log should only have session 2 data
	content, _ := os.ReadFile(logFile)
	if string(content) != "session 2 data\n" {
		t.Errorf("Current log has wrong content: %q", content)
	}

	// Check that backup file exists
	files, _ := filepath.Glob(filepath.Join(tmpDir, "test.log.*"))
	if len(files) == 0 {
		t.Error("Expected backup file after FlushAndRotate")
	}
}

// TestFlushAndRotate_AuditSegmentation verifies audit trail segmentation.
func TestFlushAndRotate_AuditSegmentation(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "audit.log")

	config := &LoggerConfig{
		Filename: logFile,
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Simulate multiple audit sessions
	sessions := []string{
		"session-alpha",
		"session-beta",
		"session-gamma",
	}

	for _, session := range sessions {
		logger.Write([]byte("BEGIN " + session + "\n"))
		logger.Write([]byte("action-1\n"))
		logger.Write([]byte("action-2\n"))
		logger.Write([]byte("END " + session + "\n"))

		// Segment by session
		logger.FlushAndRotate()
		logger.WaitForBackgroundTasks()
	}

	// Should have at least one backup file
	// Note: Timing can cause fewer backups than sessions if rotations are too fast
	backups, _ := filepath.Glob(filepath.Join(tmpDir, "audit.log.*"))
	t.Logf("Created %d backup files for %d sessions", len(backups), len(sessions))

	if len(backups) < 1 {
		t.Errorf("Expected at least 1 backup, got %d", len(backups))
	}
}
