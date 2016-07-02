package activity

import (
	"fmt"
	"reflect"

	"github.com/aws/aws-sdk-go/service/swf"
)

type ActivityHandlerFunc func(activityTask *swf.PollForActivityTaskOutput, input interface{}) (interface{}, error)

type ActivityHandler struct {
	Activity    string
	HandlerFunc ActivityHandlerFunc
	Input       interface{}
}

type CoordinatedActivityHandlerStartFunc func(*swf.PollForActivityTaskOutput, interface{}) (interface{}, error)

type CoordinatedActivityHandlerTickFunc func(*swf.PollForActivityTaskOutput, interface{}) (bool, interface{}, error)

type CoordinatedActivityHandlerCancelFunc func(*swf.PollForActivityTaskOutput, interface{}) error

type CoordinatedActivityHandlerFinishFunc func(*swf.PollForActivityTaskOutput, interface{}) error

type CoordinatedActivityHandler struct {
	// Start is called when a new activity is ready to be handled.
	Start CoordinatedActivityHandlerStartFunc

	// Tick is called regularly to process a running activity.
	// Tick that returns true, nil, nil just expresses that the job is still running.
	// Tick that returns true, &SomeStruct{}, nil will express that the job is still running and also send an 'ActivityUpdated' signal back to the FSM with SomeStruct{} as the Input.
	// Tick that returns false, &SomeStruct{}, nil, expresses that the job/activity is done and send SomeStruct{} back as the result. as well as stops heartbeating.
	// Tick that returns false, nil, nil, expresses that the job is done and send no result back, as well as stops heartbeating.
	// Tick that returns false, nil, err expresses that the job/activity failed and sends back err as the reason. as well as stops heartbeating.
	Tick CoordinatedActivityHandlerTickFunc

	// Cancel is called when a running activity receives a request to cancel
	// via heartbeat update.
	Cancel CoordinatedActivityHandlerCancelFunc

	// Finish is called at the end of handling every activity.
	// It is called no matter the outcome, eg if Start fails,
	// Tick decides to stop continuing, or the activity is canceled.
	Finish CoordinatedActivityHandlerFinishFunc

	Input    interface{}
	Activity string
}

func NewActivityHandler(activity string, handler interface{}) *ActivityHandler {
	input := inputType(handler, 1)
	output := outputType(handler, 0)
	newType := input
	if input.Kind() == reflect.Ptr {
		newType = input.Elem()
	}
	typeCheck(handler, []string{"*swf.PollForActivityTaskOutput", input.String()}, []string{output, "error"})
	return &ActivityHandler{
		Activity:    activity,
		HandlerFunc: marshalledFunc{reflect.ValueOf(handler)}.activityHandlerFunc,
		Input:       reflect.New(newType).Elem().Interface(),
	}
}

func NewCoordinatedActivityHandler(activity string, start interface{}, tick interface{}, cancel interface{}, finish interface{}) *CoordinatedActivityHandler {
	input := inputType(tick, 1)
	newType := input
	if input.Kind() == reflect.Ptr {
		newType = input.Elem()
	}
	output := outputType(tick, 1)

	typeCheck(start, []string{"*swf.PollForActivityTaskOutput", input.String()}, []string{output, "error"})
	typeCheck(tick, []string{"*swf.PollForActivityTaskOutput", input.String()}, []string{"bool", output, "error"})
	typeCheck(cancel, []string{"*swf.PollForActivityTaskOutput", input.String()}, []string{"error"})
	typeCheck(finish, []string{"*swf.PollForActivityTaskOutput", input.String()}, []string{"error"})

	return &CoordinatedActivityHandler{
		Activity: activity,
		Start:    marshalledFunc{reflect.ValueOf(start)}.handleCoordinatedActivityStart,
		Tick:     marshalledFunc{reflect.ValueOf(tick)}.handleCoordinatedActivityTick,
		Cancel:   marshalledFunc{reflect.ValueOf(cancel)}.handleCoordinatedActivityCancel,
		Finish:   marshalledFunc{reflect.ValueOf(finish)}.handleCoordinatedActivityFinish,
		Input:    reflect.New(newType).Elem().Interface(),
	}

}

func (a *ActivityHandler) ZeroInput() interface{} {
	return reflect.New(reflect.TypeOf(a.Input)).Interface()
}

type marshalledFunc struct {
	v reflect.Value
}

func (m marshalledFunc) activityHandlerFunc(task *swf.PollForActivityTaskOutput, input interface{}) (interface{}, error) {
	ret := m.v.Call([]reflect.Value{reflect.ValueOf(task), reflect.ValueOf(input)})
	return outputValue(ret[0]), errorValue(ret[1])
}

func (m marshalledFunc) longRunningActivityHandlerFunc(task *swf.PollForActivityTaskOutput, input interface{}) {
	m.v.Call([]reflect.Value{reflect.ValueOf(task), reflect.ValueOf(input)})
}

func (m marshalledFunc) handleCoordinatedActivityStart(task *swf.PollForActivityTaskOutput, input interface{}) (interface{}, error) {
	ret := m.v.Call([]reflect.Value{reflect.ValueOf(task), reflect.ValueOf(input)})
	return outputValue(ret[0]), errorValue(ret[1])
}

func (m marshalledFunc) handleCoordinatedActivityTick(task *swf.PollForActivityTaskOutput, input interface{}) (bool, interface{}, error) {
	ret := m.v.Call([]reflect.Value{reflect.ValueOf(task), reflect.ValueOf(input)})
	return outputValue(ret[0]).(bool), outputValue(ret[1]), errorValue(ret[2])
}

func (m marshalledFunc) handleCoordinatedActivityCancel(task *swf.PollForActivityTaskOutput, input interface{}) error {
	ret := m.v.Call([]reflect.Value{reflect.ValueOf(task), reflect.ValueOf(input)})
	return errorValue(ret[0])
}

func (m marshalledFunc) handleCoordinatedActivityFinish(task *swf.PollForActivityTaskOutput, input interface{}) error {
	ret := m.v.Call([]reflect.Value{reflect.ValueOf(task), reflect.ValueOf(input)})
	return errorValue(ret[0])
}

func outputValue(v reflect.Value) interface{} {
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
		if v.IsNil() {
			return nil
		} else {
			return v.Interface()
		}
	default:
		return v.Interface()
	}
}

func errorValue(v reflect.Value) error {
	if v.IsNil() {
		return nil
	}
	return v.Interface().(error)
}

func inputType(handler interface{}, idx int) reflect.Type {
	t := reflect.TypeOf(handler)
	if reflect.Func != t.Kind() {
		panic(fmt.Sprintf("kind was %v, not Func", t.Kind()))
	}
	return t.In(idx)
}

func outputType(handler interface{}, idx int) string {
	t := reflect.TypeOf(handler)
	if reflect.Func != t.Kind() {
		panic(fmt.Sprintf("kind was %v, not Func", t.Kind()))
	}
	return t.Out(idx).String()
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
