#!/bin/bash

# Script para probar el servidor DAP directamente sin VS Code

echo "Testing DAP server directly..."

# Función para enviar mensaje DAP
send_dap() {
    local message="$1"
    local length=${#message}
    printf "Content-Length: %d\r\n\r\n%s" "$length" "$message"
}

# Conectar al servidor DAP
(
    # Initialize
    send_dap '{"seq":1,"type":"request","command":"initialize","arguments":{"clientID":"test","adapterID":"goja","linesStartAt1":true,"columnsStartAt1":true}}'
    sleep 0.5
    
    # Launch
    send_dap '{"seq":2,"type":"request","command":"launch","arguments":{"program":"/Users/arturoeliasanton/github.com/arturoeanton/goja/examples/debugger/dap/test.js","stopOnEntry":false}}'
    sleep 0.5
    
    # Set breakpoint at line 4
    send_dap '{"seq":3,"type":"request","command":"setBreakpoints","arguments":{"source":{"path":"/Users/arturoeliasanton/github.com/arturoeanton/goja/examples/debugger/dap/test.js"},"breakpoints":[{"line":4}]}}'
    sleep 0.5
    
    # Configuration done
    send_dap '{"seq":4,"type":"request","command":"configurationDone"}'
    sleep 1
    
    # Threads
    send_dap '{"seq":5,"type":"request","command":"threads"}'
    sleep 0.5
    
    # Continue
    send_dap '{"seq":6,"type":"request","command":"continue","arguments":{"threadId":1}}'
    sleep 1
    
    # Next
    send_dap '{"seq":7,"type":"request","command":"next","arguments":{"threadId":1}}'
    sleep 1
    
    # Keep connection open to see more events
    sleep 5
    
) | nc localhost 5678 | while IFS= read -r line; do
    if [[ "$line" =~ Content-Length ]]; then
        echo "---"
    else
        echo "$line" | jq . 2>/dev/null || echo "$line"
    fi
done