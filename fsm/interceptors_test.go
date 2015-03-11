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
		BeforeDecisionFn: func(decision *swf.DecisionTask, ctx *FSMContext, outcome *Outcome) {
			outcome.Decisions = append(outcome.Decisions, ctx.CompleteWorkflowDecision(&TestData{}))
			outcome.Decisions = append(outcome.Decisions, ctx.CompleteWorkflowDecision(&TestData{}))
			calledBeforeCtx = true
		},
		AfterDecisionFn: func(decision *swf.DecisionTask, ctx *FSMContext, outcome *Outcome) {
			if countCompletes(outcome.Decisions) != 2 {
				t.Fatal("not 2 completes in after")
			}
			outcome.Decisions = dedupeCompletes(outcome.Decisions)
			if countCompletes(outcome.Decisions) != 1 {
				t.Fatal("not 1 completes in after dedupe")
			}
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

	_, ds, _, _ := fsm.Tick(decisionTask)

	if calledBefore == false {
		t.Fatalf("before not called")
	}

	if calledBeforeCtx == false {
		t.Fatalf("before context not called")
	}

	if calledAfter == false {
		t.Fatalf("after not called")
	}

	if countCompletes(ds) != 1 {
		t.Fatalf("Deduping completes failed %v", ds)
	}
}

func dedupeCompletes(in []swf.Decision) []swf.Decision {
	out := []swf.Decision{}
	complete := false
	for i := len(in) - 1; i >= 0; i-- {
		d := in[i]
		if *d.DecisionType == swf.DecisionTypeCompleteWorkflowExecution {
			if !complete {
				complete = true
				out = append([]swf.Decision{d}, out...)
			}
		} else {
			out = append([]swf.Decision{d}, out...)
		}
	}
	println(len(out))
	return out
}

func countCompletes(in []swf.Decision) int {
	count := 0
	for _, d := range in {
		if *d.DecisionType == swf.DecisionTypeCompleteWorkflowExecution {
			count++
		}
	}
	return count
}
