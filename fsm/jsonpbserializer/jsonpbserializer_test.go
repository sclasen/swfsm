package jsonpbserializer

import (
	"reflect"
	"testing"
)

type NormMessage struct {
	Item1 string
}

func TestProtobufSerialization(t *testing.T) {
	ser := NewWithDefaults()

	pbd := &TestMessage{
		Item1: "hello",
		Item2: "there",
	}

	pjson, err := ser.Serialize(pbd)
	if err != nil {
		t.Fatal(err)
	}

	pbdout := &TestMessage{}
	if err := ser.Deserialize(pjson, pbdout); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(pbd, pbdout) {
		t.Fatal("Round trip proto objects differ!")
	}

	nd := &NormMessage{
		Item1: "hello",
	}

	njson, err := ser.Serialize(nd)
	if err != nil {
		t.Fatal(err)
	}

	ndout := &NormMessage{}
	if err := ser.Deserialize(njson, ndout); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(nd, ndout) {
		t.Fatal("Round trip proto objects differ!")
	}
}
