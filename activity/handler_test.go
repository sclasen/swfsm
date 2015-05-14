package activity

import (
	"testing"

	"time"

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

	handled := make(chan struct{}, 1)
	longhandler := NewLongRunningActivityHandler("activity", LongHandler(handled))
	longhandler.HandlerFunc(&swf.ActivityTask{}, &TestInput{Name: "testIn"})
	select {
	case <-handled:
		t.Log("handled")
	case <-time.After(25 * time.Millisecond):
		t.Fatal("not handled")
	}

	nilHandler := NewActivityHandler("activity", NilHandler)
	ret, err = nilHandler.HandlerFunc(&swf.ActivityTask{}, &TestInput{Name: "testIn"})
	if err != nil {
		t.Fatal(ret, err)
	}

	if ret != nil {
		t.Fatal(ret)
	}

}

func Handler(task *swf.ActivityTask, input *TestInput) (*TestOutput, error) {
	return &TestOutput{Name: input.Name + "Out"}, nil
}

func NilHandler(task *swf.ActivityTask, input *TestInput) (*TestOutput, error) {
	return nil, nil
}

func LongHandler(handled chan struct{}) func(task *swf.ActivityTask, input *TestInput) {
	return func(task *swf.ActivityTask, input *TestInput) {
		handled <- struct{}{}
	}
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
