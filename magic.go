// magic.go: Magic API for Lethe with Iris integration
//
// This file provides Iris integration capabilities without importing Iris
// directly, avoiding circular dependencies while providing seamless integration.
//
// Copyright (c) 2025 AGILira
// Series: Lethe - Magic API
// SPDX-License-Identifier: MPL-2.0

package lethe

import (
	"sync"
)

// IrisIntegration provides enhanced capabilities for Iris integration
// This wraps a Lethe logger and exposes the enhanced interface that
// Iris can detect and optimize for.
type IrisIntegration struct {
	logger     *Logger
	mu         sync.RWMutex
	bufferSize int
}

// NewIrisWriter creates a Lethe logger that implements enhanced WriteSyncer
// interface for automatic Iris optimization. This is the Magic API.
//
// Usage with Iris:
//
//	writer := lethe.NewIrisWriter("app.log", &lethe.Logger{
//	    MaxSizeStr: "100MB",
//	    MaxBackups: 5,
//	    Compress:   true,
//	    Async:      true,
//	})
//
//	// Use directly with Iris - automatic optimization!
//	logger := iris.New(iris.Config{Output: writer})
func NewIrisWriter(filename string, config *Logger) *IrisIntegration {
	if config == nil {
		config = &Logger{}
	}

	// Set filename
	config.Filename = filename

	// Set sensible defaults for Iris integration
	if config.MaxSizeStr == "" {
		config.MaxSizeStr = "100MB"
	}
	if config.MaxBackups == 0 {
		config.MaxBackups = 5
	}
	if !config.Async {
		config.Async = true // Enable async for better Iris performance
	}
	if config.BufferSize == 0 {
		config.BufferSize = 8192
	}

	integration := &IrisIntegration{
		logger:     config,
		bufferSize: config.BufferSize,
	}

	return integration
}

// Write implements standard io.Writer interface
func (i *IrisIntegration) Write(data []byte) (int, error) {
	return i.logger.Write(data)
}

// WriteOwned implements zero-copy optimization interface
// This is the key method that Iris detects for Magic API optimization
func (i *IrisIntegration) WriteOwned(data []byte) (int, error) {
	// Use Lethe's zero-copy write path - caller transfers ownership
	return i.logger.WriteOwned(data)
}

// Sync implements WriteSyncer interface
func (i *IrisIntegration) Sync() error {
	// Lethe handles sync internally
	return nil
}

// Close implements WriteSyncer interface
func (i *IrisIntegration) Close() error {
	return i.logger.Close()
}

// GetOptimalBufferSize returns optimal buffer size for Iris tuning
func (i *IrisIntegration) GetOptimalBufferSize() int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.bufferSize
}

// SupportsHotReload indicates hot-reload capability
func (i *IrisIntegration) SupportsHotReload() bool {
	return true // Lethe supports hot-reload through config system
}

// GetLogger returns the underlying Lethe logger
func (i *IrisIntegration) GetLogger() *Logger {
	return i.logger
}

// QuickStart creates a Lethe writer with optimal defaults for Iris
func QuickStart(filename string) *IrisIntegration {
	return NewIrisWriter(filename, &Logger{
		MaxSizeStr:         "100MB",
		MaxBackups:         5,
		Compress:           true,
		Async:              true,
		BufferSize:         8192,
		BackpressurePolicy: "adaptive",
	})
}
