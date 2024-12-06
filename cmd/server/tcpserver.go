package main

import (
	"fmt"
	"net"
)

func (s ServerBlock) TCPListen() {
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
		go s.handleConnection(&conn)
	}

}

func (s ServerBlock) handleConnection(conn *net.Conn) {
	fmt.Println("Handling new connection")


	buf := make([]byte, 1024)
	for {
		tcp_message_handler(conn, buf)
	}
}

func tcp_message_handler(conn *net.Conn, buf []byte) {
	fmt.Println("Handling new connection")
	n, err := (*conn).Read(buf)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(buf[:n]))
}
