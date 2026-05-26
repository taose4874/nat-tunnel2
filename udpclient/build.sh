#!/bin/bash
mkdir -p bin
echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o bin/natun-linux-amd64
echo "Building for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -o bin/natun-darwin-amd64
echo "Building for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -o bin/natun-darwin-arm64
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -o bin/natun-windows-amd64.exe
echo "Build complete! Output in bin/"
