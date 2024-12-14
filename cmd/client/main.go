package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	ollama "multichannel/aiapp"
	"multichannel/cmd/callbacks"
	"multichannel/cmd/messages"
	conversion "multichannel/cmd/protos"
	"multichannel/cmd/typedefs"
	grpcclient "multichannel/grpc/client"
	"multichannel/http/lib"
	"net"
	"net/http"
	"strings"
	"time"
)

type OllamaI struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format"`
}

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
	bytechan     = make(chan []byte, 100)
	client       = lib.NewHttpClient()
	ollamaclient = ollama.NewClient("http://192.168.1.10:11435")
)

func init() {
	log.SetFlags(log.LstdFlags | log.Llongfile)

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

func (b *ClientBlock) Reg(conn *net.Conn, writer *typedefs.TcpMessageWriter) {
	payload, err := json.Marshal(map[string]interface{}{
		"client_id": b.ClientId,
		"Paths":     b.Paths,
	})
	if err != nil {
		log.Printf("Error marshalling JSON: %v", err)
		return
	}
	msg := typedefs.TcpMessage{
		Sub: "REG",
		Msg: payload,
	}

	err = writer.WriteMessage(&msg)
	if err != nil {
		log.Printf("Error sending registration message: %v", err)
		return
	}
	log.Printf("Registration message sent")
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
	}.ToBytes()
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
			"/ollama",
		},
		callbackRegistry: callbacks.NewCallbackRegistry(),
	}

	log.Printf("Initialized client block with TCP port: %d", block.TCP)

	// Register callback functions
	block.callbackRegistry.Register("/stocks", stocksCallback)
	block.callbackRegistry.Register("/weather", weatherCallback)
	block.callbackRegistry.Register("/crypto", cryptoCallback)
	block.callbackRegistry.Register("/ollama", ollamaCallback)

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
	// create a reder and writer
	reader := typedefs.NewTcpMessageReader(conn)
	writer := typedefs.NewTcpMessageWriter(conn)

	// Read welcome message
	welcome, err := reader.ReadMessage()
	if err != nil {
		log.Printf("Error reading welcome message: %v", err)
		return
	}
	log.Printf("Received welcome message: %s - %v", welcome.Sub, string(welcome.Msg))

	// Send registration message
	b.Reg(&conn, writer)

	// Handle server responses
	for {
		response, err := reader.ReadMessage()
		if response == nil {
			//log.Printf("Received nil message from server")
			continue
		}
		log.Printf("Client received message type: %s", response)
		if err != nil {
			if err == io.EOF {
				log.Printf("Connection closed by server")
				//conn.Close()
				//connect() // Reconnect
				continue
			}
			log.Printf("Error reading from server: %v", err)

		}

		//log.Printf("Client received message type: %s", response.Sub)
		switch response.Sub {
		case "REQUEST":
			log.Printf("HTTP response: %v", string(response.Msg))
			var request map[string]interface{}
			err := json.Unmarshal(response.Msg, &request)
			if err != nil {
				log.Printf("Error unmarshalling request: %v", err)
				continue
			}

			requestid := int32(request["request_id"].(float64))

			result, err := b.callbackRegistry.Execute(response.Sub, response.Msg)

			if err != nil || result == nil {
				respMsg := typedefs.TcpMessage{
					Sub:       "ERROR",
					Msg:       []byte(fmt.Sprintf("{\"error\": \"%v\"}", err)),
					RequestId: requestid,
				}
				err := writer.WriteMessage(&respMsg)
				if err != nil {
					log.Printf("Error writing to TCP server: %v", err)

				}
				log.Printf("Response sent")
				continue
			}
			respmsg := typedefs.TcpMessage{
				Sub:       "RESPONSE",
				Msg:       result,
				RequestId: requestid,
			}
			err = writer.WriteMessage(&respmsg)
			if err != nil {
				log.Printf("Error writing to TCP server: %v", err)
			}
			log.Printf("Response sent")
		case "TASK":
			log.Printf("Received task: %v", response.Msg)
		case "REG_RESPONSE":
			log.Printf("Received registration response: %v", response.Msg)
		default:
			log.Printf("Unknown response type: %s", response.Sub)
		}
	}
}

// Callback for /stocks
func stocksCallback(req typedefs.Request) interface{} {
	// Simulate a database query to retrieve stock data
	stockData := []map[string]interface{}{
		{"symbol": "AAPL", "price": 150.0},
		{"symbol": "GOOG", "price": 2500.0},
		{"symbol": "AMZN", "price": 3000.0},
	}

	return stockData
}

// Callback for /weather
func weatherCallback(req typedefs.Request) interface{} {
	// Simulate a weather API call to retrieve current weather conditions
	weatherData := map[string]interface{}{
		"temperature": 75.0,
		"humidity":    60.0,
		"conditions":  "Sunny",
	}

	return weatherData
}

// Callback for /crypto
func cryptoCallback(req typedefs.Request) interface{} {
	// Simulate a cryptocurrency API call to retrieve current prices
	cryptoData := []map[string]interface{}{
		{"symbol": "BTC", "price": 50000.0},
		{"symbol": "ETH", "price": 4000.0},
		{"symbol": "LTC", "price": 200.0},
	}

	return cryptoData
}

func ollamaCallback(req typedefs.Request) interface{} {
	// Simulate a cryptocurrency API call to retrieve current prices

	url := "http://192.168.1.10:11435/api/generate"
	method := "POST"

	payload := strings.NewReader(` {
    "model": "mistral:latest",
    "prompt": "best 10 country to live in ",
    "stream":false
}
`)

	client := &http.Client{}
	reqllama, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return err
	}
	reqllama.Header.Add("Content-Type", "application/json")

	res, err := client.Do(reqllama)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Println(string(body))
	return string(body)
}
