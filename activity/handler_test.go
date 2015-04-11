package activity

import (
	"testing"

	"github.com/awslabs/aws-sdk-go/gen/swf"
)

func TestHandler(t *testing.T) {
	handler := NewActivityHandler("activity", Handler)
	ret, err := handler.HandlerFunc(&swf.ActivityTask{}, &TestInput{Name: "testIn"})
	if ret.(*TestOutput).Name != "testInOut" {
		t.Fatal("Not testInOut")
	}

	if err != nil {
		t.Fatal("err not nil")
	}

	handler.HandlerFunc(&swf.ActivityTask{}, handler.ZeroInput())

}

func Handler(task *swf.ActivityTask, input *TestInput) (*TestOutput, error) {
	return &TestOutput{Name: input.Name + "Out"}, nil
}

type TestInput struct {
	Name string
}

type TestOutput struct {
	Name string
}
