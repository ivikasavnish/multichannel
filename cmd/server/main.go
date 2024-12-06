package main


func main() {
	serverblock := ServerBlock{
		Host: "localhost",
		HTTP: 8080,
		TCP: 8081,
	}
	serverblock.TCPListen()
	serverblock.HTTPListen()
	select {}

}


