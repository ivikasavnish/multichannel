package messages

type RegisterRequest struct {
	ClientId string `json:"client_id"`
	Paths    []string
}

type RegisterResponse struct {
	ClientId string
	Paths    []string
	TcpPort  int
}
