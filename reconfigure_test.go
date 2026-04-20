// reconfigure_test.go: Unit tests for ReconfigureRetention.
//
// Copyright (c) 2025 AGILira
// SPDX-License-Identifier: MPL-2.0
package lethe

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// effectiveRetention — fallback to construction-time fields
// ---------------------------------------------------------------------------

func TestEffectiveRetention_FallsBackToConstructionFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{
		Filename:   filepath.Join(dir, "test.log"),
		MaxBackups: 5,
		MaxFileAge: 48 * time.Hour,
		Compress:   true,
		Checksum:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close() error: %v", err)
		}
	}()

	ret := l.effectiveRetention()
	if ret.MaxBackups != 5 {
		t.Errorf("MaxBackups: got %d, want 5", ret.MaxBackups)
	}
	if ret.MaxFileAge != 48*time.Hour {
		t.Errorf("MaxFileAge: got %v, want 48h", ret.MaxFileAge)
	}
	if !ret.Compress {
		t.Error("Compress: got false, want true")
	}
	if !ret.Checksum {
		t.Error("Checksum: got false, want true")
	}
}

// ---------------------------------------------------------------------------
// ReconfigureRetention — happy paths
// ---------------------------------------------------------------------------

func TestReconfigureRetention_UpdatesPolicy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{
		Filename:   filepath.Join(dir, "test.log"),
		MaxBackups: 3,
		Compress:   false,
		Checksum:   false,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close() error: %v", err)
		}
	}()

	if err := l.ReconfigureRetention(RetentionPolicy{
		MaxBackups: 10,
		MaxFileAge: 7 * 24 * time.Hour,
		Compress:   true,
		Checksum:   true,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ret := l.effectiveRetention()
	if ret.MaxBackups != 10 {
		t.Errorf("MaxBackups: got %d, want 10", ret.MaxBackups)
	}
	if ret.MaxFileAge != 7*24*time.Hour {
		t.Errorf("MaxFileAge: got %v, want 168h", ret.MaxFileAge)
	}
	if !ret.Compress {
		t.Error("Compress: got false, want true")
	}
	if !ret.Checksum {
		t.Error("Checksum: got false, want true")
	}
}

func TestReconfigureRetention_OverridesConstructionFields(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{
		Filename:   filepath.Join(dir, "test.log"),
		MaxBackups: 5,
		Compress:   true,
		Checksum:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close() error: %v", err)
		}
	}()

	if err := l.ReconfigureRetention(RetentionPolicy{
		MaxBackups: 0,
		Compress:   false,
		Checksum:   false,
	}); err != nil {
		t.Fatal(err)
	}

	ret := l.effectiveRetention()
	if ret.MaxBackups != 0 {
		t.Errorf("MaxBackups: got %d, want 0", ret.MaxBackups)
	}
	if ret.Compress {
		t.Error("Compress: got true, want false")
	}
	if ret.Checksum {
		t.Error("Checksum: got true, want false")
	}
}

func TestReconfigureRetention_ZeroValueIsValid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{
		Filename: filepath.Join(dir, "test.log"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close() error: %v", err)
		}
	}()

	if err := l.ReconfigureRetention(RetentionPolicy{}); err != nil {
		t.Errorf("zero RetentionPolicy should be valid, got: %v", err)
	}
}

func TestReconfigureRetention_MultipleCallsLastWins(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{
		Filename: filepath.Join(dir, "test.log"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close() error: %v", err)
		}
	}()

	for i := range 5 {
		_ = l.ReconfigureRetention(RetentionPolicy{MaxBackups: i + 1})
	}

	ret := l.effectiveRetention()
	if ret.MaxBackups != 5 {
		t.Errorf("last call should win: got MaxBackups=%d, want 5", ret.MaxBackups)
	}
}

func TestReconfigureRetention_LongRetentionForCompliance(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{
		Filename: filepath.Join(dir, "audit.log"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close() error: %v", err)
		}
	}()

	sevenYears := 7 * 365 * 24 * time.Hour
	if err := l.ReconfigureRetention(RetentionPolicy{
		MaxFileAge: sevenYears,
		MaxBackups: 0,
		Checksum:   true,
		Compress:   true,
	}); err != nil {
		t.Fatalf("7-year retention should be valid: %v", err)
	}

	ret := l.effectiveRetention()
	if ret.MaxFileAge != sevenYears {
		t.Errorf("MaxFileAge: got %v, want %v", ret.MaxFileAge, sevenYears)
	}
	if !ret.Checksum {
		t.Error("Checksum must be true for compliance retention")
	}
}

