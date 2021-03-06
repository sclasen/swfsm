// Package jsonpbserializer implements a fsm.StateSerializer that uses jsonpb as
// the underlying JSON serializer, rather than stdlibs. This is used for
// providing consistent serialization of types generated by the protobuf
// compiler. If a type that is not a proto.Message is used, it will fall through
// to an alternate serializer.
package jsonpbserializer

import (
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	swfsm "github.com/sclasen/swfsm/fsm"
)

// JSONPBStateSerializer is a StateSerializer that uses protobuf json
// serialization.
type JSONPBStateSerializer struct {
	m    *jsonpb.Marshaler
	u    *jsonpb.Unmarshaler
	fbss swfsm.StateSerializer
}

// New creates a new JSONPBStateSerializer with the passed in marshaler,
// unmarshaler and fallback Serializer for non proto.Message types
func New(marshaler *jsonpb.Marshaler, unmarshaler *jsonpb.Unmarshaler, fallback swfsm.StateSerializer) *JSONPBStateSerializer {
	return &JSONPBStateSerializer{
		m:    marshaler,
		u:    unmarshaler,
		fbss: fallback,
	}
}

// NewWithDefaults creates a new JSONPBStateSerializer with a default
// jsonpb.Marshaler and jsonpb.Unmarshaler. It uses a swfsm.JSONStateSerializer
// as the fallback serializer for non-proto.Message types
func NewWithDefaults() *JSONPBStateSerializer {
	return New(&jsonpb.Marshaler{}, &jsonpb.Unmarshaler{}, &swfsm.JSONStateSerializer{})
}

// Serialize serializes the given state to a json string. If it's a
// proto.Message it will use the jsonpb Marshaler, otherwise it will use the
// fallback
func (j JSONPBStateSerializer) Serialize(state interface{}) (string, error) {
	pm, ok := state.(proto.Message)
	if ok {
		return j.m.MarshalToString(pm)
	}
	return j.fbss.Serialize(state)
}

// Deserialize unmarshalls the given (json) string into the given state. If it's
// a proto.Message it will use the jsonpb Marshaler, otherwise it will use the
// fallback
func (j JSONPBStateSerializer) Deserialize(serialized string, state interface{}) error {
	pm, ok := state.(proto.Message)
	if ok {
		return j.u.Unmarshal(strings.NewReader(serialized), pm)
	}
	return j.fbss.Deserialize(serialized, state)
}
