package main

type ServerBlock struct {
	Host       string
	HTTP       int
	TCP        int
	TCPManager *TCPManager
}
