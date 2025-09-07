// config_loader.go: Dynamic configuration hot reload using Argus
//
// Copyright (c) 2025 AGILira
// Series: an AGILira fragment
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/agilira/argus"
)

// DynamicConfigWatcher manages dynamic configuration changes using Argus
// Provides real-time hot reload of Lethe configuration with audit trail
type DynamicConfigWatcher struct {
	configPath string
	logger     *Logger
	watcher    *argus.Watcher
	enabled    int32                        // Use atomic int32 instead of bool for thread safety
	mu         sync.Mutex                   // Protect start/stop operations
	lastConfig atomic.Pointer[LoggerConfig] // Keep track of last applied config
}

// NewDynamicConfigWatcher creates a new dynamic config watcher for Lethe
// This enables runtime configuration changes by watching the configuration file
//
// Parameters:
//   - configPath: Path to the JSON configuration file to watch
//   - logger: The Lethe instance to update
//
// Example usage:
//
//	logger, err := lethe.NewWithDefaults("app.log")
//	if err != nil {
//	    return err
//	}
//	defer logger.Close()
//
//	watcher, err := lethe.NewDynamicConfigWatcher("config.json", logger)
//	if err != nil {
//	    return err
//	}
//	defer watcher.Stop()
//
//	if err := watcher.Start(); err != nil {
//	    return err
//	}
//
// Now when you modify config.json and change the configuration fields,
// the logger will automatically update its settings without restart!
func NewDynamicConfigWatcher(configPath string, logger *Logger) (*DynamicConfigWatcher, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); err != nil {
		return nil, fmt.Errorf("config file does not exist: %w", err)
	}

	// Create Argus watcher with production-ready configuration
	config := argus.Config{
		PollInterval:         2 * time.Second, // Fast response for dev, efficient for prod
		OptimizationStrategy: argus.OptimizationAuto,

		// Enable audit trail for configuration changes
		Audit: argus.AuditConfig{
			Enabled:       true,
			OutputFile:    "lethe-config-audit.jsonl",
			MinLevel:      argus.AuditInfo, // Capture all config changes
			BufferSize:    1000,
			FlushInterval: 5 * time.Second, // Faster flush for testing
		},

		// Error handling for config watcher
		ErrorHandler: func(err error, path string) {
			// Log errors through the logger's error callback if available
			if logger.ErrorCallback != nil {
				logger.ErrorCallback("config_watcher", fmt.Errorf("config watcher error for %s: %w", path, err))
			}
		},
	}

	watcher := argus.New(*config.WithDefaults())

	return &DynamicConfigWatcher{
		configPath: configPath,
		logger:     logger,
		watcher:    watcher,
		enabled:    0, // 0 = false, 1 = true for atomic int32
	}, nil
}

// Start begins watching the configuration file for changes
func (w *DynamicConfigWatcher) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if atomic.LoadInt32(&w.enabled) != 0 {
		return fmt.Errorf("watcher is already started")
	}

	// Load and store initial configuration
	initialConfig, err := LoadFromJSONFile(w.configPath)
	if err != nil {
		// Don't fail on initial load error - just continue with current config
		fmt.Fprintf(os.Stderr, "[LETHE] Warning: failed to load initial config from %s: %v\n", w.configPath, err)
	} else {
		w.lastConfig.Store(initialConfig)
		// Apply initial configuration if possible
		if err := w.applyConfigToLogger(initialConfig); err != nil {
			fmt.Fprintf(os.Stderr, "[LETHE] Warning: failed to apply initial config: %v\n", err)
		}
	}

	// Set up config file watcher with hot reload callback
	if err := w.watcher.Watch(w.configPath, func(event argus.ChangeEvent) {
		// Load and parse the updated configuration
		newConfig, err := LoadFromJSONFile(event.Path)
		if err != nil {
			if w.logger.ErrorCallback != nil {
				w.logger.ErrorCallback("config_reload", fmt.Errorf("failed to reload config from %s: %w", event.Path, err))
			}
			return
		}

		// Apply the new configuration to the logger
		if err := w.applyConfigToLogger(newConfig); err != nil {
			if w.logger.ErrorCallback != nil {
				w.logger.ErrorCallback("config_apply", fmt.Errorf("failed to apply new config: %w", err))
			}
			return
		}

		// Store the successfully applied configuration
		w.lastConfig.Store(newConfig)

		// Log successful config reload
		fmt.Fprintf(os.Stderr, "[LETHE] Configuration reloaded from %s\n", event.Path)
	}); err != nil {
		return fmt.Errorf("failed to setup file watcher: %w", err)
	}

	// Start the Argus watcher
	if err := w.watcher.Start(); err != nil {
		return fmt.Errorf("failed to start file watcher: %w", err)
	}
	atomic.StoreInt32(&w.enabled, 1) // Set to 1 (true)
	return nil
}

