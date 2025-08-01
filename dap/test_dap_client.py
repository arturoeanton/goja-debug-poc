#!/usr/bin/env python3
"""
Simple DAP client to test the debugger server directly
without VS Code to isolate if the issue is in the plugin
"""

import socket
import json
import time

class DAPClient:
    def __init__(self, host='localhost', port=5678):
        self.host = host
        self.port = port
        self.seq = 1
        self.sock = None
        
    def connect(self):
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.sock.connect((self.host, self.port))
        print(f"Connected to {self.host}:{self.port}")
        
    def send_request(self, command, arguments=None):
        request = {
            "seq": self.seq,
            "type": "request",
            "command": command
        }
        if arguments:
            request["arguments"] = arguments
            
        self.seq += 1
        
        # Serialize to JSON
        body = json.dumps(request).encode('utf-8')
        
        # Send with Content-Length header
        header = f"Content-Length: {len(body)}\r\n\r\n".encode('utf-8')
        self.sock.send(header + body)
        
        print(f"Sent: {command}")
        
    def read_response(self):
        # Read header
        header = b''
        while not header.endswith(b'\r\n\r\n'):
            header += self.sock.recv(1)
            
        # Parse content length
        content_length = int(header.decode('utf-8').split(':')[1].strip())
        
        # Read body
        body = self.sock.recv(content_length)
        response = json.loads(body.decode('utf-8'))
        
        print(f"Received: {response.get('type', '')} {response.get('command', '')} {response.get('event', '')}")
        return response
        
    def test_sequence(self):
        # Initialize
        self.send_request("initialize", {
            "clientID": "test-client",
            "adapterID": "goja",
            "linesStartAt1": True,
            "columnsStartAt1": True
        })
        self.read_response()  # initialize response
        self.read_response()  # initialized event
        
        # Launch
        self.send_request("launch", {
            "program": "/Users/arturoeliasanton/github.com/arturoeanton/goja/examples/debugger/dap/test.js",
            "stopOnEntry": False
        })
        self.read_response()
        
        # Set breakpoint at line 4
        self.send_request("setBreakpoints", {
            "source": {
                "path": "/Users/arturoeliasanton/github.com/arturoeanton/goja/examples/debugger/dap/test.js"
            },
            "breakpoints": [{"line": 4}]
        })
        self.read_response()
        
        # Configuration done
        self.send_request("configurationDone")
        self.read_response()
        
        # Wait for stopped event
        print("\nWaiting for breakpoint...")
        while True:
            resp = self.read_response()
            if resp.get('event') == 'stopped':
                print(f"Stopped at: {resp}")
                break
                
        # Get threads
        self.send_request("threads")
        self.read_response()
        
        # Step over
        print("\nStepping over...")
        self.send_request("next", {"threadId": 1})
        self.read_response()
        
        # Continue reading events
        print("\nReading events after step over...")
        for i in range(10):
            try:
                resp = self.read_response()
                if resp.get('event') == 'stopped':
                    print(f"Stopped at position: {resp}")
                    # Auto continue
                    self.send_request("next", {"threadId": 1})
                    self.read_response()
            except:
                break
                
if __name__ == "__main__":
    client = DAPClient()
    client.connect()
    client.test_sequence()