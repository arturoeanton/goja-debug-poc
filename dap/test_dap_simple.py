#!/usr/bin/env python3
"""
Simple DAP test client for the Goja debugger
Tests basic debug functionality
"""

import socket
import json
import time
import sys
import os

class DAPClient:
    def __init__(self, host='localhost', port=5678):
        self.host = host
        self.port = port
        self.sock = None
        self.seq = 1
        
    def connect(self):
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.sock.connect((self.host, self.port))
        print(f"Connected to DAP server at {self.host}:{self.port}")
        
    def close(self):
        if self.sock:
            self.sock.close()
            
    def send_request(self, command, arguments=None):
        request = {
            "seq": self.seq,
            "type": "request",
            "command": command
        }
        if arguments:
            request["arguments"] = arguments
            
        self.seq += 1
        
        msg_json = json.dumps(request)
        msg = f"Content-Length: {len(msg_json)}\r\n\r\n{msg_json}"
        self.sock.send(msg.encode())
        print(f">>> Sent: {command}")
        
    def read_message(self):
        # Read header
        header = b""
        while not header.endswith(b"\r\n\r\n"):
            header += self.sock.recv(1)
            
        # Parse content length
        content_length = 0
        for line in header.decode().split("\r\n"):
            if line.startswith("Content-Length:"):
                content_length = int(line.split(":")[1].strip())
                
        # Read body
        body = self.sock.recv(content_length)
        msg = json.loads(body)
        
        if msg.get("type") == "response":
            print(f"<<< Response: {msg.get('command')} (success: {msg.get('success')})")
        elif msg.get("type") == "event":
            print(f"<<< Event: {msg.get('event')}")
            if msg.get("event") == "output":
                print(f"    Output: {msg['body']['output'].strip()}")
                
        return msg

def test_simple_debug():
    print("Starting simple DAP test...")
    
    # Get the absolute path to the test script
    script_path = os.path.abspath("test_simple_debug.js")
    print(f"Script path: {script_path}")
    
    client = DAPClient()
    
    try:
        client.connect()
        
        # Initialize
        client.send_request("initialize", {
            "clientID": "test-client",
            "adapterID": "goja",
            "pathFormat": "path"
        })
        
        # Read responses until we get initialized event
        while True:
            msg = client.read_message()
            if msg.get("type") == "event" and msg.get("event") == "initialized":
                break
                
        # Launch
        client.send_request("launch", {
            "program": script_path,
            "stopOnEntry": True
        })
        client.read_message()
        
        # Set breakpoints
        client.send_request("setBreakpoints", {
            "source": {
                "path": script_path,
                "name": "test_simple_debug.js"
            },
            "breakpoints": [
                {"line": 8},  # Inside add function
                {"line": 14}, # After calling add
            ]
        })
        client.read_message()
        
        # Configuration done
        client.send_request("configurationDone")
        client.read_message()
        
        # Wait for stopped event
        print("\nWaiting for breakpoint...")
        stopped = False
        while not stopped:
            msg = client.read_message()
            if msg.get("type") == "event" and msg.get("event") == "stopped":
                stopped = True
                print(f"Stopped at breakpoint! Reason: {msg['body']['reason']}")
                
        # Get stack trace
        client.send_request("stackTrace", {"threadId": 1})
        stack_msg = client.read_message()
        print(f"Stack trace: {stack_msg.get('body', {}).get('stackFrames', [])}")
        
        # Continue execution
        print("\nContinuing execution...")
        client.send_request("continue", {"threadId": 1})
        client.read_message()
        
        # Wait for next breakpoint or termination
        terminated = False
        while not terminated:
            msg = client.read_message()
            if msg.get("type") == "event":
                if msg.get("event") == "stopped":
                    print(f"Stopped again! Reason: {msg['body']['reason']}")
                    # Continue again
                    client.send_request("continue", {"threadId": 1})
                    client.read_message()
                elif msg.get("event") == "terminated":
                    terminated = True
                    print("Program terminated")
                    
        print("\nTest completed successfully!")
        
    except Exception as e:
        print(f"Error: {e}")
        
    finally:
        client.close()

if __name__ == "__main__":
    test_simple_debug()