# List available recipes
default:
    just --list

# Build the remembery binary
build:
    go build -o remembery .

# Run from source
run:
    go run .

# Install to $GOBIN
install:
    go install

# Run all tests
test:
    go test ./...

# Run go vet
vet:
    go vet ./...

# Format Go source files
fmt:
    gofmt -w .

# Lint: format and run vet + gofmt check
lint:
    gofmt -w .
    go vet ./...
    test -z "$(gofmt -l .)"

# Pre-push local checks (lint, test, build, and a GoReleaser dry run)
pre-push: lint test build release-snapshot

# Tidy go.mod and go.sum
tidy:
    go mod tidy

# Clean build artifacts
clean:
    rm -rf remembery dist/

# Dry-run a GoReleaser build
release-snapshot:
    goreleaser release --snapshot --clean

# Validate .goreleaser.yaml (requires a git remote)
release-check:
    goreleaser check
