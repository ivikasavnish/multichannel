package client

import (
	"context"
	"google.golang.org/grpc"
	"log"
	pb "multichannel/proto"
)

type RegisterClient struct {
	client pb.RegisterServiceClient
	conn   *grpc.ClientConn
}

func NewRegisterClient(address string) (*RegisterClient, error) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}

	client := pb.NewRegisterServiceClient(conn)
	return &RegisterClient{
		client: client,
		conn:   conn,
	}, nil
}

// Close closes the gRPC connection
func (c *RegisterClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *RegisterClient) Register(username, email, password string) (*pb.RegisterResponse, error) {
	req := &pb.RegisterRequest{
		Username: username,
		Email:    email,
		Password: password,
	}

	ctx := context.Background()
	resp, err := c.client.Register(ctx, req)
	if err != nil {
		log.Printf("Error during registration: %v", err)
		return nil, err
	}

	return resp, nil
}

func (c *RegisterClient) RegisterPath(clientID string, paths []string) (*pb.RegisterPathResponse, error) {
	req := &pb.RegisterPathRequest{
		ClientId: clientID,
		Paths:    paths,
	}

	ctx := context.Background()
	resp, err := c.client.RegisterPath(ctx, req)
	if err != nil {
		log.Printf("Error during path registration: %v", err)
		return nil, err
	}

	return resp, nil
}
