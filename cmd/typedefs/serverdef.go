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
