package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	"multichannel/cmd/callbacks"
	"multichannel/cmd/messages"
	conversion "multichannel/cmd/protos"
	"multichannel/cmd/typedefs"
	grpcclient "multichannel/grpc/client"
	"net"
	"net/http"
	"time"
)

type ClientBlock struct {
	Host             string
	HTTP             int
	TCP              int
	GRPC             int
	Paths            []string
	ClientId         string
	callbackRegistry *callbacks.CallbackRegistry
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
}

func (b *ClientBlock) TcpConnect() {
	log.Printf("Attempting to connect to TCP server with port: %d", b.TCP)
	address := fmt.Sprintf("127.0.0.1:%d", b.TCP)
	log.Printf("Using address: %s", address)
	var conn net.Conn
	var err error

	// Function to establish a connection
	connect := func() {
		for {
			log.Printf("Dialing TCP4 at address: %s", address)
			conn, err = net.Dial("tcp4", address)
			if err == nil {
				log.Printf("Successfully connected to TCP server at %s", address)
				break
			}
			log.Printf("Error connecting to TCP server: %v", err)
			time.Sleep(5 * time.Second) // Wait before retrying
		}
	}

	// Establish initial connection
	connect()
	defer conn.Close()

	// Read welcome message
	var welcome typedefs.TcpMessage
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&welcome); err != nil {
		log.Printf("Error reading welcome message: %v", err)
		return
	}
	log.Printf("Received welcome message: %s - %v", welcome.Sub, welcome.Msg)

	// Send registration message
	b.Reg(&conn)

	// Handle server responses
	for {
		var response typedefs.TcpMessage
		if err := decoder.Decode(&response); err != nil {
			if err == io.EOF {
				log.Printf("Connection closed by server")
				conn.Close()
				connect() // Reconnect
				continue
			}
			log.Printf("Error reading from server: %v", err)
			conn.Close()
			connect() // Reconnect
			continue
		}

		log.Printf("Client received message type: %s", response.Sub)
		switch response.Sub {
		case "REQUEST":
			log.Printf("Registration response: %v", response.Msg)

			result, err := b.callbackRegistry.Execute("REG", response.Msg)
			if err != nil {
				respMsg := typedefs.TcpMessage{
					Sub: "ERROR",
					Msg: err,
				}
				jsonData, err := json.Marshal(respMsg)
				if err != nil {
					log.Printf("Error marshalling JSON: %v", err)
					return
				}
				_, err = conn.Write(jsonData)
				if err != nil {
					log.Printf("Error writing to TCP server: %v", err)
					return
				}
			}
			respmsg := typedefs.TcpMessage{
				Sub: "RESPONSE",
				Msg: result,
			}
			jsonData, err := json.Marshal(respmsg)
			if err != nil {
				log.Printf("Error marshalling JSON: %v", err)
				return
			}
			_, err = conn.Write(jsonData)
			if err != nil {
				log.Printf("Error writing to TCP server: %v", err)
				return

			}
		case "TASK":
			log.Printf("Received task: %v", response.Msg)
		default:
			log.Printf("Unknown response type: %s", response.Sub)
		}
	}
}

func (b *ClientBlock) HandleStockUpdate(symbol string, price float64) {
	log.Printf("Stock Update: %s is now $%.2f", symbol, price)
}

func (b *ClientBlock) HandleWeatherUpdate(location string, temperature float64) {
	log.Printf("Weather Update: %s is now %.1f°C", location, temperature)
}

func (b *ClientBlock) HandleCryptoUpdate(coin string, price float64) {
	log.Printf("Crypto Update: %s is now $%.2f", coin, price)
}

func (b *ClientBlock) Process() {
	msg := typedefs.TcpMessage{}
	msg.Msg = json.RawMessage{}

	// Process the message based on its type
	if msg.Sub != "" {
		results, err := b.callbackRegistry.Execute(msg.Sub, msg.Msg)
		if err != nil {
			log.Printf("Error executing callback for %s: %v", msg.Sub, err)
			return
		}
		log.Printf("Callback results: %v", results)
	}
}

func (b *ClientBlock) RegisterGRPC() error {
	grpcClient, err := grpcclient.NewRegisterClient(fmt.Sprintf("%s:%d", b.Host, b.GRPC))
	if err != nil {
		return fmt.Errorf("failed to create gRPC client: %v", err)
	}

	resp, err := grpcClient.Register(b.ClientId, "test@example.com", "password123")
	if err != nil {
		return fmt.Errorf("failed to register via gRPC: %v", err)
	}

	log.Printf("GRPC Registration response: success=%v, message=%s, userId=%s",
		resp.Success, resp.Message, resp.UserId)
	return nil
}

func (b *ClientBlock) Reg(conn *net.Conn) {
	msg := typedefs.TcpMessage{
		Sub: "REG",
		Msg: map[string]interface{}{
			"client_id": b.ClientId,
			"Paths":     b.Paths,
		},
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		return
	}

	log.Printf("Sending registration message: %s", string(jsonData))
	_, err = (*conn).Write(jsonData)
	if err != nil {
		log.Printf("Error writing to TCP server: %v", err)
		return
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

func main() {
	block := &ClientBlock{
		Host: "localhost",
		HTTP: 8080,
		TCP:  8081, // Updated to match server's TCP port
		GRPC: 50051,
		Paths: []string{
			"/stocks",
			"/weather",
			"/crypto",
		},
		callbackRegistry: callbacks.NewCallbackRegistry(),
	}

	log.Printf("Initialized client block with TCP port: %d", block.TCP)

	// Register callback functions
	block.callbackRegistry.Register("STOCK", block.HandleStockUpdate)
	block.callbackRegistry.Register("WEATHER", block.HandleWeatherUpdate)
	block.callbackRegistry.Register("CRYPTO", block.HandleCryptoUpdate)

	// Register using HTTP
	block.Register()

	// Register using gRPC
	if err := block.RegisterGRPC(); err != nil {
		log.Printf("gRPC registration failed: %v", err)
	}

	// Connect via TCP
	block.TcpConnect()
	block.Process()
}
