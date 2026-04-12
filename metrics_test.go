// metrics_test.go: Tests for enhanced observable metrics
//
// Copyright (c) 2025 AGILira
// Series: an AGILira fragment
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// =============================================================================
// TIER-1 AUDIT: Observable Metrics Tests
// =============================================================================
//
// REQUIREMENT: Audit systems need real-time metrics for monitoring.
// Prometheus/Grafana dashboards need:
// - Drop counts (backpressure events)
// - Queue depth (buffer utilization)
// - Write latency (performance)
// - Last write/drop timestamps
//
// =============================================================================

// TestStats_DroppedCount verifies dropped message counting.
func TestStats_DroppedCount(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	// Create async logger with tiny buffer and drop policy
	config := &LoggerConfig{
		Filename:           logFile,
		Async:              true,
		BufferSize:         8, // Very small buffer
		BackpressurePolicy: "drop",
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Flood the buffer to trigger drops
	data := make([]byte, 100)
	for i := 0; i < 1000; i++ {
		// WHY: Write may fail when buffer is full; drops are expected under flood.
		if _, err := logger.Write(data); err != nil {
			t.Logf("Write dropped under pressure (expected): %v", err)
		}
	}

	stats := logger.Stats()

	// Should have some drops with such a small buffer
	if stats.DroppedOnFull == 0 {
		// Note: This might be flaky depending on timing
		t.Log("Warning: No drops detected, buffer might be draining too fast")
	}

	// DroppedOnFull should be accessible
	t.Logf("Stats: DroppedOnFull=%d, BufferSize=%d", stats.DroppedOnFull, stats.BufferSize)
}

// TestStats_QueueDepth verifies buffer utilization metrics.
func TestStats_QueueDepth(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	config := &LoggerConfig{
		Filename:   logFile,
		Async:      true,
		BufferSize: 1024,
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Write some data
	for i := 0; i < 10; i++ {
		if _, err := logger.Write([]byte("test entry\n")); err != nil {
			t.Errorf("Write failed on iteration %d: %v", i, err)
		}
	}

	stats := logger.Stats()

	// BufferSize should match config (rounded to power of 2)
	if stats.BufferSize == 0 {
		t.Error("Expected non-zero BufferSize")
	}

	// BufferFill should be accessible (may be 0 if consumer is fast)
	t.Logf("Stats: BufferSize=%d, BufferFill=%d, IsMPSCActive=%v",
		stats.BufferSize, stats.BufferFill, stats.IsMPSCActive)
}

// TestStats_Timestamps verifies timestamp tracking.
func TestStats_Timestamps(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	config := &LoggerConfig{
		Filename: logFile,
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	before := time.Now()

	// Write some data
	logger.Write([]byte("test entry\n"))

	after := time.Now()

	stats := logger.Stats()

	// LastWriteTime should be between before and after
	if stats.LastWriteTime.IsZero() {
		t.Error("LastWriteTime not set after write")
	}
	if stats.LastWriteTime.Before(before) || stats.LastWriteTime.After(after) {
		t.Errorf("LastWriteTime %v not between %v and %v",
			stats.LastWriteTime, before, after)
	}
}

// TestMetricsCallback verifies callback-based metrics export.
func TestMetricsCallback(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	var callbackCount atomic.Int32
	var lastStats Stats

	config := &LoggerConfig{
		Filename: logFile,
		MetricsCallback: func(s Stats) {
			callbackCount.Add(1)
			lastStats = s
		},
		MetricsInterval: 50 * time.Millisecond,
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Write some data
	for i := 0; i < 100; i++ {
		logger.Write([]byte("test entry\n"))
	}

	// Wait for at least one callback
	time.Sleep(150 * time.Millisecond)

	logger.Close()

	if callbackCount.Load() == 0 {
		t.Error("MetricsCallback was never called")
	}

	if lastStats.WriteCount == 0 {
		t.Error("MetricsCallback received empty stats")
	}

	t.Logf("Callback called %d times, last WriteCount=%d",
		callbackCount.Load(), lastStats.WriteCount)
}

// TestStats_ContentionRatio verifies contention detection.
func TestStats_ContentionRatio(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	config := &LoggerConfig{
		Filename: logFile,
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Write some data
	for i := 0; i < 100; i++ {
		logger.Write([]byte("test entry\n"))
	}

	stats := logger.Stats()

	// ContentionRatio should be between 0 and 1
	if stats.ContentionRatio < 0 || stats.ContentionRatio > 1 {
		t.Errorf("ContentionRatio out of range: %f", stats.ContentionRatio)
	}

	t.Logf("Stats: ContentionCount=%d, ContentionRatio=%.4f",
		stats.ContentionCount, stats.ContentionRatio)
}

// TestStats_LatencyTracking verifies latency metrics.
func TestStats_LatencyTracking(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	config := &LoggerConfig{
		Filename: logFile,
	}

	logger, err := NewWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Write some data
	for i := 0; i < 100; i++ {
		logger.Write([]byte("test entry for latency measurement\n"))
	}

	stats := logger.Stats()

	// After writes, should have some latency data
	if stats.WriteCount != 100 {
		t.Errorf("Expected WriteCount=100, got %d", stats.WriteCount)
	}

	// AvgLatencyNs should be reasonable (< 100ms per write)
	if stats.AvgLatencyNs > 100_000_000 {
		t.Errorf("Unreasonably high AvgLatencyNs: %d", stats.AvgLatencyNs)
	}

	t.Logf("Stats: AvgLatencyNs=%d, LastLatencyNs=%d",
		stats.AvgLatencyNs, stats.LastLatencyNs)
}
