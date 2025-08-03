#!/bin/bash
# Goja Debug Console wrapper script
# This ensures proper terminal configuration

# Save current terminal settings
OLD_STTY=$(stty -g 2>/dev/null)

# Function to restore terminal on exit
cleanup() {
    if [ -n "$OLD_STTY" ]; then
        stty "$OLD_STTY" 2>/dev/null
    fi
}

# Set up trap to restore terminal on exit
trap cleanup EXIT INT TERM

# Configure terminal for proper operation
stty sane 2>/dev/null
stty echo 2>/dev/null
stty icanon 2>/dev/null

# Get the directory where this script is located
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Run the debugger
"$DIR/goja-debug" "$@"