package fsm

import (
	"testing"

	"github.com/awslabs/aws-sdk-go/gen/swf"
	. "github.com/sclasen/swfsm/sugar"
)

func TestInterceptors(t *testing.T) {
	calledAfter := false
	calledBefore := false
	calledBeforeCtx := false

	interceptor := &FuncInterceptor{
		BeforeTaskFn: func(decision *swf.DecisionTask) {
			calledBefore = true
		},
		BeforeDecisionFn: func(decision *swf.DecisionTask, ctx *FSMContext, outcome Outcome) {
			calledBeforeCtx = true
		},
		AfterDecisionFn: func(decision *swf.DecisionTask, ctx *FSMContext, outcome Outcome) {
			calledAfter = true
		},
	}

	fsm := &FSM{
		Name:                "test-fsm",
		DataType:            TestData{},
		DecisionInterceptor: interceptor,
		Serializer:          JSONStateSerializer{},
		systemSerializer:    JSONStateSerializer{},
	}

	fsm.AddInitialState(&FSMState{Name: "initial", Decider: func(ctx *FSMContext, e swf.HistoryEvent, d interface{}) Outcome {
		return Outcome{State: "initial", Data: d, Decisions: []swf.Decision{}}
	}})

	decisionTask := new(swf.DecisionTask)
	decisionTask.WorkflowExecution = new(swf.WorkflowExecution)
	decisionTask.WorkflowType = &swf.WorkflowType{Name: S("test"), Version: S("1")}
	decisionTask.WorkflowExecution.RunID = S("run")
	decisionTask.WorkflowExecution.WorkflowID = S("wf")
	decisionTask.PreviousStartedEventID = I(5)
	decisionTask.StartedEventID = I(15)
	decisionTask.Events = []swf.HistoryEvent{
		{
			EventID:   I(10),
			EventType: S("WorkflowExecutionStarted"),
			WorkflowExecutionStartedEventAttributes: &swf.WorkflowExecutionStartedEventAttributes{
				Input: S(fsm.Serialize(new(TestData))),
			},
		},
	}

	fsm.Tick(decisionTask)

	if calledBefore == false {
		t.Fatalf("before not called")
	}

	if calledBeforeCtx == false {
		t.Fatalf("before context not called")
	}

	if calledBefore == false {
		t.Fatalf("after not called")
	}
}
