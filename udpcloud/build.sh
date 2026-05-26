#!/bin/bash
mkdir -p bin
echo "Building server for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o bin/server-linux-amd64
echo "Building server for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -o bin/server-darwin-amd64
echo "Building server for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -o bin/server-darwin-arm64
echo "Build complete! Output in bin/"
