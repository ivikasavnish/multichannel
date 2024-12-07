package main

var (
	tcpmanager  = NewTCPManager()
	serverblock = &ServerBlock{
		Host:       "localhost",
		HTTP:       8080,
		TCP:        8081,
		TCPManager: tcpmanager,
	}
)

func main() {
	go serverblock.HttpListen()
	go serverblock.TCPListen()

	select {}

}
