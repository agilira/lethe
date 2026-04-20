// reconfigure.go: Hot-reload of retention policy without Logger restart.
//
// Copyright (c) 2025 AGILira
// Series: an AGILira fragment
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"errors"
	"time"
)

// RetentionPolicy holds the mutable retention configuration that can be
// changed at runtime via ReconfigureRetention without restarting the Logger.
//
// All fields are applied on the next rotation cycle; in-flight writes are
// not affected. The zero value is valid: zero MaxBackups disables count-based
// cleanup, zero MaxFileAge disables age-based cleanup.
type RetentionPolicy struct {
	// MaxFileAge is the maximum age for backup files before deletion.
	// Zero disables age-based cleanup.
	MaxFileAge time.Duration

	// MaxBackups is the maximum number of old log files to retain.
	// Zero retains all backups.
	MaxBackups int

	// Compress enables gzip compression of rotated files.
	Compress bool

	// Checksum enables SHA-256 checksum calculation for file integrity.
	// Required for AI Act audit trail compliance.
	Checksum bool
}

// ReconfigureRetention atomically replaces the active retention policy.
// Safe to call from any goroutine while the Logger is running.
// Changes take effect on the next rotation cycle.
//
// Returns an error if the policy is invalid (negative MaxBackups or MaxFileAge).
func (l *Logger) ReconfigureRetention(policy RetentionPolicy) error {
	if policy.MaxBackups < 0 {
		return errors.New("lethe: ReconfigureRetention: MaxBackups must be >= 0")
	}
	if policy.MaxFileAge < 0 {
		return errors.New("lethe: ReconfigureRetention: MaxFileAge must be >= 0")
	}
	p := policy
	l.retention.Store(&p)
	return nil
}

// effectiveRetention returns the active retention policy.
// Prefers the atomically stored policy if set; falls back to the
// construction-time fields for zero-allocation backward compatibility.
func (l *Logger) effectiveRetention() RetentionPolicy {
	if p := l.retention.Load(); p != nil {
		return *p
	}
	return RetentionPolicy{
		MaxFileAge: l.MaxFileAge,
		MaxBackups: l.MaxBackups,
		Compress:   l.Compress,
		Checksum:   l.Checksum,
	}
}