// ---------------------------------------------------------------------------
// ReconfigureRetention — validation errors
// ---------------------------------------------------------------------------

func TestReconfigureRetention_NegativeMaxBackupsIsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{
		Filename: filepath.Join(dir, "test.log"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close() error: %v", err)
		}
	}()

	if err := l.ReconfigureRetention(RetentionPolicy{MaxBackups: -1}); err == nil {
		t.Error("expected error for negative MaxBackups, got nil")
	}
}

func TestReconfigureRetention_NegativeMaxFileAgeIsError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{
		Filename: filepath.Join(dir, "test.log"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close() error: %v", err)
		}
	}()

	if err := l.ReconfigureRetention(RetentionPolicy{MaxFileAge: -time.Hour}); err == nil {
		t.Error("expected error for negative MaxFileAge, got nil")
	}
}

func TestReconfigureRetention_PolicyUnchangedOnValidationError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{
		Filename:   filepath.Join(dir, "test.log"),
		MaxBackups: 7,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close() error: %v", err)
		}
	}()

	_ = l.ReconfigureRetention(RetentionPolicy{MaxBackups: -1})

	ret := l.effectiveRetention()
	if ret.MaxBackups != 7 {
		t.Errorf("policy must be unchanged after validation error: got MaxBackups=%d, want 7", ret.MaxBackups)
	}
}

// ---------------------------------------------------------------------------
// Concurrency — safe under -race
// ---------------------------------------------------------------------------

func TestReconfigureRetention_ConcurrentWithWrites(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{
		Filename:   filepath.Join(dir, "test.log"),
		MaxSizeStr: "1MB",
		Async:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close() error: %v", err)
		}
	}()

	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := range goroutines {
		go func(n int) {
			defer wg.Done()
			_ = l.ReconfigureRetention(RetentionPolicy{
				MaxBackups: n + 1,
				Checksum:   n%2 == 0,
				Compress:   n%2 != 0,
			})
		}(i)

		go func() {
			defer wg.Done()
			_, _ = l.Write([]byte("concurrent write\n"))
		}()
	}

	wg.Wait()
	// No panic or data race = pass.
}

func TestReconfigureRetention_ConcurrentReaders(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{
		Filename: filepath.Join(dir, "test.log"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close() error: %v", err)
		}
	}()

	const goroutines = 16
	var wg sync.WaitGroup
	wg.Add(goroutines + 1)

	go func() {
		defer wg.Done()
		for i := range goroutines {
			_ = l.ReconfigureRetention(RetentionPolicy{MaxBackups: i})
		}
	}()

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = l.effectiveRetention()
		}()
	}

	wg.Wait()
}

// ---------------------------------------------------------------------------
// Integration — effectiveRetention drives scheduleBackgroundTasks
// ---------------------------------------------------------------------------

func TestReconfigureRetention_ChecksumReflectedInEffectivePolicy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{
		Filename: filepath.Join(dir, "test.log"),
		Checksum: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close() error: %v", err)
		}
	}()

	// Before reconfigure: Checksum is false.
	if l.effectiveRetention().Checksum {
		t.Error("Checksum should be false before reconfigure")
	}

	if err := l.ReconfigureRetention(RetentionPolicy{Checksum: true, MaxBackups: 5}); err != nil {
		t.Fatal(err)
	}

	// After reconfigure: effectiveRetention must reflect the new policy.
	// scheduleBackgroundTasks reads effectiveRetention(), so this is the
	// authoritative gate for whether checksums are generated on next rotation.
	ret := l.effectiveRetention()
	if !ret.Checksum {
		t.Error("Checksum should be true after ReconfigureRetention")
	}
	if ret.MaxBackups != 5 {
		t.Errorf("MaxBackups: got %d, want 5", ret.MaxBackups)
	}
}
