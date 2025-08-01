# Goja DAP (Debug Adapter Protocol) Implementation

This directory contains a Debug Adapter Protocol implementation for the Goja JavaScript runtime, allowing you to debug Goja scripts in VS Code and other DAP-compatible editors.

## Components

1. **DAP Server** (`adapter.go`, `protocol.go`, `main.go`) - Implements the Debug Adapter Protocol
2. **gojs CLI** (`gojs.go`) - Command-line tool to run Goja scripts with debugging support
3. **VS Code Extension** (`/gojs/`) - Extension for debugging in Visual Studio Code

## Building

```bash
go build -o gojs .
```

This creates a single binary that can act as both the gojs CLI and the DAP server.

## Usage

### Running Scripts Normally

```bash
./gojs script.js
# or
./gojs -f script.js
```

### Running Scripts in Debug Mode

```bash
./gojs -d -f script.js
```

This will:
1. Start a DAP server on port 5678 (default)
2. Wait for a debugger to connect
3. Display connection instructions for VS Code

### Custom Debug Port

```bash
./gojs -d -port 9000 -f script.js
```

## VS Code Integration

### Installing the Extension

1. Navigate to the `gojs/` directory
2. Run `npm install`
3. Run `npm run compile`
4. Package the extension: `vsce package`
5. Install the generated `.vsix` file in VS Code

### Creating a Debug Configuration

Create `.vscode/launch.json` in your project:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "type": "goja",
            "request": "launch",
            "name": "Debug Goja Script",
            "program": "${workspaceFolder}/test.js",
            "stopOnEntry": false,
            "debugServer": 5678
        }
    ]
}
```

### Debugging Workflow

1. Start your script with `gojs -d -f test.js`
2. Open the folder in VS Code
3. Set breakpoints in your JavaScript file
4. Press F5 to start debugging

## Features

- **Breakpoints**: Set breakpoints in your JavaScript code
- **Stepping**: Step into, over, and out of functions
- **Call Stack**: View the current call stack
- **Variables**: Inspect local variables (basic implementation)
- **Console Output**: View console.log output in VS Code

## Architecture

The implementation follows the DAP specification:

1. **Launch**: VS Code sends a launch request with the script path
2. **Breakpoints**: VS Code sends breakpoint locations
3. **Execution**: The adapter controls the Goja runtime
4. **Events**: The adapter sends stopped/terminated events
5. **Stack/Variables**: VS Code requests runtime information

## Limitations

- Variable inspection is simplified (full implementation would require deeper Goja integration)
- No support for conditional breakpoints yet
- Single-threaded execution (Goja limitation)
- No hot reload support

## Example Script

See `test.js` for a sample script demonstrating various JavaScript features that can be debugged.

## Troubleshooting

1. **Port already in use**: Use a different port with `-port` flag
2. **Can't connect**: Ensure the DAP server is running before starting VS Code debugger
3. **No breakpoints hit**: Verify the file paths match between the debugger and runtime