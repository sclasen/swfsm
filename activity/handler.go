package activity

import (
	"fmt"
	"reflect"

	"github.com/awslabs/aws-sdk-go/gen/swf"
)

type ActivityHandlerFunc func(activityTask *swf.ActivityTask, input interface{}) (interface{}, error)

type ActivityHandler struct {
	Activity    string
	HandlerFunc ActivityHandlerFunc
	Input       interface{}
}

func NewActivityHandler(activity string, handler interface{}) *ActivityHandler {
	input := inputType(handler)
	output := outputType(handler)
	typeCheck(handler, []string{"*swf.ActivityTask", input.String()}, []string{output, "error"})
	return &ActivityHandler{
		Activity:    activity,
		HandlerFunc: marshalledFunc{reflect.ValueOf(handler)}.activityHandlerFunc,
		Input:       reflect.New(input.Elem()).Elem().Interface(),
	}
}

func (a *ActivityHandler) ZeroInput() interface{} {
	return reflect.New(reflect.TypeOf(a.Input)).Interface()
}

type marshalledFunc struct {
	v reflect.Value
}

func (m marshalledFunc) activityHandlerFunc(task *swf.ActivityTask, input interface{}) (interface{}, error) {
	ret := m.v.Call([]reflect.Value{reflect.ValueOf(task), reflect.ValueOf(input)})
	if ret[0].IsNil() && ret[1].IsNil() {
		return nil, nil
	} else if ret[0].IsNil() {
		return nil, ret[1].Interface().(error)
	} else if ret[1].IsNil() {
		return ret[0].Interface(), nil
	} else {
		return ret[0].Interface(), ret[1].Interface().(error)
	}
}

func inputType(handler interface{}) reflect.Type {
	t := reflect.TypeOf(handler)
	if reflect.Func != t.Kind() {
		panic(fmt.Sprintf("kind was %v, not Func", t.Kind()))
	}
	return t.In(1)
}

func outputType(handler interface{}) string {
	t := reflect.TypeOf(handler)
	if reflect.Func != t.Kind() {
		panic(fmt.Sprintf("kind was %v, not Func", t.Kind()))
	}
	return t.Out(0).String()
}

func typeCheck(typedFunc interface{}, in []string, out []string) {
	t := reflect.TypeOf(typedFunc)
	if reflect.Func != t.Kind() {
		panic(fmt.Sprintf("kind was %v, not Func", t.Kind()))
	}
	if len(in) != t.NumIn() {
		panic(fmt.Sprintf(
			"input arity was %v, not %v",
			t.NumIn(), len(in),
		))
	}

	for i, rt := range in {
		if rt != t.In(i).String() {
			panic(fmt.Sprintf(
				"type of argument %v was %v, not %v",
				i, t.In(i), rt,
			))
		}
	}

	if len(out) != t.NumOut() {
		panic(fmt.Sprintf(
			"number of return values was %v, not %v",
			t.NumOut(), len(out),
		))
	}

	for i, rt := range out {
		if rt != t.Out(i).String() {
			panic(fmt.Sprintf(
				"type of return value %v was %v, not %v",
				i, t.Out(i), rt,
			))
		}
	}
}
