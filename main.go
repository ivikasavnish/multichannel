package main

import (
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"io"
	"log"
	"multichannel/cmd/typedefs"
	"multichannel/grpc/server"
	"multichannel/http/handler"
	pb "multichannel/proto"
	"net"
	"net/http"
)

var (
	tcpmanager = NewTCPManager()
	serverblock = &ServerBlock{
		Host:       "localhost",
		HTTP:       8080,
		TCP:        8081,
		TCPManager: tcpmanager,
	}
	requestid int32 = 0
)

type ServerBlock struct {
	Host       string
	HTTP       int
	TCP        int
	TCPManager *TCPManager
}

type TCPManager struct {
	connections map[string]net.Conn
}

func NewTCPManager() *TCPManager {
	return &TCPManager{
		connections: make(map[string]net.Conn),
	}
}

func (s *ServerBlock) TCPListen() {
	log.Printf("Starting TCP server with port: %d", s.TCP)
	address := fmt.Sprintf("127.0.0.1:%d", s.TCP)
	log.Printf("Using server address: %s", address)
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		log.Printf("TCP server failed to start: %v", err)
		return
	}
	log.Printf("TCP server successfully started on %s", address)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("TCP connection error: %v", err)
			continue
		}
		conn.Write([]byte("Connected to TCP server"))
		go handleTCPConnection(&conn)
	}
}

func handleTCPConnection(c *net.Conn) {
	conn := *c
	defer conn.Close()
	for {
		err := handleTCPMessage(&conn)
		if err != nil {
			if err == io.EOF {
				log.Println("Connection closed by client")
				break
			}
			log.Printf("Error handling message: %v", err)
			break
		}
	}
}

func handleTCPMessage(conn *net.Conn) error {
	var msg typedefs.TcpMessage
	decoder := json.NewDecoder(*conn)
	if err := decoder.Decode(&msg); err != nil {
		return err
	}

	log.Printf("Server received message type: %s at line 84", msg.Sub)

	// Process the message based on its type
	switch msg.Sub {
	case "REG", "register":  // Handle both REG and register for compatibility
		// Handle registration
		response := typedefs.TcpMessage{
			Sub: "REG_RESPONSE",  // Changed to be more specific
			Msg: "Registration successful",
		}
		log.Printf("Server sending response type: %s at line 94", response.Sub)
		return json.NewEncoder(*conn).Encode(response)
	default:
		log.Printf("Unknown message type: %s at line 97", msg.Sub)
	}
	return nil
}

func main() {
	// Start TCP server
	go serverblock.TCPListen()

	// Start gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	registerServer := server.NewRegisterServer()
	pb.RegisterRegisterServiceServer(grpcServer, registerServer)

	go func() {
		log.Printf("Starting gRPC server on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Create gRPC client connection for HTTP handler
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	grpcClient := pb.NewRegisterServiceClient(conn)
	registerHandler := handler.NewRegisterHandler(grpcClient)

	// Setup HTTP server
	http.HandleFunc("/register", registerHandler.Handle)

	// Start HTTP server
	log.Printf("Starting HTTP server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}
