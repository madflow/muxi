.PHONY: build test

# Build the binary
build:
	@echo "Building..."
	@go build 

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Fix 
fix:
	@echo "Running go fix..."
	@go fix ./...
