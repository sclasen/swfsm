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

	stringHandler := NewActivityHandler("activity", StringHandler)
	ret, _ = stringHandler.HandlerFunc(&swf.ActivityTask{}, "foo")
	if ret.(string) != "fooOut" {
		t.Fatal("string not fooOut")
	}

}

func Handler(task *swf.ActivityTask, input *TestInput) (*TestOutput, error) {
	return &TestOutput{Name: input.Name + "Out"}, nil
}

func StringHandler(task *swf.ActivityTask, input string) (string, error) {
	return input + "Out", nil
}

type TestInput struct {
	Name string
}

type TestOutput struct {
	Name string
}
