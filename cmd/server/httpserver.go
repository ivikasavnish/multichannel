package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func WildRoute(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		// Handle the request for the root path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Root path accessed"))
	} else {
		// Handle other paths
		log.Println("searching for tcp for path", r.URL.Path)
		paths := strings.Split(r.URL.Path, "/")
		log.Println(paths)
		if len(paths) < 2 {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Path not found"))
			return
		}
		path := paths[1]
		log.Println("searching for tcp for path", path)
		conn, exists := instance.InvertedMap["/"+path]
		if exists {
			log.Println("Found connection", conn)
			(*conn).Write([]byte("request received" + r.URL.Path))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Path not found"))
			return
		}
		w.WriteHeader(http.StatusOK)

	}
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func Clients(w http.ResponseWriter, r *http.Request) {
	if tcpmanager == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("TCP manager not initialized"))
		return
	}

	json.NewEncoder(w).Encode(serverblock.TCPManager.Clients)
}

func (s *ServerBlock) HttpListen() {
	log.Println("Starting HTTP server on", s.Host+":"+strconv.Itoa(s.HTTP))
	http.HandleFunc("/register", Register)
	http.HandleFunc("/healthz", HealthCheck)
	http.HandleFunc("/clients", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		log.Println(instance.Clients)

		json.NewEncoder(writer).Encode(instance.Clients)
	})
	http.HandleFunc("/", WildRoute)
	err := http.ListenAndServe(s.Host+":"+strconv.Itoa(s.HTTP), nil)
	if err != nil {
		log.Println(err)
		return
	}

}
