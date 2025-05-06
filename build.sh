#!/bin/bash
mkdir -p dist

# Windows
GOOS=windows GOARCH=amd64 go build -o dist/ip-radar-windows-amd64.exe
GOOS=windows GOARCH=386 go build -o dist/ip-radar-windows-386.exe

# Linux
GOOS=linux GOARCH=amd64 go build -o dist/ip-radar-linux-amd64
GOOS=linux GOARCH=arm64 go build -o dist/ip-radar-linux-arm64

# macOS
GOOS=darwin GOARCH=amd64 go build -o dist/ip-radar-macos-amd64
GOOS=darwin GOARCH=arm64 go build -o dist/ip-radar-macos-arm64

echo "The compilation is complete, and the output file is in the dist directory"