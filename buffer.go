// buffer.go: MPSC ring buffer implementation for high-throughput logging
//
// Copyright (c) 2025 AGILira
// Series: an AGILira fragment
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"context"
	"math/bits"
	"sync"
	"sync/atomic"
	"time"
)

// SafeBufferPool implements a thread-safe buffer pool using channels
// This approach guarantees that buffers are only reused after they're completely done being used
type SafeBufferPool struct {
	bufferChan chan []byte
	maxSize    int
}

// newSafeBufferPool creates a new safe buffer pool
func newSafeBufferPool(poolSize, bufferSize int) *SafeBufferPool {
	pool := &SafeBufferPool{
		bufferChan: make(chan []byte, poolSize),
		maxSize:    bufferSize,
	}

	// Pre-populate the pool with buffers
	for i := 0; i < poolSize; i++ {
		pool.bufferChan <- make([]byte, 0, bufferSize)
	}

	return pool
}

// Get retrieves a buffer from the pool, or creates a new one if pool is empty
func (p *SafeBufferPool) Get(size int) []byte {
	select {
	case buf := <-p.bufferChan:
		// Reuse buffer from pool
		if cap(buf) >= size {
			return buf[:size]
		}
		// Buffer too small, create new one (and don't return the old one)
		return make([]byte, size)
	default:
		// Pool empty, create new buffer
		return make([]byte, size)
	}
}

// Put returns a buffer to the pool (non-blocking)
func (p *SafeBufferPool) Put(buf []byte) {
	if cap(buf) != p.maxSize {
		// Wrong size buffer, don't pool it
		return
	}

	// Reset buffer and try to return to pool
	buf = buf[:0]
	select {
	case p.bufferChan <- buf:
		// Successfully returned to pool
	default:
		// Pool full, let GC handle this buffer
	}
}

// Global safe buffer pool instance
var safeBufferPool = newSafeBufferPool(100, 1024) // 100 buffers of 1KB each

// ringBuffer implements a lock-free ring buffer for MPSC communication
// Multi-Producer Single-Consumer pattern for high-throughput scenarios
type ringBuffer struct {
	buffer []atomic.Pointer[[]byte] // Ring buffer storage with atomic pointers
	mask   uint64                   // Size mask (size must be power of 2)
	head   atomic.Uint64            // Consumer head pointer
	tail   atomic.Uint64            // Producer tail pointer
}

// nextPow2 returns the next power of 2 greater than or equal to x
func nextPow2(x uint64) uint64 {
	if x <= 1 {
		return 1
	}
	return 1 << (64 - bits.LeadingZeros64(x-1))
}

// newRingBuffer creates a new ring buffer with given size (must be power of 2)
func newRingBuffer(size uint64) *ringBuffer {
	// Ensure minimum size for performance
	if size < 64 {
		size = 64
	}

	// Ensure size is power of 2 for optimal performance
	size = nextPow2(size)

	return &ringBuffer{
		buffer: make([]atomic.Pointer[[]byte], size),
		mask:   size - 1,
	}
}

// push attempts to push data to the ring buffer (producer side)
// Returns true if successful, false if buffer is full
// Thread-safe for multiple producers
//
// Design rationale: Lock-free implementation using atomic CAS operations.
// Multiple producers can push concurrently without blocking each other.
//
// CRITICAL FIX: Reserve slot first, then write data to avoid race conditions
// where multiple producers could write to the same slot before CAS.
//
// Why ring buffer:
// - Fixed memory allocation (no GC pressure)
// - Cache-friendly access patterns
// - Power-of-2 sizing enables fast modulo via bitwise AND
func (rb *ringBuffer) push(data []byte) bool {
	// Fast path with CAS loop + bounded check
	for {
		tail := rb.tail.Load()
		head := rb.head.Load()
		size := uint64(len(rb.buffer))

		// Check if buffer is full
		if tail-head >= size {
			return false // Buffer full
		}

		// CRITICAL: Reserve the slot first with CAS
		if rb.tail.CompareAndSwap(tail, tail+1) {
			// Only after successful reservation, copy data
			// This prevents race condition where multiple producers
			// write to same slot before CAS

			// Get buffer from safe pool and copy data
			dataCopy := safeBufferPool.Get(len(data))
			copy(dataCopy, data)

			// Use atomic store to ensure memory visibility
			rb.buffer[tail&rb.mask].Store(&dataCopy)
			return true
		}
		// CAS failed → another producer reserved this slot, retry
	}
}

// pushOwned pushes data to the ring buffer without copying (ownership transfer)
// The caller promises not to reuse the data slice after this call
// Returns true if successful, false if buffer is full
func (rb *ringBuffer) pushOwned(data []byte) bool {
	// Fast path with CAS loop + bounded check
	for {
		tail := rb.tail.Load()
		head := rb.head.Load()
		size := uint64(len(rb.buffer))

		// Check if buffer is full
		if tail-head >= size {
			return false // Buffer full
		}

		// Reserve the slot first with CAS
		if rb.tail.CompareAndSwap(tail, tail+1) {
			// No copy - take ownership of the data slice
			rb.buffer[tail&rb.mask].Store(&data)
			return true
		}
		// CAS failed → another producer reserved this slot, retry
	}
}

