package activity

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/swf"
)

func TestHandler(t *testing.T) {
	handler := NewActivityHandler("activity", Handler)
	ret, err := handler.HandlerFunc(&swf.PollForActivityTaskOutput{}, &TestInput{Name: "testIn"})
	if ret.(*TestOutput).Name != "testInOut" {
		t.Fatal("Not testInOut")
	}

	if err != nil {
		t.Fatal("err not nil")
	}

	handler.HandlerFunc(&swf.PollForActivityTaskOutput{}, handler.ZeroInput())

	stringHandler := NewActivityHandler("activity", StringHandler)
	ret, _ = stringHandler.HandlerFunc(&swf.PollForActivityTaskOutput{}, "foo")
	if ret.(string) != "fooOut" {
		t.Fatal("string not fooOut")
	}

	nilHandler := NewActivityHandler("activity", NilHandler)
	ret, err = nilHandler.HandlerFunc(&swf.PollForActivityTaskOutput{}, &TestInput{Name: "testIn"})
	if err != nil {
		t.Fatal(ret, err)
	}

	if ret != nil {
		t.Fatal(ret)
	}

}

func Handler(task *swf.PollForActivityTaskOutput, input *TestInput) (*TestOutput, error) {
	return &TestOutput{Name: input.Name + "Out"}, nil
}

func NilHandler(task *swf.PollForActivityTaskOutput, input *TestInput) (*TestOutput, error) {
	return nil, nil
}

func StringHandler(task *swf.PollForActivityTaskOutput, input string) (string, error) {
	return input + "Out", nil
}

type TestInput struct {
	Name string
}

type TestOutput struct {
	Name string
}
