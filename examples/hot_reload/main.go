// examples/hot_reload/main.go: Demonstration of Lethe hot reload with Argus
//
// This example shows how Lethe can dynamically update its configuration
// when the configuration file changes, without requiring a restart.
//
// Copyright (c) 2025 AGILira
// Series: an AGILira fragment
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agilira/lethe"
)

func main() {
	fmt.Println("🔥 Lethe Hot Reload Demo with Argus")
	fmt.Println("====================================")

	// Create logger with defaults
	logger, err := lethe.NewWithDefaults("demo.log")
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}
	defer logger.Close()

	// Create sample configuration file for hot reload
	configFile := "lethe_config.json"
	if err := lethe.CreateSampleConfig(configFile); err != nil {
		panic(fmt.Sprintf("Failed to create sample config: %v", err))
	}
	fmt.Printf("📄 Created sample config file: %s\n", configFile)

	// Set up hot reload watcher
	watcher, err := lethe.NewDynamicConfigWatcher(configFile, logger)
	if err != nil {
		panic(fmt.Sprintf("Failed to create config watcher: %v", err))
	}
	defer watcher.Stop()

	// Start watching for config changes
	if err := watcher.Start(); err != nil {
		panic(fmt.Sprintf("Failed to start config watcher: %v", err))
	}

	fmt.Println("🚀 Logger started with hot reload enabled!")
	fmt.Printf("📝 Edit %s to see configuration changes in real-time\n", configFile)
	fmt.Println()
	fmt.Println("Try changing these settings in the config file:")
	fmt.Println("  - max_size_str: \"50MB\" -> \"200MB\"")
	fmt.Println("  - max_age_str: \"7d\" -> \"1d\"")
	fmt.Println("  - compress: true -> false")
	fmt.Println("  - max_backups: 10 -> 5")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop...")
	fmt.Println()

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start a goroutine that continuously writes to the log
	go func() {
		counter := 0
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				counter++
				logMsg := fmt.Sprintf("Demo message #%d - time: %s\n",
					counter, time.Now().Format("15:04:05"))

				// Write to logger
				if _, err := logger.Write([]byte(logMsg)); err != nil {
					fmt.Printf("Error writing to log: %v\n", err)
				}

				// Also print to console
				fmt.Printf("📝 Wrote: %s", logMsg)

				// Print current stats every 10 messages
				if counter%10 == 0 {
					stats := logger.Stats()
					fmt.Printf("📊 Stats - Writes: %d, Size: %d bytes, Rotations: %d\n",
						stats.WriteCount, stats.CurrentFileSize, stats.RotationCount)
				}

			case <-sigChan:
				return
			}
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n🛑 Shutting down...")

	// Show final stats
	stats := logger.Stats()
	fmt.Printf("📊 Final Stats:\n")
	fmt.Printf("   Total writes: %d\n", stats.WriteCount)
	fmt.Printf("   Current file size: %d bytes\n", stats.CurrentFileSize)
	fmt.Printf("   Rotations performed: %d\n", stats.RotationCount)
	fmt.Printf("   Average latency: %d ns\n", stats.AvgLatencyNs)

	if lastConfig := watcher.GetLastConfig(); lastConfig != nil {
		fmt.Printf("   Last config - MaxSize: %s, MaxAge: %s, Compress: %t\n",
			lastConfig.MaxSizeStr, lastConfig.MaxAgeStr, lastConfig.Compress)
	}

	// Cleanup
	_ = os.Remove(configFile)
	fmt.Println("✅ Demo completed!")
}
