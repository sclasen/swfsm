package activity

import (
	"testing"

	"time"
)

func TestHandler(t *testing.T) {
	handler := NewActivityHandler("activity", Handler)
	ret, err := handler.HandlerFunc(&ActivityContext{}, &TestInput{Name: "testIn"})
	if ret.(*TestOutput).Name != "testInOut" {
		t.Fatal("Not testInOut")
	}

	if err != nil {
		t.Fatal("err not nil")
	}

	handler.HandlerFunc(&ActivityContext{}, handler.ZeroInput())

	stringHandler := NewActivityHandler("activity", StringHandler)
	ret, _ = stringHandler.HandlerFunc(&ActivityContext{}, "foo")
	if ret.(string) != "fooOut" {
		t.Fatal("string not fooOut")
	}

	handled := make(chan struct{}, 1)
	longhandler := NewLongRunningActivityHandler("activity", LongHandler(handled))
	longhandler.HandlerFunc(&ActivityContext{}, &TestInput{Name: "testIn"})
	select {
	case <-handled:
		t.Log("handled")
	case <-time.After(25 * time.Millisecond):
		t.Fatal("not handled")
	}

	nilHandler := NewActivityHandler("activity", NilHandler)
	ret, err = nilHandler.HandlerFunc(&ActivityContext{}, &TestInput{Name: "testIn"})
	if err != nil {
		t.Fatal(ret, err)
	}

	if ret != nil {
		t.Fatal(ret)
	}

}

func Handler(task *ActivityContext, input *TestInput) (*TestOutput, error) {
	return &TestOutput{Name: input.Name + "Out"}, nil
}

func NilHandler(task *ActivityContext, input *TestInput) (*TestOutput, error) {
	return nil, nil
}

func LongHandler(handled chan struct{}) func(task *ActivityContext, input *TestInput) {
	return func(task *ActivityContext, input *TestInput) {
		handled <- struct{}{}
	}
}

func StringHandler(task *ActivityContext, input string) (string, error) {
	return input + "Out", nil
}

type TestInput struct {
	Name string
}

type TestOutput struct {
	Name string
}
