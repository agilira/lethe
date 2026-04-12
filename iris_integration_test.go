// iris_integration_test.go: Tests for IrisIntegration (lethe -> iris bridge)
//
// WHY: IrisIntegration had zero test coverage. The Sync() method silently
// returned nil without delegating to Logger.Sync(), a data-loss bug that
// could skip fsync on shutdown. These tests prevent regression.
//
// Copyright (c) 2025 AGILira
// Series: Lethe
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestIrisIntegration_Write(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "write.log")

	w := NewIrisWriter(path, nil)
	defer func() {
		if err := w.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	n, err := w.Write([]byte("hello\n"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != 6 {
		t.Errorf("Write() = %d, want 6", n)
	}
}

func TestIrisIntegration_WriteOwned(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "owned.log")

	w := NewIrisWriter(path, nil)
	defer func() {
		if err := w.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	buf := make([]byte, 6)
	copy(buf, "owned\n")

	n, err := w.WriteOwned(buf)
	if err != nil {
		t.Fatalf("WriteOwned() error = %v", err)
	}
	if n != 6 {
		t.Errorf("WriteOwned() = %d, want 6", n)
	}
}

func TestIrisIntegration_Sync_DelegatesToLogger(t *testing.T) {
	// WHY: This test exposes the bug where Sync() returned nil
	// without calling Logger.Sync(). Data written before Sync()
	// must be on disk when Sync() returns.
	dir := t.TempDir()
	path := filepath.Join(dir, "sync.log")

	w := NewIrisWriter(path, nil)
	defer func() {
		if err := w.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	msg := "must be on disk after sync\n"
	if _, err := w.Write([]byte(msg)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Sync(); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	// Verify data is actually on disk
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "must be on disk after sync") {
		t.Errorf("data not on disk after Sync(): %q", string(data))
	}
}

func TestIrisIntegration_Close(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "close.log")

	w := NewIrisWriter(path, nil)
	if _, err := w.Write([]byte("before close\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Verify data survived close
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(data), "before close") {
		t.Errorf("data lost after Close(): %q", string(data))
	}
}

func TestIrisIntegration_GetOptimalBufferSize(t *testing.T) {
	w := NewIrisWriter(filepath.Join(t.TempDir(), "buf.log"), nil)
	defer func() {
		if err := w.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	size := w.GetOptimalBufferSize()
	if size <= 0 {
		t.Errorf("GetOptimalBufferSize() = %d, want > 0", size)
	}
}

func TestIrisIntegration_SupportsHotReload(t *testing.T) {
	w := NewIrisWriter(filepath.Join(t.TempDir(), "hr.log"), nil)
	defer func() {
		if err := w.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	if !w.SupportsHotReload() {
		t.Error("SupportsHotReload() = false, want true")
	}
}

func TestIrisIntegration_GetLogger(t *testing.T) {
	w := NewIrisWriter(filepath.Join(t.TempDir(), "gl.log"), nil)
	defer func() {
		if err := w.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	if w.GetLogger() == nil {
		t.Error("GetLogger() = nil, want non-nil")
	}
}

func TestQuickStart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "quick.log")

	w := QuickStart(path)
	defer func() {
		if err := w.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	if _, err := w.Write([]byte("quick\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if err := w.Sync(); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
}

func TestIrisIntegration_ConcurrentWriteOwned(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent.log")

	w := NewIrisWriter(path, nil)
	defer func() {
		if err := w.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	var wg sync.WaitGroup
	for g := 0; g < 20; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				buf := make([]byte, 32)
				copy(buf, "concurrent write\n")
				if _, err := w.WriteOwned(buf); err != nil {
					t.Errorf("WriteOwned goroutine %d iter %d: %v", id, i, err)
					return
				}
			}
		}(g)
	}
	wg.Wait()
}

func TestNewIrisWriter_NilConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nilcfg.log")

	w := NewIrisWriter(path, nil)
	defer func() {
		if err := w.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}()

	// Must have sensible defaults
	if w.logger.MaxSizeStr != "100MB" {
		t.Errorf("default MaxSizeStr = %q, want 100MB", w.logger.MaxSizeStr)
	}
	if w.logger.MaxBackups != 5 {
		t.Errorf("default MaxBackups = %d, want 5", w.logger.MaxBackups)
	}
	if !w.logger.Async {
		t.Error("default Async = false, want true")
	}
}
