#!/bin/bash
echo "Starting DAP server on port 5678..."
echo "Use VS Code to connect to the debugger"
echo ""
echo "To test manually, use telnet or nc to connect to port 5678"
echo "and send DAP messages"

go run . -server -port 5678