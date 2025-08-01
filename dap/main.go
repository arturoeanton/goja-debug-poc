package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
)

func main() {
	// Check if running as gojs CLI
	if len(os.Args) > 0 && filepath.Base(os.Args[0]) == "gojs" {
		runGojs()
		return
	}

	var port int
	var server bool

	flag.IntVar(&port, "port", 0, "Port to listen on (0 for stdio mode)")
	flag.BoolVar(&server, "server", false, "Run in server mode")
	flag.Parse()

	if port == 0 {
		// Stdio mode
		log.SetOutput(os.Stderr)
		adapter := NewDebugAdapter(os.Stdin, os.Stdout)
		adapter.Run()
	} else {
		// Server mode
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			log.Fatalf("Failed to listen on port %d: %v", port, err)
		}
		defer listener.Close()

		log.Printf("Debug adapter listening on port %d", port)

		if server {
			// Accept multiple connections
			for {
				conn, err := listener.Accept()
				if err != nil {
					log.Printf("Failed to accept connection: %v", err)
					continue
				}

				go func(c net.Conn) {
					defer c.Close()
					adapter := NewDebugAdapter(c, c)
					adapter.Run()
				}(conn)
			}
		} else {
			// Accept single connection
			conn, err := listener.Accept()
			if err != nil {
				log.Fatalf("Failed to accept connection: %v", err)
			}
			defer conn.Close()

			adapter := NewDebugAdapter(conn, conn)
			adapter.Run()
		}
	}
}
