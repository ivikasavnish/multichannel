package main

import (
	"encoding/json"
	"multichannel/cmd/messages"
	"net/http"
)

func Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req messages.RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp := messages.RegisterResponse{
		ClientId: req.ClientId,
		Paths:    req.Paths,
		TcpPort:  serverblock.TCP,
	}
	json.NewEncoder(w).Encode(resp)
}
