# Makefile

.PHONY: help install build run test clean

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# ПОМОЩЬ
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

help:
	@echo "Available commands:"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# УСТАНОВКА
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

install:
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies installed"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# BUILD
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

build:
	@echo "🔨 Building project..."
	go build -o bin/botty ./cmd/botty
	@echo "✅ Build complete: bin/botty"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# RUN
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

run: install build
	@echo "Printing environment variables from .env file:"
	@if [ -f .env ]; then sed -e '1s/^\xEF\xBB\xBF//' -e 's/\r$$//' .env; else echo "No .env file found."; fi
	@echo "Exporting environment variables from .env file..."
	@export $$(sed -e '1s/^\xEF\xBB\xBF//' -e 's/\r$$//' .env | grep -v '^#' | xargs); \
	echo "🚀 Running server..."; \
	./bin/botty

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# TESTS
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

test:
	@echo "🧪 Running tests..."
	go test -v ./...

test-coverage:
	@echo "📊 Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# CLEAN
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

clean:
	@echo "🧹 Cleaning..."
	rm -rf bin/
	rm -rf coverage.out
	rm -rf coverage.html
	@echo "✅ Clean complete"
