// onrotate_test.go: Tests for OnRotate forensic callback
//
// Validates that the OnRotate callback fires on every rotation with
// correct forensic data (sealed segment path, byte count, sequence).
// Also validates panic recovery in the callback path.
//
// Copyright (c) 2025 AGILira
// Series: an AGILira fragment
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestOnRotate_FiresOnSizeRotation verifies callback fires when file
// exceeds MaxSize and rotation is triggered.
func TestOnRotate_FiresOnSizeRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	var mu sync.Mutex
	var events []RotationEvent

	config := &LoggerConfig{
		Filename:   logFile,
		MaxSizeStr: "1KB", // Tiny size to force rotation
		OnRotate: func(event RotationEvent) {
			mu.Lock()
			defer mu.Unlock()
			events = append(events, event)
		},
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Write enough to trigger at least one rotation
	data := []byte(strings.Repeat("x", 200) + "\n")
	for i := 0; i < 20; i++ {
		if _, err := logger.Write(data); err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
	}

	logger.WaitForBackgroundTasks()

	mu.Lock()
	defer mu.Unlock()

	if len(events) == 0 {
		t.Fatal("OnRotate callback was never called")
	}

	ev := events[0]
	if ev.PreviousFile == "" {
		t.Error("PreviousFile is empty")
	}
	if ev.NewFile == "" {
		t.Error("NewFile is empty")
	}
	if ev.NewFile != logFile {
		t.Errorf("NewFile = %q, want %q", ev.NewFile, logFile)
	}
	if ev.Sequence == 0 {
		t.Error("Sequence must be >= 1 after first rotation")
	}
	if ev.BytesWritten == 0 {
		t.Error("BytesWritten must be > 0 for sealed segment")
	}
	if ev.Timestamp.IsZero() {
		t.Error("Timestamp must not be zero")
	}
}

