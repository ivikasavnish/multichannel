package main

import (
	"fmt"
	"io"
	"log"
	"multichannel/cmd/typedefs"
	"net"
	"reflect"
)

func (s *ServerBlock) TCPListen() {
	loc := getFileAndLine()
	address := fmt.Sprintf("127.0.0.1:%d", s.TCP)
	listener, err := net.Listen("tcp4", address)
	if err != nil {
		log.Printf("[%s] TCP server failed to start: %v", loc, err)
		return
	}
	log.Printf("[%s] Starting TCP server on %s", loc, address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("[%s] TCP connection error: %v", loc, err)
			continue
		}
		log.Printf("[%s] New client connected from %s", loc, conn.RemoteAddr())
		conn.Write([]byte("Connected to TCP server"))
		go TcpLoop(&conn)
	}
}

func TcpLoop(c *net.Conn) {
	loc := getFileAndLine()
	conn := *c
	defer conn.Close()

	log.Printf("[%s] Starting TCP loop for client %s", loc, conn.RemoteAddr())
	for {
		err := tcpMessageHandler(&conn)
		if err != nil {
			if err == io.EOF {
				log.Printf("[%s] Connection closed by client %s", loc, conn.RemoteAddr())
				break
			}
			log.Printf("[%s] Error handling message from %s: %v", loc, conn.RemoteAddr(), err)
			break
		}
	}
}

func tcpMessageHandler(conn *net.Conn) error {
	loc := getFileAndLine()
	log.Printf("[%s] Handling new message from %s", loc, (*conn).RemoteAddr())

	reader := typedefs.NewTcpMessageReader(*conn)
	writer := typedefs.NewTcpMessageWriter(*conn)
	msg, err := reader.ReadMessage()
	if err != nil {
		log.Printf("[%s] Error reading message from %s: %v", loc, (*conn).RemoteAddr(), err)
		return err
	}
	switch msg.Sub {
	case "REG":
		reg, ok := msg.Msg.(map[string]interface{})
		if !ok {
			log.Printf("[%s] Error casting message to RegisterRequest: %s", loc, reflect.TypeOf(msg.Msg).Kind().String())
			return nil
		}

		clientId, ok := reg["client_id"].(string)
		if !ok {
			log.Printf("[%s] Error getting client_id from registration", loc)
			return nil
		}

		paths, ok := reg["Paths"].([]interface{})
		if !ok {
			log.Printf("[%s] Error getting paths from registration", loc)
			return nil
		}

		log.Printf("[%s] Registering client %s with paths: %v", loc, clientId, paths)
		tcpmanager.Register(clientId, paths, conn)

		response := typedefs.TcpMessage{
			Sub: "REG_RESPONSE",
			Msg: "Registration successful",
		}
		if err := writer.WriteMessage(&response); err != nil {
			log.Printf("[%s] Error sending registration response: %v", loc, err)
			return err
		}
		log.Println("Registered client with id", clientId, "messege sent to client")

	case "HEARTBEAT":
		log.Printf("[%s] Received heartbeat from %s", loc, (*conn).RemoteAddr())
		response := typedefs.TcpMessage{
			Sub: "HEARTBEAT_RESPONSE",
			Msg: "Heartbeat acknowledged",
		}
		if err := writer.WriteMessage(&response); err != nil {
			log.Printf("[%s] Error sending heartbeat response: %v", loc, err)
			return err
		}

	default:
		log.Printf("[%s] Unknown message type: %s", loc, msg.Sub)
	}
	return nil
}
