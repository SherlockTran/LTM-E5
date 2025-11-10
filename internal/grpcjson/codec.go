package grpcjson

import (
	"encoding/json"

	"google.golang.org/grpc/encoding"
)

const Name = "json"

func init() {
	encoding.RegisterCodec(JSONCodec{})
}

type JSONCodec struct{}

func (JSONCodec) Name() string { return Name }

func (JSONCodec) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (JSONCodec) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}


