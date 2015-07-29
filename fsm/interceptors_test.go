package fsm

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/swf"
	. "github.com/sclasen/swfsm/sugar"
)

func TestInterceptors(t *testing.T) {
	calledAfter := false
	calledBefore := false
	calledBeforeCtx := false

	interceptor := &FuncInterceptor{
		BeforeTaskFn: func(decision *swf.PollForDecisionTaskOutput) {
			calledBefore = true
		},
		BeforeDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome) {
			outcome.Decisions = append(outcome.Decisions, ctx.CompleteWorkflowDecision(&TestData{}))
			outcome.Decisions = append(outcome.Decisions, ctx.CompleteWorkflowDecision(&TestData{}))
			calledBeforeCtx = true
		},
		AfterDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome) {
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

	fsm.AddInitialState(&FSMState{Name: "initial", Decider: func(ctx *FSMContext, e *swf.HistoryEvent, d interface{}) Outcome {
		return Outcome{State: "initial", Data: d, Decisions: []*swf.Decision{}}
	}})

	decisionTask := new(swf.PollForDecisionTaskOutput)
	decisionTask.WorkflowExecution = new(swf.WorkflowExecution)
	decisionTask.WorkflowType = &swf.WorkflowType{Name: S("test"), Version: S("1")}
	decisionTask.WorkflowExecution.RunID = S("run")
	decisionTask.WorkflowExecution.WorkflowID = S("wf")
	decisionTask.PreviousStartedEventID = I(5)
	decisionTask.StartedEventID = I(15)
	decisionTask.Events = []*swf.HistoryEvent{
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

func dedupeCompletes(in []*swf.Decision) []*swf.Decision {
	out := []*swf.Decision{}
	complete := false
	for i := len(in) - 1; i >= 0; i-- {
		d := in[i]
		if *d.DecisionType == swf.DecisionTypeCompleteWorkflowExecution {
			if !complete {
				complete = true
				out = append([]*swf.Decision{d}, out...)
			}
		} else {
			out = append([]*swf.Decision{d}, out...)
		}
	}
	println(len(out))
	return out
}

func countCompletes(in []*swf.Decision) int {
	count := 0
	for _, d := range in {
		if *d.DecisionType == swf.DecisionTypeCompleteWorkflowExecution {
			count++
		}
	}
	return count
}

func TestComposedInterceptor(t *testing.T) {
	calledFirst := false
	calledThird := false

	c := NewComposedDecisionInterceptor(
		&FuncInterceptor{
			BeforeTaskFn: func(decision *swf.PollForDecisionTaskOutput) {
				calledFirst = true
			},
		},
		nil, // shouldn't blow up on nil second,
		&FuncInterceptor{
			BeforeTaskFn: func(decision *swf.PollForDecisionTaskOutput) {
				calledThird = true
			},
		},
	)

	c.BeforeTask(nil)

	if !calledFirst {
		t.Fatalf("first not called")
	}

	if !calledThird {
		t.Fatalf("third not called")
	}

	c.AfterDecision(nil, nil, nil) // shouldn't blow up on non-implemented methods
}

func TestManagedContinuationsInterceptor(t *testing.T) {
	interceptor := ManagedContinuations(3, 1000, 10)

	//test that interceptor starts the contiuation age timer on start
	start := &swf.PollForDecisionTaskOutput{
		Events: []*swf.HistoryEvent{
			{
				EventID:   L(1),
				EventType: S(swf.EventTypeWorkflowExecutionStarted),
			},
		},
		PreviousStartedEventID: L(0),
	}

	startOutcome := &Outcome{
		State:     "state",
		Data:      "data",
		Decisions: []*swf.Decision{},
	}

	interceptor.AfterDecision(start, interceptorTestContext(), startOutcome)

	if len(startOutcome.Decisions) != 1 || *startOutcome.Decisions[0].DecisionType != swf.DecisionTypeStartTimer {
		t.Fatal(startOutcome.Decisions)
	}

	//test that the interceptor starts the retry timer if it is unable to continue
	cont := &swf.PollForDecisionTaskOutput{
		Events: []*swf.HistoryEvent{
			{
				EventID:   L(2),
				EventType: S(swf.EventTypeTimerFired),
				TimerFiredEventAttributes: &swf.TimerFiredEventAttributes{
					TimerID: S(ContinueTimer),
				},
			},
		},
		PreviousStartedEventID: L(1),
	}

	contOutcome := &Outcome{
		State:     "state",
		Data:      "data",
		Decisions: []*swf.Decision{},
	}

	ctx := interceptorTestContext()
	ctx.eventCorrelator.checkInit()
	ctx.eventCorrelator.Activities["1"] = &ActivityInfo{}

	interceptor.AfterDecision(cont, ctx, contOutcome)

	//assert the ContinueTimer was restarted
	if len(contOutcome.Decisions) != 1 || *contOutcome.Decisions[0].DecisionType != swf.DecisionTypeStartTimer {
		t.Fatal(contOutcome.Decisions)
	}

	delete(ctx.eventCorrelator.Activities, "1")
	contOutcome.Decisions = []*swf.Decision{}

	interceptor.AfterDecision(cont, ctx, contOutcome)

	//assert that the workflow was continued
	if len(contOutcome.Decisions) != 1 || *contOutcome.Decisions[0].DecisionType != swf.DecisionTypeContinueAsNewWorkflowExecution {
		t.Fatal(contOutcome.Decisions)
	}

	t.Log(contOutcome.Decisions[0])

	histCont := &swf.PollForDecisionTaskOutput{
		Events: []*swf.HistoryEvent{
			{
				EventID:   L(10),
				EventType: S(swf.EventTypeExternalWorkflowExecutionSignaled), //n
			},
		},
		PreviousStartedEventID: L(7),
	}

	histContOutcome := &Outcome{
		State:     "state",
		Data:      "data",
		Decisions: []*swf.Decision{},
	}

	ctx = interceptorTestContext()
	ctx.eventCorrelator.checkInit()
	ctx.eventCorrelator.Activities["1"] = &ActivityInfo{}

	interceptor.AfterDecision(histCont, ctx, histContOutcome)

	//assert the ContinueTimer was restarted
	if len(histContOutcome.Decisions) != 1 || *histContOutcome.Decisions[0].DecisionType != swf.DecisionTypeStartTimer {
		t.Fatal(histContOutcome.Decisions)
	}

	delete(ctx.eventCorrelator.Activities, "1")
	histContOutcome.Decisions = []*swf.Decision{}

	interceptor.AfterDecision(histCont, ctx, histContOutcome)

	//assert that the workflow was continued
	if len(histContOutcome.Decisions) != 1 || *histContOutcome.Decisions[0].DecisionType != swf.DecisionTypeContinueAsNewWorkflowExecution {
		t.Fatal(histContOutcome.Decisions)
	}

	t.Log(histContOutcome.Decisions[0])

	//test that ContinueSignal is handled
	sigCont := &swf.PollForDecisionTaskOutput{
		Events: []*swf.HistoryEvent{
			{
				EventID:   L(10),
				EventType: S(swf.EventTypeWorkflowExecutionSignaled),
				WorkflowExecutionSignaledEventAttributes: &swf.WorkflowExecutionSignaledEventAttributes{
					SignalName: S(ContinueSignal),
				},
			},
		},
		PreviousStartedEventID: L(7),
	}

	sigContOutcome := &Outcome{
		State:     "state",
		Data:      "data",
		Decisions: []*swf.Decision{},
	}

	ctx = interceptorTestContext()
	ctx.eventCorrelator.checkInit()
	ctx.eventCorrelator.Activities["1"] = &ActivityInfo{}

	interceptor.AfterDecision(sigCont, ctx, sigContOutcome)

	//assert the ContinueTimer was restarted
	if len(sigContOutcome.Decisions) != 1 || *sigContOutcome.Decisions[0].DecisionType != swf.DecisionTypeStartTimer {
		t.Fatal(sigContOutcome.Decisions)
	}

	delete(ctx.eventCorrelator.Activities, "1")
	sigContOutcome.Decisions = []*swf.Decision{}

	interceptor.AfterDecision(sigCont, ctx, sigContOutcome)

	//assert that the workflow was continued
	if len(sigContOutcome.Decisions) != 1 || *sigContOutcome.Decisions[0].DecisionType != swf.DecisionTypeContinueAsNewWorkflowExecution {
		t.Fatal(sigContOutcome.Decisions)
	}

	t.Log(sigContOutcome.Decisions[0])

}

func interceptorTestContext() *FSMContext {
	return NewFSMContext(&FSM{Serializer: &JSONStateSerializer{}},
		swf.WorkflowType{Name: S("foo"), Version: S("1")},
		swf.WorkflowExecution{WorkflowID: S("id"), RunID: S("runid")},
		&EventCorrelator{}, "state", "data", 1)
}
