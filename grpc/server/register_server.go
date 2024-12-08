package server

import (
	"context"
	"log"
	pb "multichannel/proto"
)

type RegisterServer struct {
	pb.UnimplementedRegisterServiceServer
	registeredPaths map[string][]string // clientID -> paths
}

func NewRegisterServer() *RegisterServer {
	return &RegisterServer{
		registeredPaths: make(map[string][]string),
	}
}

func (s *RegisterServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// TODO: Implement actual registration logic
	return &pb.RegisterResponse{
		Success: true,
		Message: "Registration successful",
		UserId:  "user123", // Generate real user ID in production
	}, nil
}

func (s *RegisterServer) RegisterPath(ctx context.Context, req *pb.RegisterPathRequest) (*pb.RegisterPathResponse, error) {
	if req.ClientId == "" {
		return &pb.RegisterPathResponse{
			Success: false,
			Message: "Client ID is required",
		}, nil
	}

	if len(req.Paths) == 0 {
		return &pb.RegisterPathResponse{
			Success: false,
			Message: "At least one path is required",
		}, nil
	}

	// Store the paths for this client
	s.registeredPaths[req.ClientId] = req.Paths
	log.Printf("Registered paths for client %s: %v", req.ClientId, req.Paths)

	return &pb.RegisterPathResponse{
		Success:         true,
		Message:        "Paths registered successfully",
		RegisteredPaths: req.Paths,
	}, nil
}
