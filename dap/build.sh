#!/bin/bash

echo "Building Goja DAP Server..."

# Build the DAP server
go build -o gojs .

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo "   DAP server binary: ./gojs"
    echo ""
    echo "Usage:"
    echo "  ./gojs script.js              # Run script normally"
    echo "  ./gojs -d script.js           # Run in debug mode (port 5678)"
    echo "  ./gojs -d -port 9000 script.js # Run in debug mode (custom port)"
    echo ""
    echo "For VS Code debugging, use a launch.json configuration like:"
    echo '{'
    echo '    "type": "goja",'
    echo '    "request": "launch",'
    echo '    "name": "Debug Goja Script",'
    echo '    "program": "${workspaceFolder}/your-script.js",'
    echo '    "debugServer": 5678'
    echo '}'
else
    echo "❌ Build failed!"
    exit 1
fi