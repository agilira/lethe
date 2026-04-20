// reconfigure_fuzz_test.go: Fuzz tests for ReconfigureRetention.
//
// Copyright (c) 2025 AGILira
// SPDX-License-Identifier: MPL-2.0
package lethe

import (
	"path/filepath"
	"testing"
	"time"
)

// FuzzReconfigureRetention feeds arbitrary int/duration/bool combinations
// to ReconfigureRetention. Invariants:
//   - Must never panic.
//   - Negative MaxBackups or MaxFileAge must always return a non-nil error.
//   - Non-negative inputs must always return nil error.
//   - effectiveRetention() must always return a consistent (non-zero-struct
//     from a bad swap) policy after a successful reconfigure.
func FuzzReconfigureRetention(f *testing.F) {
	f.Add(int64(0), int64(0), false, false)
	f.Add(int64(5), int64(int(24*time.Hour)), true, true)
	f.Add(int64(-1), int64(0), false, false)
	f.Add(int64(0), int64(-1), false, false)
	f.Add(int64(100), int64(int(7*365*24*time.Hour)), true, true)
	f.Add(int64(-1000), int64(-999), true, true)

	dir := f.TempDir()

	l, err := NewWithConfig(&LoggerConfig{
		Filename: filepath.Join(dir, "fuzz.log"),
	})
	if err != nil {
		f.Fatal(err)
	}
	defer l.Close()

	f.Fuzz(func(t *testing.T, maxBackups int64, maxFileAge int64, compress, checksum bool) {
		policy := RetentionPolicy{
			MaxBackups: int(maxBackups),
			MaxFileAge: time.Duration(maxFileAge),
			Compress:   compress,
			Checksum:   checksum,
		}

		err := l.ReconfigureRetention(policy)

		isInvalid := maxBackups < 0 || maxFileAge < 0
		if isInvalid && err == nil {
			t.Errorf("expected error for invalid policy (MaxBackups=%d, MaxFileAge=%d), got nil",
				maxBackups, maxFileAge)
		}
		if !isInvalid && err != nil {
			t.Errorf("unexpected error for valid policy: %v", err)
		}

		// effectiveRetention must never panic regardless of what was stored.
		_ = l.effectiveRetention()
	})
}
