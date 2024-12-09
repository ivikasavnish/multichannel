package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"multichannel/cmd/typedefs"
	"multichannel/grpc/server"
	"multichannel/http/handler"
	pb "multichannel/proto"
	"net"
	"net/http"
	"strings"
	_ "sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
)

var (
	tcpmanager  = NewTCPManager()
	serverblock = &ServerBlock{
		Host:       "localhost",
		HTTP:       8080,
		TCP:        8081,
		TCPManager: tcpmanager,
	}
	requestid int32 = 0
	responses       = make(map[int]*ResponseManager)
)

type ServerBlock struct {
	Host       string
	HTTP       int
	TCP        int
	TCPManager *TCPManager
}

type TCPClient struct {
	ClientId string
	Conn     *net.Conn
	Paths    []string
}

type TCPManager struct {
	Clients     map[string]*TCPClient // clientId -> client info
	InvertedMap map[string]*net.Conn  // path -> connection
}

func NewTCPManager() *TCPManager {
	return &TCPManager{
		Clients:     make(map[string]*TCPClient),
		InvertedMap: make(map[string]*net.Conn),
	}
}

func (m *TCPManager) Register(id string, paths []interface{}, conn *net.Conn) {
	log.Println("Registering paths:", paths)
	pathSlice := make([]string, len(paths))
	for i, path := range paths {
		pathSlice[i] = path.(string)
	}

	m.Clients[id] = &TCPClient{
		ClientId: id,
		Conn:     conn,
		Paths:    pathSlice,
	}
	log.Printf("Registered client with id: %s", id)

	// Update inverted map for quick path lookup
	for _, path := range pathSlice {
		m.InvertedMap[path] = conn
	}
	serverblock.TCPManager = m
}

func (s *ServerBlock) TCPListen() {
	log.Printf("Starting TCP server with port: %d", s.TCP)
	address := fmt.Sprintf("127.0.0.1:%d", s.TCP)
	log.Printf("Using server address: %s", address)
	listener, err := net.Listen("tcp", address)
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
		// Send welcome message in JSON format
		welcome := typedefs.TcpMessage{
			Sub: "WELCOME",
			Msg: "Connected to TCP server",
		}
		welcomeData, err := json.Marshal(welcome)
		if err != nil {
			log.Printf("Error marshalling welcome message: %v", err)
			continue
		}
		if _, err := conn.Write(welcomeData); err != nil {
			log.Printf("Error sending welcome message: %v", err)
			continue
		}
		go handleTCPConnection(&conn)
	}
}

func handleTCPConnection(c *net.Conn) {
	conn := *c
	// defer conn.Close()
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
	// Create a buffer to read data
	var fullMessage []byte
	buf := make([]byte, 1024)

	// Read the first chunk
	n, err := (*conn).Read(buf)
	if err != nil {
		return err
	}
	fullMessage = append(fullMessage, buf[:n]...)

	var msg typedefs.TcpMessage
	if err := json.Unmarshal(fullMessage, &msg); err != nil {
		log.Printf("Error unmarshalling message: %v", err)
		return err
	}

	log.Printf("Server received message type: %s", msg.Sub)

	switch msg.Sub {
	case "REG", "register":
		// Cast the message to a map
		reg, ok := msg.Msg.(map[string]interface{})
		if !ok {
			log.Printf("Error casting message to RegisterRequest")
			return nil
		}

		clientId, ok := reg["client_id"].(string)
		if !ok {
			log.Printf("Error getting client_id from registration")
			return nil
		}

		paths, ok := reg["Paths"].([]interface{})
		if !ok {
			log.Printf("Error getting paths from registration")
			return nil
		}

		log.Printf("Registering client %s with paths: %v", clientId, paths)
		tcpmanager.Register(clientId, paths, conn)

		response := typedefs.TcpMessage{
			Sub: "REG_RESPONSE",
			Msg: "Registration successful",
		}
		if err := json.NewEncoder(*conn).Encode(response); err != nil {
			log.Printf("Error sending registration response: %v", err)
			return err
		}

	case "RESPONSE":
		// Handle response from client for HTTP request
		resp, ok := msg.Msg.(map[string]interface{})
		if !ok {
			log.Printf("Error casting response message")
			return nil
		}

		requestId, ok := resp["request_id"].(float64)
		if !ok {
			log.Printf("Error getting request_id from response")
			return nil
		}

		// Store the response for the HTTP handler to pick up
		responseManager := &ResponseManager{
			Requestid:  int(requestId),
			Response:   []byte(fmt.Sprintf("%v", resp["body"])),
			StatusCode: int(resp["status_code"].(float64)),
		}
		responses[int(requestId)] = responseManager

	default:
		log.Printf("Unknown message type: %s", msg.Sub)
	}
	return nil
}

func WildRoute(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		// Handle the root path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Root path accessed"))
		return
	}

	// Handle other paths
	log.Printf("Searching for TCP handler for path: %s", r.URL.Path)
	paths := strings.Split(r.URL.Path, "/")
	if len(paths) < 2 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Path not found"))
		return
	}

	path := "/" + paths[1]
	log.Printf("Looking up handler for path: %s", path)
	conn, exists := tcpmanager.InvertedMap[path]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("No handler registered for this path"))
		return
	}

	// Get request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error reading request body"))
		return
	}

	// Create headers map
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Generate unique request ID
	currentRequestId := atomic.AddInt32(&requestid, 1)

	// Create TCP message
	tcpRequest := typedefs.TcpMessage{
		Sub: "REQUEST",
		Msg: map[string]interface{}{
			"request_id": currentRequestId,
			"method":     r.Method,
			"path":       r.URL.Path,
			"headers":    headers,
			"body":       string(body),
		},
	}

	// Send request to TCP client
	if err := json.NewEncoder(*conn).Encode(tcpRequest); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error sending request to handler"))
		return
	}

	// Wait for response with timeout
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			w.WriteHeader(http.StatusGatewayTimeout)
			w.Write([]byte("Request timed out"))
			return
		case <-ticker.C:
			if response, ok := responses[int(currentRequestId)]; ok {
				w.WriteHeader(response.StatusCode)
				w.Write(response.Response)
				delete(responses, int(currentRequestId)) // Clean up
				return
			}
		}
	}
}

func ClientsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Create a more readable response structure
	type ClientInfo struct {
		ClientId string   `json:"client_id"`
		Paths    []string `json:"registered_paths"`
	}

	clientList := make([]ClientInfo, 0)
	for _, client := range tcpmanager.Clients {
		clientList = append(clientList, ClientInfo{
			ClientId: client.ClientId,
			Paths:    client.Paths,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_clients": len(clientList),
		"clients":       clientList,
	})
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
	http.HandleFunc("/clients", ClientsHandler)
	http.HandleFunc("/", WildRoute)

	// Start HTTP server
	log.Printf("Starting HTTP server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("failed to serve HTTP: %v", err)
	}
}

type ResponseManager struct {
	Requestid  int
	Response   []byte
	StatusCode int
}