// TestOnRotate_FiresOnFlushAndRotate verifies callback fires on manual rotation.
func TestOnRotate_FiresOnFlushAndRotate(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "audit.log")

	var received atomic.Int32

	config := &LoggerConfig{
		Filename: logFile,
		OnRotate: func(event RotationEvent) {
			received.Add(1)
		},
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Write some data
	if _, err := logger.Write([]byte("audit entry\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Manual rotation
	if err := logger.FlushAndRotate(); err != nil {
		t.Fatalf("FlushAndRotate: %v", err)
	}
	logger.WaitForBackgroundTasks()

	if received.Load() != 1 {
		t.Errorf("OnRotate called %d times, want 1", received.Load())
	}
}

// TestOnRotate_SequenceIsMonotonic verifies that Sequence increments
// on each rotation and never goes backward.
func TestOnRotate_SequenceIsMonotonic(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "seq.log")

	var mu sync.Mutex
	var sequences []uint64

	config := &LoggerConfig{
		Filename: logFile,
		OnRotate: func(event RotationEvent) {
			mu.Lock()
			defer mu.Unlock()
			sequences = append(sequences, event.Sequence)
		},
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Perform 3 manual rotations
	for i := 0; i < 3; i++ {
		if _, err := logger.Write([]byte(fmt.Sprintf("segment %d\n", i))); err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
		if err := logger.FlushAndRotate(); err != nil {
			t.Fatalf("FlushAndRotate %d: %v", i, err)
		}
		logger.WaitForBackgroundTasks()
		// Small delay to ensure unique backup names (resolution is seconds)
		time.Sleep(20 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(sequences) < 3 {
		t.Fatalf("expected 3 rotations, got %d", len(sequences))
	}

	for i := 1; i < len(sequences); i++ {
		if sequences[i] <= sequences[i-1] {
			t.Errorf("sequence not monotonic: seq[%d]=%d <= seq[%d]=%d",
				i, sequences[i], i-1, sequences[i-1])
		}
	}
}

// TestOnRotate_BytesWrittenMatchesData verifies that BytesWritten in the
// event reflects the actual data written to the sealed segment.
func TestOnRotate_BytesWrittenMatchesData(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "bytes.log")

	var mu sync.Mutex
	var capturedBytes uint64

	config := &LoggerConfig{
		Filename: logFile,
		OnRotate: func(event RotationEvent) {
			mu.Lock()
			defer mu.Unlock()
			capturedBytes = event.BytesWritten
		},
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Write known amount of data
	payload := []byte("exactly 20 bytes!!!\n")
	totalWritten := 0
	for i := 0; i < 5; i++ {
		n, err := logger.Write(payload)
		if err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
		totalWritten += n
	}

	if err := logger.FlushAndRotate(); err != nil {
		t.Fatalf("FlushAndRotate: %v", err)
	}
	logger.WaitForBackgroundTasks()

	mu.Lock()
	defer mu.Unlock()

	if capturedBytes != uint64(totalWritten) {
		t.Errorf("BytesWritten = %d, want %d", capturedBytes, totalWritten)
	}
}

// TestOnRotate_PreviousFileExists verifies that PreviousFile in the event
// points to an actual file on disk at callback time.
func TestOnRotate_PreviousFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "prev.log")

	var mu sync.Mutex
	var prevFile string

	config := &LoggerConfig{
		Filename: logFile,
		OnRotate: func(event RotationEvent) {
			mu.Lock()
			defer mu.Unlock()
			prevFile = event.PreviousFile
		},
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	defer func() { _ = logger.Close() }()

	if _, err := logger.Write([]byte("data\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if err := logger.FlushAndRotate(); err != nil {
		t.Fatalf("FlushAndRotate: %v", err)
	}
	logger.WaitForBackgroundTasks()

	mu.Lock()
	defer mu.Unlock()

	if prevFile == "" {
		t.Fatal("PreviousFile was not set")
	}

	// WHY check existence: the callback fires before compression/cleanup
	// so the sealed file must still exist at callback time
	if _, err := os.Stat(prevFile); err != nil {
		t.Errorf("PreviousFile %q does not exist: %v", prevFile, err)
	}
}

// TestOnRotate_NilCallbackNoError verifies that rotation works normally
// when OnRotate is not set (nil). Regression guard.
func TestOnRotate_NilCallbackNoError(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "nil.log")

	config := &LoggerConfig{
		Filename:   logFile,
		MaxSizeStr: "1KB",
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Write enough to trigger rotation without OnRotate set
	data := []byte(strings.Repeat("y", 200) + "\n")
	for i := 0; i < 20; i++ {
		if _, err := logger.Write(data); err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
	}

	logger.WaitForBackgroundTasks()

	// If we get here without panic/crash, the test passes
}

// TestOnRotate_PanicRecovery verifies that a panicking OnRotate callback
// does not crash the rotation path or block future rotations.
func TestOnRotate_PanicRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "panic.log")

	var errorReported atomic.Int32

	config := &LoggerConfig{
		Filename:   logFile,
		MaxSizeStr: "1KB",
		OnRotate: func(_ RotationEvent) {
			panic("callback explosion")
		},
		ErrorCallback: func(operation string, _ error) {
			if operation == "on_rotate_panic" {
				errorReported.Add(1)
			}
		},
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Write enough to trigger rotation -- the panic must be caught
	data := []byte(strings.Repeat("z", 200) + "\n")
	for i := 0; i < 20; i++ {
		if _, err := logger.Write(data); err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
	}

	logger.WaitForBackgroundTasks()

	if errorReported.Load() == 0 {
		t.Error("panic in OnRotate was not reported via ErrorCallback")
	}

	// Verify rotation still works after panic (write more data)
	for i := 0; i < 20; i++ {
		if _, err := logger.Write(data); err != nil {
			t.Fatalf("Write after panic %d: %v", i, err)
		}
	}

	logger.WaitForBackgroundTasks()
}

// TestOnRotate_DirectLoggerStruct verifies OnRotate works when set
// directly on the Logger struct (not via LoggerConfig).
func TestOnRotate_DirectLoggerStruct(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "direct.log")

	var called atomic.Int32

	logger := &Logger{
		Filename:   logFile,
		MaxSizeStr: "1KB",
		OnRotate: func(_ RotationEvent) {
			called.Add(1)
		},
	}
	defer func() { _ = logger.Close() }()

	data := []byte(strings.Repeat("d", 200) + "\n")
	for i := 0; i < 20; i++ {
		if _, err := logger.Write(data); err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
	}

	logger.WaitForBackgroundTasks()

	if called.Load() == 0 {
		t.Error("OnRotate not called when set directly on Logger struct")
	}
}
