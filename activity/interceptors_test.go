package activity

import (
	"testing"

	"github.com/awslabs/aws-sdk-go/gen/swf"
	. "github.com/sclasen/swfsm/sugar"
)

func TestInterceptors(t *testing.T) {
	calledFail := false
	calledBefore := false
	calledComplete := false

	task := &swf.ActivityTask{
		ActivityType:      &swf.ActivityType{Name: S("test"), Version: S("test")},
		ActivityID:        S("ID"),
		WorkflowExecution: &swf.WorkflowExecution{WorkflowID: S("ID"), RunID: S("run")},
	}

	interceptor := &FuncInterceptor{
		BeforeTaskFn: func(decision *swf.ActivityTask) {
			calledBefore = true
		},
		AfterTaskCompleteFn: func(decision *swf.ActivityTask, result interface{}) {
			calledComplete = true
		},
		AfterTaskFailedFn: func(decision *swf.ActivityTask, err error) {
			calledFail = true
		},
	}

	worker := &ActivityWorker{
		ActivityInterceptor: interceptor,
		SWF:                 MockSWF{},
	}

	handler := &ActivityHandler{
		Activity: "test",
		HandlerFunc: func(activityTask *swf.ActivityTask, input interface{}) (interface{}, error) {
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
