#!/bin/bash

echo "Building gojs with DAP support..."

# Build the binary
go build -o gojs .

if [ $? -eq 0 ]; then
    echo "Build successful! Binary created: ./gojs"
    echo ""
    echo "Usage:"
    echo "  Run normally:     ./gojs script.js"
    echo "  Debug mode:       ./gojs -d -f script.js"
    echo "  Custom port:      ./gojs -d -port 9000 -f script.js"
    echo ""
    echo "To install system-wide:"
    echo "  sudo cp gojs /usr/local/bin/"
else
    echo "Build failed!"
    exit 1
fi