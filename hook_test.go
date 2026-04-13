// hook_test.go: Tests for pre-write hook functionality
//
// Copyright (c) 2025 AGILira
// Series: an AGILira fragment
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// =============================================================================
// TIER-1 AUDIT: Pre-Write Hook Tests
// =============================================================================
//
// REQUIREMENT: Audit logging needs to sign data before writing.
// A pre-write hook enables transparent signing/encryption without
// modifying the caller's code.
//
// Use cases:
// - HMAC signing for tamper-evidence
// - Encryption for sensitive logs
// - Canonicalization for deterministic serialization
// - Metrics collection
//
// =============================================================================

// TestPreWriteHook_Basic verifies hook is called and transforms data.
func TestPreWriteHook_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	// Hook that uppercases all data
	config := &LoggerConfig{
		Filename: logFile,
		PreWriteHook: func(data []byte) ([]byte, error) {
			return bytes.ToUpper(data), nil
		},
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	input := []byte("hello world\n")
	expected := []byte("HELLO WORLD\n")

	n, err := logger.Write(input)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(expected) {
		t.Errorf("Expected %d bytes written, got %d", len(expected), n)
	}

	// Verify transformed data was written
	_ = logger.Close()
	content, _ := os.ReadFile(logFile)
	if string(content) != string(expected) {
		t.Errorf("File content mismatch: got %q, want %q", content, expected)
	}
}

// TestPreWriteHook_HMAC verifies HMAC signing use case.
func TestPreWriteHook_HMAC(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "audit.log")

	signingKey := []byte("super-secret-key-32-bytes-long!!")

	// Hook that appends HMAC signature to each line
	config := &LoggerConfig{
		Filename: logFile,
		PreWriteHook: func(data []byte) ([]byte, error) {
			// Strip trailing newline for signing
			content := bytes.TrimSuffix(data, []byte("\n"))

			// Compute HMAC-SHA256
			mac := hmac.New(sha256.New, signingKey)
			mac.Write(content)
			sig := hex.EncodeToString(mac.Sum(nil))

			// Return: original content + |sig + newline
			return append(content, []byte("|"+sig+"\n")...), nil
		},
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Write audit entry
	entry := []byte(`{"action":"test","actor":"user1"}` + "\n")
	_, err = logger.Write(entry)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}

	// Verify signature was appended
	_ = logger.Close()
	content, _ := os.ReadFile(logFile)

	// Should contain pipe separator
	if !bytes.Contains(content, []byte("|")) {
		t.Error("Expected HMAC signature with | separator")
	}

	// Verify signature is valid
	parts := bytes.SplitN(bytes.TrimSuffix(content, []byte("\n")), []byte("|"), 2)
	if len(parts) != 2 {
		t.Fatalf("Invalid format: %s", content)
	}

	// Recompute signature
	mac := hmac.New(sha256.New, signingKey)
	mac.Write(parts[0])
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if string(parts[1]) != expectedSig {
		t.Errorf("Signature mismatch:\n  got:  %s\n  want: %s", parts[1], expectedSig)
	}
}

// TestPreWriteHook_Error verifies hook errors are propagated.
func TestPreWriteHook_Error(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	hookError := errors.New("hook failed: invalid data")

	config := &LoggerConfig{
		Filename: logFile,
		PreWriteHook: func(data []byte) ([]byte, error) {
			// Reject data containing "bad"
			if bytes.Contains(data, []byte("bad")) {
				return nil, hookError
			}
			return data, nil
		},
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Good data should work
	_, err = logger.Write([]byte("good data\n"))
	if err != nil {
		t.Errorf("Good data rejected: %v", err)
	}

	// Bad data should fail
	_, err = logger.Write([]byte("bad data\n"))
	if err == nil {
		t.Error("Expected error for bad data, got nil")
	}
	if !strings.Contains(err.Error(), "hook failed") {
		t.Errorf("Expected hook error, got: %v", err)
	}
}

// TestPreWriteHook_NoHook verifies default behavior without hook.
func TestPreWriteHook_NoHook(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	// No hook configured
	config := &LoggerConfig{
		Filename: logFile,
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	data := []byte("unchanged data\n")
	_, err = logger.Write(data)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}

	// Verify data unchanged
	_ = logger.Close()
	content, _ := os.ReadFile(logFile)
	if string(content) != string(data) {
		t.Errorf("Data modified without hook: got %q, want %q", content, data)
	}
}

// TestPreWriteHook_WithAsync verifies hook works in async mode.
func TestPreWriteHook_WithAsync(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	var hookCalls atomic.Int32

	config := &LoggerConfig{
		Filename: logFile,
		Async:    true,
		PreWriteHook: func(data []byte) ([]byte, error) {
			hookCalls.Add(1)
			return bytes.ToUpper(data), nil
		},
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Write multiple entries
	for i := 0; i < 10; i++ {
		if _, err := logger.Write([]byte("entry\n")); err != nil {
			t.Errorf("Write failed on iteration %d: %v", i, err)
		}
	}

	// Wait for async processing
	_ = logger.Close()

	// Verify hook was called for each write
	if hookCalls.Load() != 10 {
		t.Errorf("Expected 10 hook calls, got %d", hookCalls.Load())
	}

	// Verify content is transformed
	content, _ := os.ReadFile(logFile)
	if !bytes.Contains(content, []byte("ENTRY")) {
		t.Error("Content not transformed by hook in async mode")
	}
}

// TestPreWriteHook_WithWriteOwned verifies hook works with zero-copy.
func TestPreWriteHook_WithWriteOwned(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	config := &LoggerConfig{
		Filename: logFile,
		PreWriteHook: func(data []byte) ([]byte, error) {
			return bytes.ToUpper(data), nil
		},
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Use WriteOwned
	data := []byte("owned data\n")
	n, err := logger.WriteOwned(data)
	if err != nil {
		t.Errorf("WriteOwned failed: %v", err)
	}
	if n != len("OWNED DATA\n") {
		t.Errorf("Unexpected byte count: %d", n)
	}

	_ = logger.Close()
	content, _ := os.ReadFile(logFile)
	if string(content) != "OWNED DATA\n" {
		t.Errorf("WriteOwned hook not applied: %q", content)
	}
}

// TestPreWriteHook_WithWriteContext verifies hook works with context.
func TestPreWriteHook_WithWriteContext(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	config := &LoggerConfig{
		Filename: logFile,
		PreWriteHook: func(data []byte) ([]byte, error) {
			return append([]byte("[PREFIX]"), data...), nil
		},
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Use WriteContext
	ctx := t.Context()
	n, err := logger.WriteContext(ctx, []byte("context data\n"))
	if err != nil {
		t.Errorf("WriteContext failed: %v", err)
	}
	if n != len("[PREFIX]context data\n") {
		t.Errorf("Unexpected byte count: %d", n)
	}

	_ = logger.Close()
	content, _ := os.ReadFile(logFile)
	if string(content) != "[PREFIX]context data\n" {
		t.Errorf("WriteContext hook not applied: %q", content)
	}
}
