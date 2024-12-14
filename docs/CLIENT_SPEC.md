# Client Specification

## Overview
The client component is a multi-protocol client that can communicate with the server via HTTP, TCP, and gRPC. It supports registering paths and handling incoming requests through a callback mechanism.

## Client Components

### 1. HTTP Client
- Handles initial registration with server
- Sends HTTP requests to server
- Configurable timeout and retry mechanisms

### 2. TCP Client
- **Default Port**: 8081
- Maintains persistent connection with server
- Handles incoming requests and sends responses
- Supports message types: REG, REQUEST, RESPONSE, ERROR

### 3. gRPC Client
- **Default Port**: 50051
- Handles registration through gRPC
- Supports path registration

## Configuration

### ClientBlock Structure
```go
type ClientBlock struct {
    Host            string
    HTTP            int
    TCP             int
    GRPC            int
    Paths           []string
    ClientId        string
    callbackRegistry *callbacks.CallbackRegistry
}
```

### Default Configuration
```go
{
    Host: "localhost",
    HTTP: 8080,
    TCP:  8081,
    GRPC: 50051,
    Paths: [
        "/stocks",
        "/weather",
        "/crypto",
        "/ollama"
    ]
}
```

## Callback System

### Registration
```go
callbackRegistry.Register("/path", callbackFunction)
```

### Callback Function Signature
```go
type CallbackFunc func(typedefs.Request) interface{}
```

### Example Callback
```go
func stocksCallback(req typedefs.Request) interface{} {
    return []map[string]interface{}{
        {"symbol": "AAPL", "price": 150.0},
        {"symbol": "GOOG", "price": 2500.0},
    }
}
```

## Message Protocol

### 1. Registration Message
```json
{
    "Sub": "REG",
    "Msg": {
        "client_id": "uuid",
        "Paths": ["/path1", "/path2"]
    }
}
```

### 2. Request Message
```json
{
    "Sub": "REQUEST",
    "RequestId": 123,
    "Msg": {
        "method": "GET",
        "path": "/path",
        "headers": {},
        "body": []byte
    }
}
```

### 3. Response Message
```json
{
    "Sub": "RESPONSE",
    "RequestId": 123,
    "Msg": "response_data"
}
```

## Client Behavior

### Startup Process
1. Generate unique client ID
2. Register with HTTP server
3. Establish TCP connection
4. Register with gRPC server
5. Start processing messages

### Message Processing
1. Receive message from server
2. Parse message type and payload
3. Execute appropriate callback
4. Send response back to server

### Error Handling
- Connection retry mechanism
- Error response formatting
- Logging of errors and responses
- Timeout handling
- Graceful shutdown

## Implementation Guidelines

### Connection Management
- Implement reconnection logic for TCP
- Handle connection timeouts
- Monitor connection health
- Buffer messages during reconnection

### Security Considerations
- TLS support (optional)
- Request validation
- Response sanitization
- Error message sanitization

### Performance Optimization
- Message buffering
- Connection pooling
- Callback execution timeouts
- Resource cleanup
- Memory management

## Testing
- Unit tests for callbacks
- Integration tests for protocols
- Load testing guidelines
- Error scenario testing
- Connection failure recovery testing
