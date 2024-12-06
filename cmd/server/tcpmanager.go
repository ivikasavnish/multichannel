package main

import (
	"net"
	"sync"
)

var (
	instance *TCPManager
	once     sync.Once
)

type TCPClient struct {
	ClientId string
	Conn     *net.TCPConn
	Paths    []string
}
type TCPManager struct {
	Clients map[string]*TCPClient
}

func NewTCPManager() *TCPManager {
	once.Do(func() {
		instance = &TCPManager{
			Clients: make(map[string]*TCPClient),
		}
	})
	return instance
}
