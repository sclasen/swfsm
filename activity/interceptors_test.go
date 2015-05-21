package activity

import (
	"errors"
	"testing"

	"github.com/awslabs/aws-sdk-go/service/swf"
	. "github.com/sclasen/swfsm/sugar"
)

func TestInterceptors(t *testing.T) {
	calledFail := false
	calledBefore := false
	calledComplete := false
	calledCanceled := false

	task := &swf.PollForActivityTaskOutput{
		ActivityType:      &swf.ActivityType{Name: S("test"), Version: S("test")},
		ActivityID:        S("ID"),
		WorkflowExecution: &swf.WorkflowExecution{WorkflowID: S("ID"), RunID: S("run")},
	}

	interceptor := &FuncInterceptor{
		BeforeTaskFn: func(decision *swf.PollForActivityTaskOutput) {
			calledBefore = true
		},
		AfterTaskCompleteFn: func(decision *swf.PollForActivityTaskOutput, result interface{}) {
			calledComplete = true
		},
		AfterTaskFailedFn: func(decision *swf.PollForActivityTaskOutput, err error) {
			calledFail = true
		},
		AfterTaskCanceledFn: func(decision *swf.PollForActivityTaskOutput, details string) {
			calledCanceled = true
		},
	}

	worker := &ActivityWorker{
		ActivityInterceptor: interceptor,
		SWF:                 &MockSWF{},
	}

	handler := &ActivityHandler{
		Activity: "test",
		HandlerFunc: func(activityTask *swf.PollForActivityTaskOutput, input interface{}) (interface{}, error) {
			return nil, nil
		},
	}

	worker.AddHandler(handler)

	worker.handleActivityTask(task)

	if !calledBefore {
		t.Fatal("no before")
	}

	if !calledComplete {
		t.Fatal("no after ok")
	}

	task.ActivityType.Name = S("nottest")

	calledFail = false
	calledBefore = false
	calledComplete = false

	worker.handleActivityTask(task)

	if !calledBefore {
		t.Fatal("no before")
	}

	if !calledFail {
		t.Fatal("no after fail")
	}

}

func TestFailedInterceptor(t *testing.T) {
	var (
		calledFail     = false
		calledBefore   = false
		calledComplete = false
		calledCanceled = false
		failMessage    string
	)
	task := &swf.PollForActivityTaskOutput{
		ActivityType:      &swf.ActivityType{Name: S("test"), Version: S("test")},
		ActivityID:        S("ID"),
		WorkflowExecution: &swf.WorkflowExecution{WorkflowID: S("ID"), RunID: S("run")},
	}
	interceptor := &FuncInterceptor{
		BeforeTaskFn: func(decision *swf.PollForActivityTaskOutput) {
			calledBefore = true
		},
		AfterTaskCompleteFn: func(decision *swf.PollForActivityTaskOutput, result interface{}) {
			calledComplete = true
		},
		AfterTaskFailedFn: func(decision *swf.PollForActivityTaskOutput, err error) {
			calledFail = true
			failMessage = err.Error()
		},
		AfterTaskCanceledFn: func(decision *swf.PollForActivityTaskOutput, details string) {
			calledCanceled = true
		},
	}
	worker := &ActivityWorker{
		ActivityInterceptor: interceptor,
		SWF:                 &MockSWF{},
	}
	handler := &ActivityHandler{
		Activity: "test",
		HandlerFunc: func(activityTask *swf.PollForActivityTaskOutput, input interface{}) (interface{}, error) {
			return nil, errors.New("fail")
		},
	}

	worker.AddHandler(handler)
	worker.handleActivityTask(task)
	if !calledBefore {
		t.Fatal("no before")
	}
	if !calledFail {
		t.Fatal("no after fail")
	}
	if failMessage != "fail" {
		t.Fatal("wong error message")
	}

}

func TestCanceledInterceptor(t *testing.T) {
	var (
		calledFail     = false
		calledBefore   = false
		calledComplete = false
		calledCanceled = false
		details        string
	)
	task := &swf.PollForActivityTaskOutput{
		ActivityType:      &swf.ActivityType{Name: S("test"), Version: S("test")},
		ActivityID:        S("ID"),
		WorkflowExecution: &swf.WorkflowExecution{WorkflowID: S("ID"), RunID: S("run")},
	}
	interceptor := &FuncInterceptor{
		BeforeTaskFn: func(decision *swf.PollForActivityTaskOutput) {
			calledBefore = true
		},
		AfterTaskCompleteFn: func(decision *swf.PollForActivityTaskOutput, result interface{}) {
			calledComplete = true
		},
		AfterTaskFailedFn: func(decision *swf.PollForActivityTaskOutput, err error) {
			calledFail = true
		},
		AfterTaskCanceledFn: func(decision *swf.PollForActivityTaskOutput, det string) {
			calledCanceled = true
			details = det
		},
	}
	worker := &ActivityWorker{
		ActivityInterceptor: interceptor,
		SWF:                 &MockSWF{},
	}
	handler := &ActivityHandler{
		Activity: "test",
		HandlerFunc: func(activityTask *swf.PollForActivityTaskOutput, input interface{}) (interface{}, error) {
			return nil, ActivityTaskCanceledError{details: "details"}
		},
	}

	worker.AddHandler(handler)
	worker.handleActivityTask(task)
	if !calledBefore {
		t.Fatal("no before")
	}
	if !calledCanceled {
		t.Fatal("no after canceled")
	}
	if details != "details" {
		t.Fatalf("wong task canceled details. Got: %q", details)
	}

}
