package activity

import (
	"fmt"
	"reflect"
)

type ActivityHandlerFunc func(ctx *ActivityContext, input interface{}) (interface{}, error)

type ActivityHandler struct {
	Activity    string
	HandlerFunc ActivityHandlerFunc
	Input       interface{}
}

type LongRunningActivityHandlerFunc func(ctx *ActivityContext, input interface{})

type LongRunningActivityHandler struct {
	Activity    string
	HandlerFunc LongRunningActivityHandlerFunc
	Input       interface{}
}

type CoordinatedActivityHandlerStartFunc func(*ActivityContext, interface{}) (interface{}, error)

type CoordinatedActivityHandlerTickFunc func(*ActivityContext, interface{}) (bool, interface{}, error)

type CoordinatedActivityHandlerCancelFunc func(*ActivityContext, interface{}) error

type CoordinatedActivityHandler struct {
	Start    CoordinatedActivityHandlerStartFunc
	Tick     CoordinatedActivityHandlerTickFunc
	Cancel   CoordinatedActivityHandlerCancelFunc
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
	typeCheck(handler, []string{"*activity.ActivityContext", input.String()}, []string{output, "error"})
	return &ActivityHandler{
		Activity:    activity,
		HandlerFunc: marshalledFunc{reflect.ValueOf(handler)}.activityHandlerFunc,
		Input:       reflect.New(newType).Elem().Interface(),
	}
}

func NewLongRunningActivityHandler(activity string, handler interface{}) *LongRunningActivityHandler {
	input := inputType(handler, 1)
	newType := input
	if input.Kind() == reflect.Ptr {
		newType = input.Elem()
	}

	typeCheck(handler, []string{"*activity.ActivityContext", input.String()}, []string{})
	return &LongRunningActivityHandler{
		Activity:    activity,
		HandlerFunc: marshalledFunc{reflect.ValueOf(handler)}.longRunningActivityHandlerFunc,
		Input:       reflect.New(newType).Elem().Interface(),
	}
}

func NewCoordinatedActivityHandler(activity string, start interface{}, tick interface{}, cancel interface{}) *CoordinatedActivityHandler {

	input := inputType(tick, 1)
	newType := input
	if input.Kind() == reflect.Ptr {
		newType = input.Elem()
	}
	output := outputType(tick, 1)

	typeCheck(start, []string{"*activity.ActivityContext", input.String()}, []string{output, "error"})
	typeCheck(tick, []string{"*activity.ActivityContext", input.String()}, []string{"bool", output, "error"})
	typeCheck(cancel, []string{"*activity.ActivityContext", input.String()}, []string{"error"})

	return &CoordinatedActivityHandler{
		Activity: activity,
		Start:    marshalledFunc{reflect.ValueOf(start)}.handleCoordinatedActivityStart,
		Tick:     marshalledFunc{reflect.ValueOf(tick)}.handleCoordinatedActivityTick,
		Cancel:   marshalledFunc{reflect.ValueOf(cancel)}.handleCoordinatedActivityCancel,
		Input:    reflect.New(newType).Elem().Interface(),
	}

}

func (a *ActivityHandler) ZeroInput() interface{} {
	return reflect.New(reflect.TypeOf(a.Input)).Interface()
}

func (a *LongRunningActivityHandler) ZeroInput() interface{} {
	return reflect.New(reflect.TypeOf(a.Input)).Interface()
}

type marshalledFunc struct {
	v reflect.Value
}

func (m marshalledFunc) activityHandlerFunc(task *ActivityContext, input interface{}) (interface{}, error) {
	ret := m.v.Call([]reflect.Value{reflect.ValueOf(task), reflect.ValueOf(input)})
	return outputValue(ret[0]), errorValue(ret[1])
}

func (m marshalledFunc) longRunningActivityHandlerFunc(task *ActivityContext, input interface{}) {
	m.v.Call([]reflect.Value{reflect.ValueOf(task), reflect.ValueOf(input)})
}

func (m marshalledFunc) handleCoordinatedActivityStart(task *ActivityContext, input interface{}) (interface{}, error) {
	ret := m.v.Call([]reflect.Value{reflect.ValueOf(task), reflect.ValueOf(input)})
	return outputValue(ret[0]), errorValue(ret[1])
}

func (m marshalledFunc) handleCoordinatedActivityTick(task *ActivityContext, input interface{}) (bool, interface{}, error) {
	ret := m.v.Call([]reflect.Value{reflect.ValueOf(task), reflect.ValueOf(input)})
	return outputValue(ret[0]).(bool), outputValue(ret[1]), errorValue(ret[2])
}

func (m marshalledFunc) handleCoordinatedActivityCancel(task *ActivityContext, input interface{}) error {
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
