package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/dop251/goja"
)

func runGojs() {
	var debugMode bool
	var debugPort int
	var fileName string

	flag.BoolVar(&debugMode, "d", false, "Enable debug mode")
	flag.IntVar(&debugPort, "port", 5678, "Debug adapter port (default: 5678)")
	flag.StringVar(&fileName, "f", "", "JavaScript file to run")
	flag.Parse()

	// Check if file is provided
	if fileName == "" && flag.NArg() > 0 {
		fileName = flag.Arg(0)
	}

	if fileName == "" {
		fmt.Fprintf(os.Stderr, "Usage: gojs [-d] [-port <port>] -f <file.js>\n")
		fmt.Fprintf(os.Stderr, "   or: gojs [-d] [-port <port>] <file.js>\n")
		os.Exit(1)
	}

	// Verify file exists
	if _, err := os.Stat(fileName); err != nil {
		fmt.Fprintf(os.Stderr, "Error: file '%s' not found\n", fileName)
		os.Exit(1)
	}

	if debugMode {
		fmt.Printf("Starting in debug mode on port %d...\n", debugPort)
		fmt.Println("Waiting for debugger to connect...")

		// Get absolute path
		absPath, err := filepath.Abs(fileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting absolute path: %v\n", err)
			os.Exit(1)
		}

		// Start the debug adapter in a goroutine
		go func() {
			// Create and run the debug adapter
			listener, err := net.Listen("tcp", fmt.Sprintf(":%d", debugPort))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to listen on port %d: %v\n", debugPort, err)
				os.Exit(1)
			}
			defer listener.Close()

			fmt.Printf("\nDebug adapter listening on port %d\n", debugPort)
			fmt.Printf("To debug in VS Code:\n")
			fmt.Printf("1. Open VS Code\n")
			fmt.Printf("2. Open the folder containing '%s'\n", fileName)
			fmt.Printf("3. Create or update .vscode/launch.json with:\n")
			fmt.Printf(`{
    "version": "0.2.0",
    "configurations": [
        {
            "type": "goja",
            "request": "launch",
            "name": "Debug Goja Script",
            "program": "%s",
            "debugServer": %d
        }
    ]
}
`, absPath, debugPort)
			fmt.Printf("\n4. Press F5 to start debugging\n")
			fmt.Printf("\nWaiting for debugger connection...\n")

			// Accept connection
			conn, err := listener.Accept()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to accept connection: %v\n", err)
				return
			}
			defer conn.Close()

			// Run the debug adapter
			adapter := NewDebugAdapter(conn, conn)
			adapter.program = absPath
			adapter.Run()
		}()

		// Keep the main process running
		select {}
	} else {
		// Run normally without debugging
		fmt.Printf("Running %s...\n", fileName)

		// Run the script directly using goja
		content, err := os.ReadFile(fileName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}

		vm := goja.New()

		// Set up console.log
		console := vm.NewObject()
		console.Set("log", func(args ...interface{}) {
			fmt.Print("console.log: ")
			for i, arg := range args {
				if i > 0 {
					fmt.Print(" ")
				}
				fmt.Print(arg)
			}
			fmt.Println()
		})
		vm.Set("console", console)

		// Run the script
		result, err := vm.RunScript(fileName, string(content))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Print result if not undefined
		if result != nil && !goja.IsUndefined(result) && !goja.IsNull(result) {
			fmt.Printf("Result: %v\n", result)
		}
	}
}
