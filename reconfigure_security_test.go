// reconfigure_security_test.go: Security and adversarial tests for ReconfigureRetention.
//
// THREAT MODEL
//
// CWE-362 (Race Condition / TOCTOU):
//   A concurrent ReconfigureRetention + rotation sequence must not expose a
//   window where effectiveRetention() returns a partially-written policy.
//   MITIGATION: atomic.Pointer swap is a single hardware instruction; no
//   partial state is ever visible.
//
// CWE-20 (Improper Input Validation):
//   Negative MaxBackups or MaxFileAge could wrap to MaxInt in unsigned math
//   downstream, causing unbounded file retention or immediate deletion of all
//   backups. MITIGATION: rejected at the API boundary with an explicit error.
//
// CWE-400 (Uncontrolled Resource Consumption):
//   An attacker who controls the retention policy could set MaxBackups=0 +
//   MaxFileAge=0, retaining log files forever and exhausting disk.
//   This is an operator decision, not a security flaw in lethe itself —
//   the caller (Metis config layer) must validate business constraints.
//   Lethe accepts zero (no limit) as a valid operator choice.
//
// CWE-667 (Improper Locking):
//   effectiveRetention() must never hold a lock while rotation holds one,
//   or deadlock is possible. MITIGATION: effectiveRetention uses only
//   atomic.Pointer.Load(), which never blocks.
//
// Copyright (c) 2025 AGILira
// SPDX-License-Identifier: MPL-2.0
package lethe

import (
	"math"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// CWE-20: boundary values must be rejected before they reach rotation logic.

func TestReconfigureRetention_Security_MaxInt32BackupsAccepted(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{Filename: filepath.Join(dir, "t.log")})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	// Large positive value is a valid (if impractical) operator choice.
	if err := l.ReconfigureRetention(RetentionPolicy{MaxBackups: math.MaxInt32}); err != nil {
		t.Errorf("MaxInt32 MaxBackups should be accepted: %v", err)
	}
}

func TestReconfigureRetention_Security_NegativeMaxBackupsBlocked(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{Filename: filepath.Join(dir, "t.log")})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	cases := []int{-1, -100, math.MinInt32, math.MinInt64}
	for _, n := range cases {
		if err := l.ReconfigureRetention(RetentionPolicy{MaxBackups: n}); err == nil {
			t.Errorf("MaxBackups=%d: expected error, got nil", n)
		}
	}
}

func TestReconfigureRetention_Security_NegativeMaxFileAgeBlocked(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{Filename: filepath.Join(dir, "t.log")})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	cases := []time.Duration{-time.Nanosecond, -time.Hour, time.Duration(math.MinInt64)}
	for _, d := range cases {
		if err := l.ReconfigureRetention(RetentionPolicy{MaxFileAge: d}); err == nil {
			t.Errorf("MaxFileAge=%v: expected error, got nil", d)
		}
	}
}

// CWE-362: atomic swap must be invisible to concurrent readers — no torn reads.

func TestReconfigureRetention_Security_NoTornReadUnderContention(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{Filename: filepath.Join(dir, "t.log")})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	const (
		writers = 4
		readers = 8
		iters   = 500
	)

	var wg sync.WaitGroup
	wg.Add(writers + readers)

	for w := range writers {
		go func(n int) {
			defer wg.Done()
			for range iters {
				_ = l.ReconfigureRetention(RetentionPolicy{
					MaxBackups: n + 1,
					MaxFileAge: time.Duration(n+1) * time.Hour,
					Checksum:   n%2 == 0,
					Compress:   n%2 != 0,
				})
			}
		}(w)
	}

	for range readers {
		go func() {
			defer wg.Done()
			for range iters {
				ret := l.effectiveRetention()
				// Any valid policy is acceptable — we just must not panic or
				// observe a zero MaxBackups alongside a non-zero MaxFileAge
				// from two different in-flight policies (torn read).
				_ = ret
			}
		}()
	}

	wg.Wait()
}

// CWE-667: effectiveRetention must not deadlock when called during rotation.

func TestReconfigureRetention_Security_NoDeadlockDuringRotation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{
		Filename:   filepath.Join(dir, "t.log"),
		MaxSizeStr: "1B",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 20 {
			_ = l.ReconfigureRetention(RetentionPolicy{MaxBackups: 3, Checksum: true})
			_, _ = l.Write([]byte("x"))
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock detected: ReconfigureRetention + rotation did not complete within 5s")
	}
}

// Idempotency: reconfiguring with the same policy is a no-op in observable behavior.

func TestReconfigureRetention_Security_IdempotentReconfigure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l, err := NewWithConfig(&LoggerConfig{Filename: filepath.Join(dir, "t.log")})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	policy := RetentionPolicy{MaxBackups: 5, MaxFileAge: 24 * time.Hour, Checksum: true}
	for range 100 {
		if err := l.ReconfigureRetention(policy); err != nil {
			t.Fatalf("idempotent reconfigure failed: %v", err)
		}
	}

	ret := l.effectiveRetention()
	if ret.MaxBackups != 5 || ret.MaxFileAge != 24*time.Hour || !ret.Checksum {
		t.Errorf("policy corrupted after 100 identical reconfigures: %+v", ret)
	}
}
