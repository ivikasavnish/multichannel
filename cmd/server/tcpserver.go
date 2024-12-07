package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"multichannel/cmd/typedefs"
	"net"
	"reflect"
)

func (s *ServerBlock) TCPListen() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.TCP))
	if err != nil {
		fmt.Println(err)
		return
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		conn.Write([]byte("Connected to TCP server"))
		go TcpLoop(&conn) // Use a goroutine to handle each connection
	}
}

func TcpLoop(c *net.Conn) {
	conn := *c
	defer conn.Close() // Ensure the connection is closed when the function exits
	for {
		err := tcpMessageHandler(&conn)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection closed by client")
				break
			}
			fmt.Println("Error handling message:", err)
			break
		}
	}
}

func tcpMessageHandler(conn *net.Conn) error {
	fmt.Println("Handling new message")
	var fullMessage []byte
	buf := make([]byte, 1024)

	for {
		n, err := (*conn).Read(buf)
		if err != nil {
			return err
		}
		fullMessage = append(fullMessage, buf[:n]...)
		if n < len(buf) {
			// Assuming the message is complete if fewer bytes than the buffer size are read
			break
		}
	}

	msg := typedefs.TcpMessage{}
	log.Println(string(fullMessage))
	err := json.Unmarshal(fullMessage, &msg)
	if err != nil {
		log.Println("Error unmarshalling message:", err)
		return err
	}

	switch msg.Sub {
	case "REG":
		fmt.Println("Received message for sub1:", msg.Msg)
		fmt.Fprintf(*conn, "Received message for sub1: %s", msg.Msg)

		reg, ok := msg.Msg.(map[string]interface{})
		if !ok {
			fmt.Println("Error casting message to RegisterRequest", reflect.TypeOf(msg.Msg).Kind().String())

			return nil
		}
		log.Printf("Received registration request: %+v", reg)
		clientId, ok := reg["client_id"].(string)
		paths, ok := reg["Paths"].([]interface{})
		log.Println(paths, reflect.TypeOf(paths).Kind().String())
		tcpmanager.Register(clientId, paths, conn)
		//serverblock.TCPManager.Register(clientId, paths, conn)

	case "HTTP":
		fmt.Println("Received message for sub2:", msg.Msg)
		fmt.Fprintf(*conn, "Received message for sub2: %s", msg.Msg)
	case "HEARTBEAT":
		fmt.Println("Received heartbeat:", msg.Msg)
		fmt.Fprintf(*conn, "Received heartbeat: %s", msg.Msg)
	case "REQ":
		fmt.Println("Received message for sub3:", msg.Msg)
		fmt.Fprintf(*conn, "Received message for sub3: %s", msg.Msg)
	case "RESPONSE":
		fmt.Println("Received message for sub4:", msg.Msg)
		fmt.Fprintf(*conn, "Received message for sub4: %s", msg.Msg)
	default:
		fmt.Println("Received unknown subscription:", msg.Sub)
	}

	return err
}
