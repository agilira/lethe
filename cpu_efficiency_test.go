// cpu_efficiency_test.go: Tests for CPU efficiency of MPSC consumer
//
// These tests verify that the consumer goroutine does not waste CPU cycles
// when the buffer is empty (idle state).
//
// Copyright (c) 2025 AGILira
// Series: an AGILira fragment
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestMPSCConsumer_IdleCPUEfficiency verifies that the MPSC consumer
// does not waste CPU cycles polling when the buffer is empty.
//
// We verify this by checking that flush latency after idle is fast (event-driven)
// rather than slow (polling with backoff).
func TestMPSCConsumer_IdleCPUEfficiency(t *testing.T) {
	testFile := filepath.Join(os.TempDir(), "lethe_cpu_test.log")
	defer func() { _ = os.Remove(testFile) }()
	defer func() {
		matches, _ := filepath.Glob(testFile + "*")
		for _, m := range matches {
			_ = os.Remove(m)
		}
	}()

	// Create async logger to activate MPSC consumer
	logger := &Logger{
		Filename:   testFile,
		MaxSizeStr: "100MB",
		Async:      true,
		BufferSize: 1024,
	}

	// Write one message to trigger MPSC initialization
	_, err := logger.Write([]byte("init\n"))
	if err != nil {
		t.Fatalf("Initial write failed: %v", err)
	}

	// Wait for consumer to process and become idle
	time.Sleep(50 * time.Millisecond)

	// Let consumer go completely idle for a longer period
	// If it's polling, it will consume CPU
	// If it's event-driven, it will be blocked on cond.Wait()
	time.Sleep(200 * time.Millisecond)

	// Now measure response time - if event-driven, it should wake up instantly
	// If polling with backoff, it could take up to the backoff interval
	var totalLatency time.Duration
	const iterations = 5

	for i := 0; i < iterations; i++ {
		// Let it go idle again
		time.Sleep(50 * time.Millisecond)

		start := time.Now()
		testMsg := []byte("test message\n")
		_, err := logger.Write(testMsg)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		// Wait for flush by checking file
		deadline := time.Now().Add(100 * time.Millisecond)
		for time.Now().Before(deadline) {
			info, _ := os.Stat(testFile)
			if info != nil && info.Size() > int64(5+len("init\n")+i*len(testMsg)) {
				break
			}
			time.Sleep(100 * time.Microsecond)
		}
		latency := time.Since(start)
		totalLatency += latency
	}

	avgLatency := totalLatency / iterations
	_ = logger.Close()

	t.Logf("Average flush latency after idle: %v", avgLatency)

	// Event-driven wakeup should respond in < 1ms
	// Polling with 5ms backoff would average ~2.5ms
	// We allow up to 2ms for system scheduling variance
	if avgLatency > 2*time.Millisecond {
		t.Errorf("Flush latency too high (%.2fms), expected < 2ms for event-driven consumer",
			float64(avgLatency.Microseconds())/1000)
	}
}

// TestMPSCConsumer_WakeupCount tracks how many times the consumer wakes up
// when idle vs when there's work to do.
func TestMPSCConsumer_WakeupCount(t *testing.T) {
	testFile := filepath.Join(os.TempDir(), "lethe_wakeup_test.log")
	defer func() { _ = os.Remove(testFile) }()
	defer func() {
		matches, _ := filepath.Glob(testFile + "*")
		for _, m := range matches {
			_ = os.Remove(m)
		}
	}()

	// Create logger with async mode
	logger := &Logger{
		Filename:      testFile,
		MaxSizeStr:    "100MB",
		Async:         true,
		BufferSize:    1024,
		FlushInterval: 1 * time.Millisecond, // Current default - aggressive
	}

	// Initialize MPSC
	_, err := logger.Write([]byte("init\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Let it stabilize
	time.Sleep(20 * time.Millisecond)

	// Measure goroutines before idle period
	numGoroutinesBefore := runtime.NumGoroutine()

	// Idle period - no writes for 200ms
	idleStart := time.Now()
	time.Sleep(200 * time.Millisecond)
	idleDuration := time.Since(idleStart)

	numGoroutinesAfter := runtime.NumGoroutine()
	_ = logger.Close()

	t.Logf("Idle duration: %v", idleDuration)
	t.Logf("Goroutines before: %d, after: %d", numGoroutinesBefore, numGoroutinesAfter)

	// With 1ms polling and 200ms idle, consumer wakes up ~200 times
	// With adaptive backoff (5ms), it wakes up ~40 times after backoff kicks in
	// With event-driven (ideal), it wakes up 0 times during idle
	//
	// This test documents the current behavior. After fix:
	// - Consumer should block on a condition variable when buffer is empty
	// - Wake up only when new data arrives or on shutdown

	// For now we just verify the goroutine count is stable (no leaks)
	if numGoroutinesAfter > numGoroutinesBefore+2 {
		t.Errorf("Goroutine leak detected: before=%d, after=%d", numGoroutinesBefore, numGoroutinesAfter)
	}
}

// TestMPSCConsumer_EventDrivenWakeup tests that the consumer wakes up
// promptly when new data arrives, even after being idle.
func TestMPSCConsumer_EventDrivenWakeup(t *testing.T) {
	testFile := filepath.Join(os.TempDir(), "lethe_event_test.log")
	defer func() { _ = os.Remove(testFile) }()
	defer func() {
		matches, _ := filepath.Glob(testFile + "*")
		for _, m := range matches {
			_ = os.Remove(m)
		}
	}()

	logger := &Logger{
		Filename:      testFile,
		MaxSizeStr:    "100MB",
		Async:         true,
		BufferSize:    1024,
		AdaptiveFlush: true, // Enable adaptive to test backoff recovery
	}

	// Initialize
	_, _ = logger.Write([]byte("init\n"))
	time.Sleep(50 * time.Millisecond)

	// Let consumer go idle and backoff
	time.Sleep(100 * time.Millisecond)

	// Now write new data and measure latency to disk
	writeStart := time.Now()
	testMsg := []byte("test message after idle\n")
	_, err := logger.Write(testMsg)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Wait for flush with timeout
	deadline := time.Now().Add(50 * time.Millisecond)
	var found bool
	for time.Now().Before(deadline) {
		content, _ := os.ReadFile(testFile)
		if len(content) >= len(testMsg) {
			// Check if content contains our test message
			if string(content[len(content)-len(testMsg):]) == string(testMsg) {
				found = true
				break
			}
		}
		time.Sleep(1 * time.Millisecond)
	}
	flushLatency := time.Since(writeStart)

	_ = logger.Close()

	t.Logf("Flush latency after idle: %v", flushLatency)

	if !found {
		t.Error("Message not flushed within 50ms deadline")
	}

	// After implementing event-driven wakeup, latency should be < 5ms
	// Currently with backoff, it could be up to 5ms (backoff interval)
	if flushLatency > 10*time.Millisecond {
		t.Errorf("Flush latency too high after idle: %v (expected < 10ms)", flushLatency)
	}
}
