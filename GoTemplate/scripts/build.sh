#!/bin/bash

# Build the API server
echo "Building API server..."
go build -o bin/api cmd/api/main.go

# Run tests
echo "Running tests..."
go test ./...
