package handler

import (
    "encoding/json"
    "net/http"
    "context"
    pb "multichannel/proto"
)

type RegisterHandler struct {
    grpcClient pb.RegisterServiceClient
}

type RegisterRequest struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

func NewRegisterHandler(grpcClient pb.RegisterServiceClient) *RegisterHandler {
    return &RegisterHandler{
        grpcClient: grpcClient,
    }
}

func (h *RegisterHandler) Handle(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req RegisterRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Convert HTTP request to gRPC request
    grpcReq := &pb.RegisterRequest{
        Username: req.Username,
        Email:    req.Email,
        Password: req.Password,
    }

    // Call gRPC service
    resp, err := h.grpcClient.Register(context.Background(), grpcReq)
    if err != nil {
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // Convert gRPC response to HTTP response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "success": resp.Success,
        "message": resp.Message,
        "user_id": resp.UserId,
    })
}
