package protobufserializer

import (
	"encoding/base64"

	"github.com/golang/protobuf/proto"
	"github.com/juju/errors"
)

// ProtobufStateSerializer is a StateSerializer that uses base64 encoded protobufs.
type ProtobufStateSerializer struct{}

// Serialize serializes the given struct into bytes with protobuf, then base64 encodes it.  The struct passed to Serialize must satisfy proto.Message.
func (p ProtobufStateSerializer) Serialize(state interface{}) (string, error) {
	bin, err := proto.Marshal(state.(proto.Message))
	if err != nil {
		return "", errors.Trace(err)
	}
	return base64.StdEncoding.EncodeToString(bin), nil
}

// Deserialize base64 decodes the given string then unmarshalls the bytes into the struct using protobuf. The struct passed to Deserialize must satisfy proto.Message.
func (p ProtobufStateSerializer) Deserialize(serialized string, state interface{}) error {
	bin, err := base64.StdEncoding.DecodeString(serialized)
	if err != nil {
		return err
	}
	err = proto.Unmarshal(bin, state.(proto.Message))

	return errors.Trace(err)
}
