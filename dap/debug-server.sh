#!/bin/bash

echo "Starting DAP Debug Server"
echo "========================"
echo ""
echo "This will start the Debug Adapter Protocol server on port 5678"
echo "Logs will be shown here to help diagnose any issues"
echo ""
echo "To connect from VS Code:"
echo "1. Make sure the 'Goja Debug' extension is installed"
echo "2. Open this folder in VS Code" 
echo "3. Set breakpoints in test.js"
echo "4. Press F5 to start debugging"
echo ""
echo "Server starting..."
echo ""

# Run with verbose logging to stderr
GODEBUG=gctrace=0 ./gojs -port 5678 2>&1 | while IFS= read -r line; do
    echo "[$(date '+%H:%M:%S')] $line"
done