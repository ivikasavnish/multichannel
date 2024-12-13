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
	"reflect"
	"strings"
	_ "sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Llongfile)
}

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
		writer := typedefs.NewTcpMessageWriter(conn)
		// Send welcome message in JSON format
		welcome := typedefs.TcpMessage{
			Sub: "WELCOME",
			Msg: []byte("Connected to TCP server"),
		}
		if err := writer.WriteMessage(&welcome); err != nil {
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
	reader := typedefs.NewTcpMessageReader(*conn)

	msg, err := reader.ReadMessage()
	if err != nil {
		log.Printf("Error reading message: %v", err)
		return err
	}

	switch msg.Sub {
	case "REG", "register":
		// Cast the message to a map

		var reg map[string]interface{}
		err := json.Unmarshal(msg.Msg, &reg)
		if err != nil {
			log.Printf("Error unmarshalling registration message: %v", err)
			return err
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
			Msg: []byte("Registration successful"),
		}
		if err := json.NewEncoder(*conn).Encode(response); err != nil {
			log.Printf("Error sending registration response: %v", err)
			return err
		}

	case "RESPONSE":
		// Handle response from client for HTTP request
		if msg.Msg == nil {

			log.Printf("Error getting response message")
			return nil
		}
		log.Println(reflect.TypeOf(msg.Msg).Kind().String())

		responses[int(msg.RequestId)] = &ResponseManager{
			Requestid:  int(msg.RequestId),
			Response:   msg.Msg,
			StatusCode: 200,
		}

	case "ERROR":
		log.Printf("Error message: %v", string(msg.Msg))
		if msg.Msg == nil {

			log.Printf("Error getting response message")
			return nil
		}
		var resp map[string]interface{}
		err := json.Unmarshal(msg.Msg, &resp)
		if err != nil {
			log.Printf("Error unmarshalling error message: %v", err)
			return err
		}

		responses[int(msg.RequestId)] = &ResponseManager{
			Requestid:  int(msg.RequestId),
			Response:   []byte("error"),
			StatusCode: 500,
		}
		log.Println(reflect.TypeOf(msg.Msg).Kind().String())

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
	msg := typedefs.Request{
		RequestId: currentRequestId,
		Method:    r.Method,
		Path:      r.URL.Path,
		Headers:   headers,
		Body:      body,
	}
	msgpayload, err := json.Marshal(msg)
	if err != nil {
		return
	}
	tcpRequest := typedefs.TcpMessage{
		Sub:       "REQUEST",
		RequestId: currentRequestId,
		Msg:       msgpayload,
	}
	writer := typedefs.NewTcpMessageWriter(*conn)

	err = writer.WriteMessage(&tcpRequest)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error sending TCP request"))
		return
	}
	log.Printf("Sent request to client for path: %s with request ID: %d", path, currentRequestId)

	// Wait for response with timeout
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(1 * time.Millisecond)
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

func tcpMessageHandler(conn *net.Conn) error {

	log.Printf("[%s] Handling new message from %s", (*conn).RemoteAddr())

	reader := typedefs.NewTcpMessageReader(*conn)
	writer := typedefs.NewTcpMessageWriter(*conn)
	msg, err := reader.ReadMessage()
	if err != nil {
		log.Printf("[%s] Error reading message from %s: %v", (*conn).RemoteAddr(), err)
		return err
	}
	switch msg.Sub {
	case "REG":
		var reg map[string]interface{}
		err := json.Unmarshal(msg.Msg, &reg)
		if err != nil {
			log.Printf("[%s] Error unmarshalling registration message: %v", err)
			return err
		}

		clientId, ok := reg["client_id"].(string)
		if !ok {
			log.Printf("[%s] Error getting client_id from registration")
			return nil
		}

		paths, ok := reg["Paths"].([]interface{})
		if !ok {
			log.Printf("[%s] Error getting paths from registration")
			return nil
		}

		log.Printf("[%s] Registering client %s with paths: %v", clientId, paths)
		tcpmanager.Register(clientId, paths, conn)

		response := typedefs.TcpMessage{
			Sub: "REG_RESPONSE",
			Msg: []byte("Registration successful"),
		}
		if err := writer.WriteMessage(&response); err != nil {
			log.Printf("[%s] Error sending registration response: %v", err)
			return err
		}
		log.Println("Registered client with id", clientId, "messege sent to client")

	case "HEARTBEAT":
		log.Printf("[%s] Received heartbeat from %s", (*conn).RemoteAddr())
		response := typedefs.TcpMessage{
			Sub: "HEARTBEAT_RESPONSE",
			Msg: []byte("Heartbeat acknowledged"),
		}
		if err := writer.WriteMessage(&response); err != nil {
			log.Printf("[%s] Error sending heartbeat response: %v", err)
			return err
		}

	default:
		log.Printf("[%s] Unknown message type: %s", msg.Sub)
	}
	return nil
}
