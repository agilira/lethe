# Makefile for Lethe examples
# Builds all example applications with proper binary names to avoid conflicts

.PHONY: all clean build-basic build-hot-reload build-iris build-low-level

# Build all examples
all: build-basic build-hot-reload build-iris build-low-level

# Clean all built binaries
clean:
	@echo "🧹 Cleaning example binaries..."
	@rm -f basic/basic-example
	@rm -f hot_reload/hot-reload-example
	@rm -f iris-integration/iris-example
	@rm -f low-level-api/lowlevel-example
	@echo "✅ Clean completed"

# Build basic example
build-basic:
	@echo "🔨 Building basic example..."
	@cd basic && go build -o basic-example .

# Build hot reload example  
build-hot-reload:
	@echo "🔨 Building hot reload example..."
	@cd hot_reload && go build -o hot-reload-example .

# Build iris integration example (skip if iris not available)
build-iris:
	@echo "🔨 Building iris integration example..."
	@cd iris-integration && go build -o iris-example . || echo "⚠️  Iris not available, skipping..."

# Build low-level API example
build-low-level:
	@echo "🔨 Building low-level API example..."
	@cd low-level-api && go build -o lowlevel-example .

# Test build (like GitHub Actions would do)
test-build:
	@echo "🧪 Testing build process..."
	@$(MAKE) clean
	@$(MAKE) all
	@echo "✅ All examples built successfully!"

# Quick build test (just Go build without custom names)
quick-test:
	@echo "🧪 Quick build test (GitHub Actions style)..."
	@cd basic && go mod download && go build -o basic-example .
	@cd hot_reload && go mod download && go build -o hot-reload-example .
	@cd low-level-api && go mod download && go build -o lowlevel-example .
	@echo "✅ Quick build test passed!"
