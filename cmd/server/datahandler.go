package main

import (
	"google.golang.org/protobuf/proto"
	conversion "multichannel/cmd/protos"
)

// Encode Protobuf to HTTP request
func encodeProtoHTTPRequest(data *conversion.HttpRequest) ([]byte, error) {
	// Marshal the TcpData into Protobuf
	protoData, err := proto.Marshal(data)
	if err != nil {
		return nil, err
	}

	return protoData, nil
}
