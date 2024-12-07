package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	"multichannel/cmd/messages"
	conversion "multichannel/cmd/protos"
	"multichannel/cmd/typedefs"
	"net"
	"net/http"
	"time"
)

type ClientBlock struct {
	Host     string
	HTTP     int
	TCP      int
	Paths    []string
	ClientId string
}

var (
	bytechan = make(chan []byte, 100)
)

func init() {
	log.SetFlags(log.LstdFlags | log.Llongfile)
}

func (b *ClientBlock) Register() {
	request := messages.RegisterRequest{
		ClientId: uuid.New().String(),
		Paths:    b.Paths,
	}
	b.ClientId = request.ClientId
	url := fmt.Sprintf("http://%s:%d/register", b.Host, b.HTTP)
	log.Println("Registering with server at", url)
	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Response status:", resp.Status)
	response := messages.RegisterResponse{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return
	}
	b.TCP = response.TcpPort

}

func (b *ClientBlock) TcpConnect() {
	address := fmt.Sprintf("%s:%d", b.Host, b.TCP)
	var conn net.Conn
	var err error

	// Function to establish a connection
	connect := func() {
		for {
			conn, err = net.Dial("tcp", address)
			if err == nil {
				fmt.Println("Connected to TCP server at", address)
				break
			}
			fmt.Println("Error connecting to TCP server:", err)
			time.Sleep(5 * time.Second) // Wait before retrying
		}
	}

	// Establish initial connection
	connect()
	defer conn.Close()

	// Heartbeat ticker
	heartbeatTicker := time.NewTicker(5 * time.Second)
	defer heartbeatTicker.Stop()

	go func() {
		b.Reg(&conn)

	}()

	// Example of sending and receiving multiple messages
	for {
		select {

		default:
			// Read the response
			buffer := make([]byte, 1024)
			n, err := conn.Read(buffer)
			if err != nil {
				if err == io.EOF {
					fmt.Println("Connection closed by server")
					conn.Close()
					connect() // Reconnect
					continue
				}
				fmt.Println("Error reading from TCP server:", err)
				conn.Close()
				connect() // Reconnect
				continue
			}

			fmt.Println("Received from server:", string(buffer[:n]))

			b.TcpSend(&conn)
		}
	}
}

func (b *ClientBlock) Process() {

}

func main() {
	newClientBlock := &ClientBlock{
		Host:  "localhost",
		HTTP:  8080,
		TCP:   8081,
		Paths: []string{"/stocks", "/weather", "/crypto"},
	}

	newClientBlock.Register()
	newClientBlock.TcpConnect()
	newClientBlock.Process()

}
func (b *ClientBlock) Reg(conn *net.Conn) {
	msg := typedefs.TcpMessage{}
	msg.Sub = "REG"
	msg.Msg = messages.RegisterRequest{
		ClientId: b.ClientId,
		Paths:    b.Paths,
	}
	//msg.Msg = []byte(fmt.Sprintf("Client %s is alive at %s", b.ClientId, time.Now().String()))
	jsonData, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
	}
	//jsonMessage := append(jsonData, '\n')
	_, err = (*conn).Write(jsonData)
	if err != nil {
		fmt.Println("Error writing to TCP server:", err)
	}

}

func TcpSender(bytechan chan []byte, conn *net.Conn) {
	for {
		select {
		case msg := <-bytechan:
			_, err := (*conn).Write(msg)
			if err != nil {
				fmt.Println("Error writing to TCP server:", err)
			}
		}
	}
}
func (b *ClientBlock) TcpSend(conn *net.Conn) {
	msg := typedefs.TcpMessage{}
	msg.Sub = "RESPONSE"
	msg.Msg = conversion.HttpResponse{
		StatusCode: 200,
		Headers:    nil,
		Body:       []byte("Hello World"),
	}
	//msg.Msg = []byte(fmt.Sprintf("Client %s is alive at %s", b.ClientId, time.Now().String()))
	jsonData, err := json.Marshal(msg)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
	}
	//jsonMessage := append(jsonData, '\n')
	_, err = (*conn).Write(jsonData)
	if err != nil {
		fmt.Println("Error writing to TCP server:", err)
	}
}
