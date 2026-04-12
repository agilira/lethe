// context_test.go: Tests for context-aware write APIs
//
// Copyright (c) 2025 AGILira
// Series: an AGILira fragment
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// =============================================================================
// TIER-1 AUDIT: Context-Aware Write API Tests
// =============================================================================
//
// REQUIREMENT: Audit logging must respect context cancellation.
// If context is cancelled, writes should fail immediately rather than block.
//
// This enables:
// - Timeout control on audit writes
// - Graceful shutdown without hanging
// - Request-scoped cancellation propagation
//
// =============================================================================

// TestWriteContext_Success verifies WriteContext works with valid context.
func TestWriteContext_Success(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logger, err := NewWithDefaults(logFile)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	}()

	ctx := context.Background()
	data := []byte("test log entry\n")

	n, err := logger.WriteContext(ctx, data)
	if err != nil {
		t.Errorf("WriteContext failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected %d bytes written, got %d", len(data), n)
	}

	// Verify data was written
	if err := logger.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
	content, _ := os.ReadFile(logFile)
	if string(content) != string(data) {
		t.Errorf("File content mismatch: got %q, want %q", content, data)
	}
}

// TestWriteContext_CancelledContext verifies immediate return on cancelled context.
func TestWriteContext_CancelledContext(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logger, err := NewWithDefaults(logFile)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	}()

	// Create already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	data := []byte("should not be written\n")

	n, err := logger.WriteContext(ctx, data)
	if err == nil {
		t.Error("Expected error for cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
	if n != 0 {
		t.Errorf("Expected 0 bytes written for cancelled context, got %d", n)
	}

	// Verify nothing was written
	if err := logger.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
	content, _ := os.ReadFile(logFile)
	if len(content) > 0 {
		t.Errorf("Data written despite cancelled context: %q", content)
	}
}

// TestWriteContext_Timeout verifies timeout is respected.
func TestWriteContext_Timeout(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logger, err := NewWithDefaults(logFile)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	}()

	// Create context that's already timed out
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	// Small delay to ensure timeout fires
	time.Sleep(time.Millisecond)

	data := []byte("should not be written\n")

	n, err := logger.WriteContext(ctx, data)
	if err == nil {
		t.Error("Expected error for timed out context, got nil")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got: %v", err)
	}
	if n != 0 {
		t.Errorf("Expected 0 bytes written for timed out context, got %d", n)
	}
}

// TestWriteOwnedContext_Success verifies WriteOwnedContext works.
func TestWriteOwnedContext_Success(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logger, err := NewWithDefaults(logFile)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	}()

	ctx := context.Background()
	data := []byte("zero-copy test entry\n")

	n, err := logger.WriteOwnedContext(ctx, data)
	if err != nil {
		t.Errorf("WriteOwnedContext failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected %d bytes written, got %d", len(data), n)
	}
}

// TestWriteOwnedContext_CancelledContext verifies zero-copy respects cancellation.
func TestWriteOwnedContext_CancelledContext(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logger, err := NewWithDefaults(logFile)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	}()

	// Create already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	data := []byte("should not be written\n")

	n, err := logger.WriteOwnedContext(ctx, data)
	if err == nil {
		t.Error("Expected error for cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}
	if n != 0 {
		t.Errorf("Expected 0 bytes written, got %d", n)
	}
}

// TestWriteContext_MultipleWrites verifies sequential context writes work.
func TestWriteContext_MultipleWrites(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	logger, err := NewWithDefaults(logFile)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() {
		if err := logger.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	}()

	ctx := context.Background()

	entries := []string{
		"entry 1\n",
		"entry 2\n",
		"entry 3\n",
	}

	for i, entry := range entries {
		n, err := logger.WriteContext(ctx, []byte(entry))
		if err != nil {
			t.Errorf("Write %d failed: %v", i, err)
		}
		if n != len(entry) {
			t.Errorf("Write %d: expected %d bytes, got %d", i, len(entry), n)
		}
	}

	// Verify all data written
	if err := logger.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
	content, _ := os.ReadFile(logFile)
	expected := "entry 1\nentry 2\nentry 3\n"
	if string(content) != expected {
		t.Errorf("Content mismatch: got %q, want %q", content, expected)
	}
}
