package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Llongfile)
}

var (
	tcpmanager  = NewTCPManager()
	serverblock = &ServerBlock{
		Host:       "127.0.0.1", // Using explicit IPv4 address
		HTTP:       8080,
		TCP:        8081,
		TCPManager: tcpmanager,
	}
	requestid int32 = 0
)

type ResponseManager struct {
	Requestid  int
	Response   []byte
	StatusCode int
}

func (r *ResponseManager) CheckResponse() error {
	if r.Response == nil {
		return errors.New("Response is nil")
	}
	return nil
}

func getFileAndLine() string {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return "unknown:0"
	}
	return fmt.Sprintf("%s:%d", file, line)
}

func main() {
	// Configure logging with file and line numbers
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	loc := getFileAndLine()
	log.Printf("[%s] Starting server on %s", loc, serverblock.Host)
	log.Printf("[%s] HTTP port: %d, TCP port: %d", loc, serverblock.HTTP, serverblock.TCP)

	// Start HTTP server
	go func() {
		if err := serverblock.HttpListen(); err != nil {
			log.Printf("[%s] HTTP server error: %v", loc, err)
			os.Exit(1)
		}
	}()

	// Start TCP server
	go func() {
		serverblock.TCPListen()
	}()

	log.Printf("[%s] Server started successfully", loc)
	select {} // Keep the main goroutine running
}
