package main

import (
	"log"
	"net"
	"sync"
)

var (
	instance *TCPManager
	once     sync.Once
)

type TCPClient struct {
	ClientId    string
	Conn        *net.Conn
	Paths       []string
	InvertedMap map[string]*net.Conn
}
type TCPManager struct {
	Clients     map[string]*TCPClient
	InvertedMap map[string]*net.Conn
}

func (m *TCPManager) Register(id string, paths []interface{}, conn *net.Conn) {
	log.Println(paths)
	pathslice := make([]string, len(paths))
	for i, path := range paths {
		pathslice[i] = path.(string)
	}
	instance.Clients[id] = &TCPClient{
		ClientId: id,
		Conn:     conn,
		Paths:    pathslice,
	}
	log.Println("Registered client with id", id)
	for _, path := range pathslice {
		instance.InvertedMap[path] = conn
	}
	log.Println(*conn)
	serverblock.TCPManager = m

}

func NewTCPManager() *TCPManager {
	once.Do(func() {
		instance = &TCPManager{
			Clients:     make(map[string]*TCPClient),
			InvertedMap: make(map[string]*net.Conn),
		}
	})
	return instance
}
