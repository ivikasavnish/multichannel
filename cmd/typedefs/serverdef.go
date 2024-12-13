package typedefs

import (
	"bytes"
	"encoding/json"
	"net"
)

type ServerBlock struct {
	Host string
	HTTP int
	TCP  int
}
type TcpMessage struct {
	Sub       string `json:"sub"`
	Msg       []byte `json:"msg"`
	RequestId int32  `json:"request"`
}

type Request struct {
	RequestId int32             `json:"request_id"`
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Headers   map[string]string `json:"headers"`
	Body      []byte            `json:"body"`
}
type TcpInput struct {
	Sub       string            `json:"sub"`
	RequestID int32             `json:"request"`
	Body      []byte            `json:"body"`
	Headers   map[string]string `json:"headers"`
}

// TcpMessageReader reads TCP messages with framing support
type TcpMessageReader struct {
	conn net.Conn
}

// NewTcpMessageReader creates a new TcpMessageReader
func NewTcpMessageReader(conn net.Conn) *TcpMessageReader {
	return &TcpMessageReader{conn: conn}
}

// ReadMessage reads a TCP message with framing support
func (r *TcpMessageReader) ReadMessage() (*TcpMessage, error) {
	// Read the length of the message
	lengthBytes := make([]byte, 4)
	_, err := r.conn.Read(lengthBytes)
	if err != nil {
		return nil, err
	}
	length := int32(lengthBytes[0])<<24 | int32(lengthBytes[1])<<16 | int32(lengthBytes[2])<<8 | int32(lengthBytes[3])

	// Read the message itself
	messageBytes := make([]byte, length)
	_, err = r.conn.Read(messageBytes)
	if err != nil {
		return nil, err
	}

	// Trim the null bytes from the message
	messageBytes = bytes.Trim(messageBytes, "\x00")

	// Unmarshal the message
	var message TcpMessage
	err = json.Unmarshal(messageBytes, &message)
	if err != nil {
		return nil, err
	}

	return &message, nil
}

// TcpMessageWriter writes TCP messages with framing support
type TcpMessageWriter struct {
	conn net.Conn
}

// NewTcpMessageWriter creates a new TcpMessageWriter
func NewTcpMessageWriter(conn net.Conn) *TcpMessageWriter {
	return &TcpMessageWriter{conn: conn}
}

func (w *TcpMessageWriter) WriteMessage(message *TcpMessage) error {
	// Marshal the message
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Create a buffer to hold the length prefix and the message
	buf := make([]byte, 4+len(messageBytes))

	// Write the length prefix
	buf[0] = byte(len(messageBytes) >> 24)
	buf[1] = byte(len(messageBytes) >> 16)
	buf[2] = byte(len(messageBytes) >> 8)
	buf[3] = byte(len(messageBytes))

	// Write the message itself
	copy(buf[4:], messageBytes)

	// Write the buffer to the connection
	_, err = w.conn.Write(buf)
	if err != nil {
		return err
	}

	return nil
}
