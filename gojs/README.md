# Goja Debug Extension for VS Code

This extension enables debugging of Goja JavaScript runtime scripts in Visual Studio Code.

## Features

- Set breakpoints in JavaScript files
- Step through code (step in, step over, step out)
- Inspect variables and call stack
- Evaluate expressions in debug console

## Usage

1. Install the extension in VS Code
2. Create a `.vscode/launch.json` configuration:

```json
{
    "version": "0.2.0",
    "configurations": [
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

3. Run your script with the gojs CLI in debug mode:
   ```bash
   gojs -d -f script.js
   ```

4. Press F5 in VS Code to start debugging

## Requirements

- Goja runtime with debugger support
- gojs CLI tool with DAP server

## Building the Extension

```bash
npm install
npm run compile
```

To package the extension:
```bash
npm install -g vsce
vsce package
```