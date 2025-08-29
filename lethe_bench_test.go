package lethe

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/agilira/go-timecache"
)

// BenchmarkSyncMode tests performance in synchronous mode
func BenchmarkSyncMode(b *testing.B) {
	testFile := generateTestFile("bench_sync")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename: testFile,
		MaxSize:  100,   // Large enough to avoid rotation during bench
		Async:    false, // Force sync mode
	}
	defer logger.Close()

	data := []byte("Benchmark test message for sync mode\n")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = logger.Write(data) // Ignore errors in benchmark
		}
	})
}

// BenchmarkMPSCMode tests performance in MPSC async mode
func BenchmarkMPSCMode(b *testing.B) {
	testFile := generateTestFile("bench_mpsc")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename: testFile,
		MaxSize:  100,  // Large enough to avoid rotation during bench
		Async:    true, // Force MPSC mode
	}
	defer logger.Close()

	data := []byte("Benchmark test message for MPSC mode\n")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = logger.Write(data) // Ignore errors in benchmark
		}
	})
}

// BenchmarkAutoScalingMode tests performance with auto-scaling enabled
func BenchmarkAutoScalingMode(b *testing.B) {
	testFile := generateTestFile("bench_auto")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename: testFile,
		MaxSize:  100, // Large enough to avoid rotation during bench
		// Note: Async is false, let auto-scaling decide
	}
	defer logger.Close()

	data := []byte("Benchmark test message for auto-scaling mode\n")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = logger.Write(data) // Ignore errors in benchmark
		}
	})
}

// BenchmarkHighContentionSync tests sync mode under high contention
func BenchmarkHighContentionSync(b *testing.B) {
	testFile := generateTestFile("bench_contention_sync")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename: testFile,
		MaxSize:  1, // Force frequent rotation to create contention
		Async:    false,
	}
	defer logger.Close()

	data := make([]byte, 1024) // 1KB message
	for i := range data {
		data[i] = 'A'
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = logger.Write(data) // Ignore errors in benchmark
		}
	})
}

// BenchmarkHighContentionMPSC tests MPSC mode under high contention
func BenchmarkHighContentionMPSC(b *testing.B) {
	testFile := generateTestFile("bench_contention_mpsc")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename: testFile,
		MaxSize:  1, // Force frequent rotation to create contention
		Async:    true,
	}
	defer logger.Close()

	data := make([]byte, 1024) // 1KB message
	for i := range data {
		data[i] = 'A'
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = logger.Write(data) // Ignore errors in benchmark
		}
	})
}

// BenchmarkThroughputComparison provides a realistic throughput test
func BenchmarkThroughputComparison(b *testing.B) {
	scenarios := []struct {
		name  string
		async bool
	}{
		{"Sync", false},
		{"MPSC", true},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			testFile := generateTestFile(fmt.Sprintf("bench_throughput_%s", scenario.name))
			defer cleanupTestFile(testFile)

			logger := &Logger{
				Filename: testFile,
				MaxSize:  100,
				Async:    scenario.async,
			}
			defer logger.Close()

			// Realistic log message
			data := []byte("2025-01-28 10:30:45 INFO [service] Processing request ID=12345 user=john.doe@example.com duration=245ms\n")

			const numGoroutines = 10
			const messagesPerGoroutine = 1000

			b.ResetTimer()

			var wg sync.WaitGroup
			start := make(chan struct{})

			// Start goroutines
			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start // Wait for signal

					for j := 0; j < messagesPerGoroutine; j++ {
						_, _ = logger.Write(data) // Ignore errors in benchmark
					}
				}()
			}

			// Signal all goroutines to start
			close(start)
			wg.Wait()

			b.StopTimer()
		})
	}
}

// BenchmarkTimeCacheVsTimeNow compares performance of timecache vs time.Now()
func BenchmarkTimeCacheVsTimeNow(b *testing.B) {
	b.Run("TimeNow", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = time.Now()
		}
	})

	b.Run("TimeCacheDefault", func(b *testing.B) {
		cache := timecache.DefaultCache()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = cache.CachedTime()
		}
	})

	b.Run("TimeCacheHighRes", func(b *testing.B) {
		cache := timecache.NewWithResolution(time.Millisecond)
		defer cache.Stop()
		b.ResetTimer()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = cache.CachedTime()
		}
	})
}

// BenchmarkLetheWithTimeCacheIntegration tests Lethe's performance with timecache
func BenchmarkLetheWithTimeCacheIntegration(b *testing.B) {
	// Test performance with timecache enabled (automatically initialized)
	b.Run("WithTimeCache", func(b *testing.B) {
		testFile := generateTestFile("bench_timecache_enabled")
		defer cleanupTestFile(testFile)

		logger := &Logger{
			Filename: testFile,
			MaxSize:  100,
			Async:    false, // Force sync mode for controlled testing
		}
		defer logger.Close()

		// Pre-initialize to ensure timecache is active
		_, _ = logger.Write([]byte("init\n")) // Ignore errors in benchmark

		data := []byte("Benchmark test entry for timecache comparison\n")
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := logger.Write(data)
			if err != nil {
				b.Fatalf("Write failed: %v", err)
			}
		}
	})
}

// BenchmarkHighThroughputWithTimeCache tests high-throughput scenarios
func BenchmarkHighThroughputWithTimeCache(b *testing.B) {
	testFile := generateTestFile("bench_high_throughput_timecache")
	defer cleanupTestFile(testFile)

	logger := &Logger{
		Filename:           testFile,
		MaxSizeStr:         "100MB", // Large size to avoid rotation overhead
		Async:              true,    // Enable MPSC for maximum performance
		BufferSize:         4096,    // Large buffer
		BackpressurePolicy: "adaptive",
	}
	defer logger.Close()

	// Pre-initialize
	_, _ = logger.Write([]byte("init\n")) // Ignore errors in benchmark

	data := []byte("High-throughput benchmark entry with timecache optimization\n")

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := logger.Write(data)
		if err != nil {
			b.Fatalf("Write failed: %v", err)
		}
	}
}

// BenchmarkZeroAllocations verifies our zero-allocation optimizations
func BenchmarkZeroAllocations(b *testing.B) {
	b.Run("SyncModeZeroAlloc", func(b *testing.B) {
		testFile := generateTestFile("bench_zero_alloc_sync")
		defer cleanupTestFile(testFile)

		logger := &Logger{
			Filename: testFile,
			MaxSize:  100,
			Async:    false,
		}
		defer logger.Close()

		data := []byte("Zero allocation benchmark test entry\n")
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := logger.Write(data)
			if err != nil {
				b.Fatalf("Write failed: %v", err)
			}
		}

		// Results will show allocations in benchmark output
	})

	b.Run("MPSCModeMinimalAlloc", func(b *testing.B) {
		testFile := generateTestFile("bench_zero_alloc_mpsc")
		defer cleanupTestFile(testFile)

		logger := &Logger{
			Filename:   testFile,
			MaxSize:    100,
			Async:      true,
			BufferSize: 1024,
		}
		defer logger.Close()

		// Pre-initialize
		_, _ = logger.Write([]byte("init\n")) // Ignore errors in benchmark

		data := []byte("Zero allocation benchmark test entry\n")
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := logger.Write(data)
			if err != nil {
				b.Fatalf("Write failed: %v", err)
			}
		}

		// Results will show allocations in benchmark output
	})
}
