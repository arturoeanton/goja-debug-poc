#!/bin/bash

echo "Starting DAP server on port 5678..."
echo "This server will wait for VS Code to connect and send a launch request."
echo ""
echo "To use:"
echo "1. Open VS Code in this directory"
echo "2. Make sure .vscode/launch.json exists"
echo "3. Press F5 to start debugging"
echo ""

# Start the DAP server in server mode
go run . -port 5678