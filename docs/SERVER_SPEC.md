# Server Specification

## Overview
The server component is a multi-protocol server that supports HTTP, TCP, and gRPC communications. It acts as a central hub for routing requests between clients and handling client registrations.

## Server Components

### 1. HTTP Server
- **Port**: 8080
- **Endpoints**:
  - `/register`: Handles client registration
  - `/clients`: Lists all registered clients and their paths
  - `/*`: Wildcard route that forwards requests to appropriate TCP clients

### 2. TCP Server
- **Port**: 8081
- **Features**:
  - Maintains persistent connections with clients
  - Handles client registration
  - Routes HTTP requests to appropriate TCP clients
  - Supports bidirectional communication

### 3. gRPC Server
- **Port**: 50051
- **Services**:
  - Registration service
  - Path registration service

## Data Structures

### TCPManager
```go
type TCPManager struct {
    Clients     map[string]*TCPClient
    InvertedMap map[string]*net.Conn
}
```

### TCPMessage
```go
type TcpMessage struct {
    Sub       string          // Message subject/type
    Msg       json.RawMessage // Message payload
    RequestId int32          // Request identifier
}
```

### ResponseManager
```go
type ResponseManager struct {
    RequestId  int
    Response   []byte
    StatusCode int
}
```

## Message Types

### 1. Registration
- **Subject**: "REG" or "register"
- **Payload**:
  ```json
  {
    "client_id": "string",
    "Paths": ["string"]
  }
  ```

### 2. Request
- **Subject**: "REQUEST"
- **Payload**:
  ```json
  {
    "request_id": int,
    "method": "string",
    "path": "string",
    "headers": map[string]string,
    "body": []byte
  }
  ```

### 3. Response
- **Subject**: "RESPONSE"
- **Payload**: Raw response data with status code

## Server Behavior

### Client Registration Process
1. Client connects to TCP server
2. Server sends welcome message
3. Client sends registration message with paths
4. Server registers client in TCPManager
5. Server sends registration confirmation

### Request Routing
1. Server receives HTTP request
2. Looks up appropriate TCP client based on path
3. Forwards request to TCP client
4. Waits for response (timeout: 300 seconds)
5. Returns response to original HTTP client

### Error Handling
- TCP connection errors are logged
- Invalid messages are logged and ignored
- Client disconnections are handled gracefully
- Request timeouts return 504 Gateway Timeout
- Invalid paths return 404 Not Found

## Performance Considerations
- Asynchronous message handling
- Goroutines for concurrent client connections
- Request ID tracking for response matching
- Connection pooling for TCP clients
- Timeout mechanisms for request handling
