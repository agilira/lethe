# Makefile for Lethe - Universal log rotation library
# Copyright (c) 2025 AGILira
# SPDX-License-Identifier: MPL-2.0

.PHONY: help test test-race lint fmt vet golangci-lint gosec build clean install deps examples

# Default target
.DEFAULT_GOAL := help

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOVET := $(GOCMD) vet

# Project parameters
BINARY_NAME := lethe
MAIN_PACKAGE := .
TEST_PACKAGES := ./...

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

help: ## Show this help message
	@echo "$(BLUE)Lethe - Universal log rotation library$(NC)"
	@echo "$(BLUE)=====================================$(NC)"
	@echo ""
	@echo "$(GREEN)Available commands:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(GREEN)Quick start:$(NC)"
	@echo "  make test     - Run tests with race detection"
	@echo "  make lint     - Run all quality checks"
	@echo "  make build    - Build the project"
	@echo "  make clean    - Clean build artifacts"

test: ## Run tests
	@echo "$(BLUE)Running tests...$(NC)"
	$(GOTEST) -v $(TEST_PACKAGES)

test-race: ## Run tests with race detection
	@echo "$(BLUE)Running tests with race detection...$(NC)"
	$(GOTEST) -race -v $(TEST_PACKAGES)

test-coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic $(TEST_PACKAGES)
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

lint: fmt vet golangci-lint gosec ## Run all quality checks

fmt: ## Check code formatting
	@echo "$(BLUE)Checking code formatting...$(NC)"
	@if [ "$$($(GOFMT) -s -l . | wc -l)" -gt 0 ]; then \
		echo "$(RED)❌ Code is not formatted. Run 'make fmt-fix' to fix.$(NC)"; \
		$(GOFMT) -s -l .; \
		exit 1; \
	else \
		echo "$(GREEN)✅ Code is properly formatted$(NC)"; \
	fi

fmt-fix: ## Fix code formatting
	@echo "$(BLUE)Fixing code formatting...$(NC)"
	$(GOFMT) -s -w .
	@echo "$(GREEN)✅ Code formatting fixed$(NC)"

vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	$(GOVET) $(TEST_PACKAGES)
	@echo "$(GREEN)✅ go vet passed$(NC)"

golangci-lint: ## Run golangci-lint
	@echo "$(BLUE)Running golangci-lint...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
		echo "$(GREEN)✅ golangci-lint passed$(NC)"; \
	else \
		echo "$(YELLOW)⚠️  golangci-lint not installed. Install with:$(NC)"; \
		echo "   curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2"; \
		exit 1; \
	fi

gosec: ## Run gosec security scanner
	@echo "$(BLUE)Running gosec security scanner...$(NC)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -fmt sarif -out gosec.sarif ./...; \
		echo "$(GREEN)✅ gosec passed$(NC)"; \
	else \
		echo "$(YELLOW)⚠️  gosec not installed. Install with:$(NC)"; \
		echo "   go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
		exit 1; \
	fi

build: ## Build the project
	@echo "$(BLUE)Building project...$(NC)"
	$(GOBUILD) -v $(MAIN_PACKAGE)
	@echo "$(GREEN)✅ Build completed$(NC)"

build-examples: ## Build examples
	@echo "$(BLUE)Building examples...$(NC)"
	cd examples && $(GOBUILD) -v ./...
	@echo "$(GREEN)✅ Examples built$(NC)"

install: ## Install the binary
	@echo "$(BLUE)Installing $(BINARY_NAME)...$(NC)"
	$(GOCMD) install $(MAIN_PACKAGE)
	@echo "$(GREEN)✅ $(BINARY_NAME) installed$(NC)"

deps: ## Download dependencies
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "$(GREEN)✅ Dependencies updated$(NC)"

clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	$(GOCLEAN)
	rm -f coverage.out coverage.html gosec.sarif
	@echo "$(GREEN)✅ Clean completed$(NC)"

examples: ## Run example tests
	@echo "$(BLUE)Running example tests...$(NC)"
	cd examples && $(GOTEST) -v ./...

bench: ## Run benchmarks
	@echo "$(BLUE)Running benchmarks...$(NC)"
	$(GOTEST) -bench=. -benchmem $(TEST_PACKAGES)

bench-cpu: ## Run CPU benchmarks
	@echo "$(BLUE)Running CPU benchmarks...$(NC)"
	$(GOTEST) -bench=. -benchmem -cpuprofile=cpu.prof $(TEST_PACKAGES)
	@echo "$(GREEN)✅ CPU profile generated: cpu.prof$(NC)"

bench-mem: ## Run memory benchmarks
	@echo "$(BLUE)Running memory benchmarks...$(NC)"
	$(GOTEST) -bench=. -benchmem -memprofile=mem.prof $(TEST_PACKAGES)
	@echo "$(GREEN)✅ Memory profile generated: mem.prof$(NC)"

# CI targets (used by GitHub Actions)
ci-test: test-race
ci-quality: lint
ci-build: build build-examples

# Development helpers
dev-setup: deps ## Setup development environment
	@echo "$(BLUE)Setting up development environment...$(NC)"
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing golangci-lint...$(NC)"; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2; \
	fi
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing gosec...$(NC)"; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
	fi
	@echo "$(GREEN)✅ Development environment ready$(NC)"

# Quick development cycle
dev: fmt-fix test-race ## Quick development cycle (format + test)
	@echo "$(GREEN)✅ Development cycle completed$(NC)"

# Pre-commit checks
pre-commit: lint test-race ## Run pre-commit checks
	@echo "$(GREEN)✅ Pre-commit checks passed$(NC)"

# Release preparation
release-check: clean deps lint test-race build build-examples ## Full release check
	@echo "$(GREEN)✅ Release checks passed$(NC)"
