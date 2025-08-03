# Goja Debug POC

⚠️ **PROOF OF CONCEPT - EXPERIMENTAL** ⚠️

This is a Proof of Concept (POC) for debugging JavaScript code running in the [Goja](https://github.com/dop251/goja) JavaScript runtime using the Debug Adapter Protocol (DAP) and Visual Studio Code.

**Important Note**: This implementation has known limitations and is intended for demonstration purposes. For production debugging needs, consider using the more stable console debugger available in the [Goja fork](https://github.com/arturoeanton/goja), which provides better functionality with fewer issues.

## Overview

This project consists of two main components:
1. **DAP Server** (`dap/`): A Go-based Debug Adapter Protocol server that interfaces with a modified Goja runtime
2. **VS Code Extension** (`gojs/`): A Visual Studio Code extension that provides debugging capabilities for Goja scripts

## Prerequisites

- Go 1.20 or later
- Node.js and npm
- Visual Studio Code
- `vsce` (Visual Studio Code Extension Manager): `npm install -g vsce`

## Setup Instructions

### 1. Clone and Initialize

```bash
git clone https://github.com/arturoeanton/goja-debug-poc.git
cd goja-debug-poc
```

### 2. Build the DAP Server

```bash
cd dap
./build.sh
```

This will create the `gojs` binary that acts as both the JavaScript runtime and the DAP server.

### 3. Build and Install the VS Code Extension

```bash
cd ../gojs
npm install
npm run compile
vsce package
```

This creates a `goja-debug-0.0.1.vsix` file. Install it in VS Code:

```bash
code --install-extension goja-debug-0.0.1.vsix
```

### 4. Configure VS Code Debugging

Create a `.vscode/launch.json` file in your project root:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "type": "goja",
            "request": "attach",
            "name": "Attach to Goja",
            "debugServer": 5678
        },
        {
            "type": "goja",
            "request": "launch",
            "name": "Debug Goja Script",
            "program": "${workspaceFolder}/script.js",
            "stopOnEntry": false,
            "debugServer": 5678
        }
    ]
}
```

## Usage

### Running Scripts Normally

```bash
cd dap
./gojs script.js
```

### Debugging Scripts

#### Method 1: Attach Mode
1. Start the script in debug mode:
   ```bash
   ./gojs -d script.js
   ```
2. Open VS Code in the project folder
3. Set breakpoints in your JavaScript file
4. Press `F5` and select "Attach to Goja"

#### Method 2: Launch Mode  
1. Open VS Code in the project folder
2. Set breakpoints in your JavaScript file
3. Press `F5` and select "Debug Goja Script"

### Custom Debug Port

```bash
./gojs -d -port 9000 script.js
```

Remember to update the `debugServer` port in your launch configuration accordingly.

## Example

Create a file `example.js`:

```javascript
console.log("Starting debug example");

function factorial(n) {
    if (n <= 1) {
        return 1;
    }
    return n * factorial(n - 1);
}

let result = factorial(5);
console.log("Factorial of 5 is:", result);

// Test variables
let obj = { name: "test", value: 42 };
let arr = [1, 2, 3, 4, 5];

console.log("Object:", obj);
console.log("Array:", arr);
```

Debug it:
```bash
./gojs -d example.js
```

## Features

- ✅ **Breakpoints**: Set breakpoints in JavaScript code
- ✅ **Stepping**: Step into, over, and out of functions
- ✅ **Call Stack**: View the current execution stack
- ✅ **Variables**: Basic variable inspection (local and global)
- ✅ **Console Output**: View console.log output in VS Code
- ✅ **Expression Evaluation**: Evaluate expressions in debug console

## Architecture

The implementation follows the Debug Adapter Protocol specification:

1. **Launch/Attach**: VS Code connects to the DAP server
2. **Breakpoints**: VS Code sends breakpoint locations to the adapter
3. **Execution Control**: The adapter controls the Goja runtime execution
4. **Events**: The adapter sends stopped/continued/terminated events
5. **Stack/Variables**: VS Code requests runtime information when paused

## Project Structure

```
├── dap/                    # DAP Server implementation
│   ├── adapter.go          # Main DAP adapter logic
│   ├── protocol.go         # DAP protocol messages
│   ├── main.go             # Entry point and CLI
│   ├── gojs.go             # Goja runtime wrapper
│   ├── build.sh            # Build script
│   ├── test_simple_debug.js # Example test file
│   ├── test_dap_simple.py  # DAP test client
│   └── docs/               # Goja debugger documentation
├── gojs/                   # VS Code extension
│   ├── src/
│   │   ├── extension.ts    # Extension activation
│   │   └── gojaDebug.ts    # Debug adapter implementation
│   ├── package.json        # Extension manifest
│   └── README.md           # Extension documentation
└── script.js               # Sample JavaScript file
```

## Known Limitations

⚠️ **This POC has several known issues that affect debugging reliability:**

### Critical Issues
- **Function Stepping**: Step-into functionality with function calls may not work consistently
- **Stack Synchronization**: VS Code pointer position may become desynchronized with actual execution
- **Expression Evaluation**: Limited expression evaluation while paused (simple variables only)
- **Call Stack Display**: Stack trace information may be incomplete or inaccurate

### General Limitations
- Variable inspection shows basic information only
- No support for conditional breakpoints
- Single-threaded execution only
- Limited object property expansion
- No hot reload support
- Debugging complex closures and async operations is not supported

### Recommended Alternative

For more reliable debugging experience, we recommend using the **console debugger** from the Goja fork:
- Repository: [https://github.com/arturoeanton/goja](https://github.com/arturoeanton/goja)
- Location: `cmd/goja-debug-console/`
- Features: More stable function stepping, better variable inspection, fewer synchronization issues

The console debugger provides a terminal-based debugging interface that's more mature and reliable than this VS Code integration.

## Technical Details

### Modified Goja Runtime

This POC uses a fork of Goja with debugging capabilities:
- Repository: `github.com/arturoeanton/goja`
- Adds debugger hooks and breakpoint support
- Provides stepping and execution control APIs

## Troubleshooting

### Common Issues

1. **Port already in use**: Use `-port` flag with a different port
2. **Can't connect**: Ensure the DAP server is running before starting the VS Code debugger
3. **No breakpoints hit**: Verify file paths match between debugger and runtime
4. **Extension not found**: Make sure the `.vsix` file is properly installed

### Debug Logs

The DAP adapter creates a log file `dap-adapter.log` for troubleshooting.

## Contributing

🤝 **Contributions Welcome!**

This project is a proof of concept with known limitations. If you're interested in improving the VS Code debugging experience for Goja, contributions are welcome!

### How to Contribute

1. **Fork the repository**
2. **Create a feature branch** for your improvements
3. **Test your changes** thoroughly with the provided test scripts
4. **Submit a pull request** with a clear description of the improvements

### Areas for Improvement

- Fix function stepping synchronization issues
- Improve stack trace accuracy
- Enhance expression evaluation capabilities
- Add support for conditional breakpoints
- Better error handling and recovery
- Performance optimizations

### Development Setup

See the setup instructions above to get the development environment running. The `dap/` directory contains test scripts that can help validate improvements.

## License

MIT