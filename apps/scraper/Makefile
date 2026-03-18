.PHONY: build run test clean install-playwright

# Build the scraper binary
build:
	go build -o bin/scraper ./cmd/scraper

# Run the scraper
run: build
	./bin/scraper

# Run with environment
dev:
	DATABASE_URL="postgresql://postgres:TYsKnvXOw29eTzou@db.jtyabjocvdimkayglzhq.supabase.co:5432/postgres" \
	REDIS_ADDR="localhost:6379" \
	POOL_SIZE=3 \
	go run ./cmd/scraper

# Install Playwright browsers
install-playwright:
	go run github.com/playwright-community/playwright-go/cmd/playwright@latest install --with-deps chromium

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Tidy dependencies
tidy:
	go mod tidy