// pop attempts to pop data from the ring buffer (consumer side)
// Returns data and true if successful, nil and false if buffer is empty
// Should only be called by single consumer thread
func (rb *ringBuffer) pop() ([]byte, bool) {
	for {
		head := rb.head.Load()
		tail := rb.tail.Load()

		// Check if buffer is empty
		if head >= tail {
			return nil, false
		}

		// Try to reserve the slot
		if rb.head.CompareAndSwap(head, head+1) {
			idx := head & rb.mask
			// Use atomic load to ensure memory visibility
			dataPtr := rb.buffer[idx].Load()
			if dataPtr == nil {
				// Should not happen, but handle gracefully
				continue
			}
			data := *dataPtr
			// Help GC by clearing reference
			rb.buffer[idx].Store(nil)
			return data, true
		}
		// CAS failed, retry (shouldn't happen often with single consumer)
	}
}

// MPSCConsumer handles the single consumer logic for the MPSC pattern.
// This type manages the background goroutine that drains the ring buffer
// and writes data to the log file. It implements adaptive flush timing
// and graceful shutdown for optimal performance.
//
// Note: This type is exported for type safety but should not be used directly.
// Instances are managed internally by the Logger.
type MPSCConsumer struct {
	buffer *ringBuffer
	logger *Logger
	ctx    context.Context
	cancel context.CancelFunc
	ticker *time.Ticker
	wg     sync.WaitGroup
}

// newMPSCConsumer creates a new MPSC consumer with configurable flush timing
func newMPSCConsumer(buffer *ringBuffer, logger *Logger) *MPSCConsumer {
	ctx, cancel := context.WithCancel(context.Background())

	// Configure flush interval
	flushInterval := logger.FlushInterval
	if flushInterval <= 0 {
		flushInterval = 1 * time.Millisecond // Default: high frequency flush
	}

	consumer := &MPSCConsumer{
		buffer: buffer,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
		ticker: time.NewTicker(flushInterval),
	}

	// Start consumer goroutine
	consumer.wg.Add(1)
	go consumer.run()

	return consumer
}

// run executes the consumer loop with optional adaptive timing
func (c *MPSCConsumer) run() {
	defer c.ticker.Stop()
	defer c.wg.Done()

	emptyRounds := 0
	baseInterval := c.ticker.C

	for {
		select {
		case <-c.ctx.Done():
			// Final flush before shutdown
			c.flushAll()
			return
		case <-baseInterval:
			itemsProcessed := c.flushAll()

			// Adaptive flush timing based on buffer activity
			if c.logger.AdaptiveFlush {
				c.adjustFlushTiming(itemsProcessed, &emptyRounds)
			}
		}
	}
}

// adjustFlushTiming implements adaptive flush timing algorithm
func (c *MPSCConsumer) adjustFlushTiming(itemsProcessed int, emptyRounds *int) {
	if itemsProcessed == 0 {
		*emptyRounds++
		// Backoff when buffer is consistently empty
		if *emptyRounds >= 10 {
			// Reduce frequency when idle
			c.ticker.Reset(5 * time.Millisecond)
			*emptyRounds = 0
		}
	} else {
		*emptyRounds = 0
		// Increase frequency when busy
		if itemsProcessed > 10 {
			c.ticker.Reset(500 * time.Microsecond) // Higher frequency for busy periods
		} else {
			// Reset to base interval
			flushInterval := c.logger.FlushInterval
			if flushInterval <= 0 {
				flushInterval = 1 * time.Millisecond
			}
			c.ticker.Reset(flushInterval)
		}
	}
}

// flushAll drains available data from ring buffer to file
// Returns the number of items processed
func (c *MPSCConsumer) flushAll() int {
	itemsProcessed := 0
	// Process all available entries
	for {
		data, ok := c.buffer.pop()
		if !ok {
			break // Buffer empty
		}

		c.writeToFile(data)
		itemsProcessed++
	}
	return itemsProcessed
}

// writeToFile writes data directly to file (consumer is single-threaded)
func (c *MPSCConsumer) writeToFile(data []byte) {
	// Write to file FIRST - this must complete before returning buffer to pool
	if c.logger.currentFile.Load() != nil {
		file := c.logger.currentFile.Load()
		n, err := file.Write(data)
		if err == nil {
			// Update size and check rotation (n from Write() is always >= 0, but be safe)
			if n < 0 {
				n = 0
			}
			newSize := c.logger.bytesWritten.Add(uint64(n)) // #nosec G115 -- n checked for negative values above
			if c.logger.shouldRotate(newSize) {
				c.logger.triggerRotation()
			}
		}
	}

	// Return buffer to safe pool after file write completes
	// This is safe because file.Write() has completed and data is no longer being accessed
	safeBufferPool.Put(data)
}

// stop gracefully stops the consumer
func (c *MPSCConsumer) stop() {
	c.cancel()
	c.wg.Wait() // Wait for consumer to finish
}