// applyConfigToLogger applies configuration changes to the logger
// This method handles the safe update of logger configuration at runtime
func (w *DynamicConfigWatcher) applyConfigToLogger(config *LoggerConfig) error {
	// Note: Some configuration changes cannot be applied to a running logger
	// (like filename, buffer size, etc.) as they would require a complete restart.
	// For now, we focus on runtime-changeable settings.

	// Runtime-changeable settings that can be safely updated:

	// 1. Size-based rotation settings
	if config.MaxSizeStr != "" {
		if size, err := ParseSize(config.MaxSizeStr); err == nil {
			w.logger.maxSizeBytes = size
			w.logger.MaxSizeStr = config.MaxSizeStr
		}
	} else if config.MaxSize > 0 {
		w.logger.maxSizeBytes = config.MaxSize * 1024 * 1024 // MB to bytes
		w.logger.MaxSize = config.MaxSize
	}

	// 2. Age-based rotation settings
	if config.MaxAgeStr != "" {
		if duration, err := ParseDuration(config.MaxAgeStr); err == nil {
			w.logger.MaxAge = duration
			w.logger.MaxAgeStr = config.MaxAgeStr
		}
	} else if config.MaxAge > 0 {
		w.logger.MaxAge = config.MaxAge
	}

	// 3. File age cleanup settings
	if config.MaxFileAge > 0 {
		w.logger.MaxFileAge = config.MaxFileAge
	}

	// 4. Backup retention
	if config.MaxBackups > 0 {
		w.logger.MaxBackups = config.MaxBackups
	}

	// 5. Feature flags
	w.logger.Compress = config.Compress
	w.logger.Checksum = config.Checksum
	w.logger.LocalTime = config.LocalTime

	// 6. MPSC settings (some can be updated)
	if config.BackpressurePolicy != "" {
		w.logger.BackpressurePolicy = config.BackpressurePolicy
	}

	// 7. Flush settings for adaptive behavior
	if config.FlushInterval > 0 {
		w.logger.FlushInterval = config.FlushInterval
	}
	w.logger.AdaptiveFlush = config.AdaptiveFlush

	// 8. Error callback (if provided)
	if config.ErrorCallback != nil {
		w.logger.ErrorCallback = config.ErrorCallback
	}

	// Note: Some settings like Filename, Async mode, BufferSize cannot be changed
	// at runtime as they would require recreating the logger infrastructure.
	// These would require a logger restart to take effect.

	return nil
}

// Stop stops watching the configuration file
func (w *DynamicConfigWatcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if atomic.LoadInt32(&w.enabled) == 0 {
		return fmt.Errorf("watcher is not started")
	}

	// Stop the Argus watcher
	if err := w.watcher.Stop(); err != nil {
		return fmt.Errorf("failed to stop file watcher: %w", err)
	}
	atomic.StoreInt32(&w.enabled, 0) // Set to 0 (false)
	return nil
}

// IsRunning returns true if the watcher is currently active
func (w *DynamicConfigWatcher) IsRunning() bool {
	return atomic.LoadInt32(&w.enabled) != 0
}

// GetLastConfig returns the last successfully applied configuration
// Returns nil if no configuration has been applied yet
func (w *DynamicConfigWatcher) GetLastConfig() *LoggerConfig {
	return w.lastConfig.Load()
}

// EnableDynamicConfig creates and starts a config watcher for the given logger and config file
// This is a convenience function that combines NewDynamicConfigWatcher + Start
//
// Example:
//
//	logger, err := lethe.NewWithDefaults("app.log")
//	if err != nil {
//	    return err
//	}
//	defer logger.Close()
//
//	watcher, err := lethe.EnableDynamicConfig(logger, "config.json")
//	if err != nil {
//	    log.Printf("Dynamic config disabled: %v", err)
//	} else {
//	    defer watcher.Stop()
//	    log.Println("✅ Dynamic configuration changes enabled!")
//	}
func EnableDynamicConfig(logger *Logger, configPath string) (*DynamicConfigWatcher, error) {
	watcher, err := NewDynamicConfigWatcher(configPath, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic config watcher: %w", err)
	}

	if err := watcher.Start(); err != nil {
		return nil, fmt.Errorf("failed to start dynamic config watcher: %w", err)
	}

	return watcher, nil
}

// CreateSampleConfig creates a sample configuration file for hot reload testing
// This utility function helps users get started with hot reload functionality
//
// Parameters:
//   - filename: Path where to create the sample config file
//
// Example:
//
//	err := lethe.CreateSampleConfig("lethe-config.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Now you can use this config file with hot reload
//	logger, _ := lethe.NewWithDefaults("app.log")
//	watcher, _ := lethe.EnableDynamicConfig(logger, "lethe-config.json")
//	defer watcher.Stop()
func CreateSampleConfig(filename string) error {
	// Write a clean JSON config
	configWithComments := `{
  "filename": "app.log",
  "max_size_str": "100MB",
  "max_age_str": "7d", 
  "max_backups": 10,
  "compress": true,
  "checksum": false,
  "local_time": true,
  "backpressure_policy": "adaptive",
  "adaptive_flush": true
}`

	if err := os.WriteFile(filename, []byte(configWithComments), 0644); err != nil {
		return fmt.Errorf("failed to write sample config: %w", err)
	}

	return nil
}
