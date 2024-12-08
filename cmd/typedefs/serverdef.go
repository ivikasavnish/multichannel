package typedefs

type ServerBlock struct {
	Host string
	HTTP int
	TCP  int
}
type TcpMessage struct {
	Sub string      `json:"sub"`
	Msg interface{} `json:"msg"`
}
type TcpInput struct {
	Sub       string            `json:"sub"`
	RequestID int32             `json:"request"`
	Body      []byte            `json:"body"`
	Headers   map[string]string `json:"headers"`
}
