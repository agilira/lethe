// mpsc_race_test.go: Tests for MPSC ring buffer race condition fix
//
// This test verifies that the consumer does NOT skip slots when the producer
// is slow between CAS(tail) and Store. This is a critical correctness test.
//
// Copyright (c) 2025 AGILira
// Series: Lethe - MPSC Race Fix
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestMPSCRingBuffer_SlowProducerNoSkip simulates a slow producer scenario
// where the producer is artificially delayed between CAS and Store.
// The consumer must NOT skip the slot - it should return false and retry.
func TestMPSCRingBuffer_SlowProducerNoSkip(t *testing.T) {
	rb := newRingBuffer(64)

	// Simulate slow producer: reserve slot, delay, then write
	tail := rb.tail.Load()
	rb.tail.Store(tail + 1) // Simulate CAS success (slot reserved)

	// Consumer tries to pop immediately - should return false (data not ready)
	_, ok := rb.pop()
	if ok {
		t.Error("pop() should return false when data not yet written")
	}

	// Verify head was NOT incremented (this is the critical fix)
	if rb.head.Load() != 0 {
		t.Error("CRITICAL: head was incremented before data was written - message will be lost!")
	}

	// Now producer writes data
	msg := []byte("delayed message")
	rb.buffer[tail&rb.mask].Store(&msg)

	// Consumer should now succeed
	data, ok := rb.pop()
	if !ok || data == nil {
		t.Error("pop() should succeed after data is written")
	}
}

// TestMPSCRingBuffer_NoMessageLoss verifies zero message loss with proper
// push/pop API usage (not direct manipulation).
func TestMPSCRingBuffer_NoMessageLoss(t *testing.T) {
	const (
		numProducers  = 4
		msgsPerProd   = 500
		totalMessages = numProducers * msgsPerProd
	)

	rb := newRingBuffer(4096) // Large enough buffer
	var sent, received atomic.Int64
	var producersWg sync.WaitGroup

	// Start consumer first
	stopConsumer := make(chan struct{})
	consumerDone := make(chan struct{})

	go func() {
		defer close(consumerDone)
		for {
			select {
			case <-stopConsumer:
				// Drain any remaining messages
				for i := 0; i < 100; i++ {
					for {
						if data, ok := rb.pop(); ok && data != nil {
							received.Add(1)
						} else {
							break
						}
					}
					time.Sleep(time.Millisecond)
				}
				return
			default:
				if data, ok := rb.pop(); ok && data != nil {
					received.Add(1)
				} else {
					time.Sleep(10 * time.Microsecond)
				}
			}
		}
	}()

	// Producers
	producersWg.Add(numProducers)
	for p := 0; p < numProducers; p++ {
		go func() {
			defer producersWg.Done()
			for i := 0; i < msgsPerProd; i++ {
				data := []byte("test message")
				for !rb.push(data) {
					runtime.Gosched()
				}
				sent.Add(1)
			}
		}()
	}

	producersWg.Wait()
	close(stopConsumer)
	<-consumerDone

	sentTotal := sent.Load()
	gotTotal := received.Load()

	if gotTotal != sentTotal {
		t.Errorf("Message loss! Sent %d, received %d (lost %d)",
			sentTotal, gotTotal, sentTotal-gotTotal)
	}
}

// TestMPSCRingBuffer_ConcurrentStress is a higher-stress version.
func TestMPSCRingBuffer_ConcurrentStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	const (
		numProducers = 4
		msgsPerProd  = 1000
	)

	rb := newRingBuffer(4096)
	var sent, received atomic.Int64
	var producersWg sync.WaitGroup

	// Consumer runs continuously until all messages received
	consumerDone := make(chan struct{})

	go func() {
		defer close(consumerDone)
		expected := int64(numProducers * msgsPerProd)
		timeout := time.After(10 * time.Second)

		for received.Load() < expected {
			select {
			case <-timeout:
				return // Timeout - test will fail with message count
			default:
				if data, ok := rb.pop(); ok && data != nil {
					received.Add(1)
				} else {
					time.Sleep(10 * time.Microsecond)
				}
			}
		}
	}()

	producersWg.Add(numProducers)
	for p := 0; p < numProducers; p++ {
		go func() {
			defer producersWg.Done()
			for i := 0; i < msgsPerProd; i++ {
				data := []byte("stress test message")
				for !rb.push(data) {
					runtime.Gosched()
				}
				sent.Add(1)
			}
		}()
	}

	producersWg.Wait()
	<-consumerDone

	sentTotal := sent.Load()
	gotTotal := received.Load()

	if gotTotal != sentTotal {
		t.Errorf("CRITICAL: Message loss! Sent %d, received %d (lost %d, %.2f%%)",
			sentTotal, gotTotal, sentTotal-gotTotal,
			float64(sentTotal-gotTotal)/float64(sentTotal)*100)
	}
}
