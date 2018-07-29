package fsm

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/stretchr/testify/assert"

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
		SystemSerializer:    JSONStateSerializer{},
	}

	fsm.AddInitialState(&FSMState{Name: "initial", Decider: func(ctx *FSMContext, e *swf.HistoryEvent, d interface{}) Outcome {
		return Outcome{State: "initial", Data: d, Decisions: []*swf.Decision{}}
	}})

	decisionTask := new(swf.PollForDecisionTaskOutput)
	decisionTask.WorkflowExecution = new(swf.WorkflowExecution)
	decisionTask.WorkflowType = &swf.WorkflowType{Name: S("test"), Version: S("1")}
	decisionTask.WorkflowExecution.RunId = S("run")
	decisionTask.WorkflowExecution.WorkflowId = S("wf")
	decisionTask.PreviousStartedEventId = I(5)
	decisionTask.StartedEventId = I(15)
	decisionTask.Events = []*swf.HistoryEvent{
		{
			EventId:   I(10),
			EventType: S("WorkflowExecutionStarted"),
			WorkflowExecutionStartedEventAttributes: &swf.WorkflowExecutionStartedEventAttributes{
				Input: StartFSMWorkflowInput(fsm, new(TestData)),
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
				EventId:   L(1),
				EventType: S(swf.EventTypeWorkflowExecutionStarted),
			},
		},
		PreviousStartedEventId: L(0),
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
				EventId:   L(2),
				EventType: S(swf.EventTypeTimerFired),
				TimerFiredEventAttributes: &swf.TimerFiredEventAttributes{
					TimerId: S(ContinueTimer),
				},
			},
		},
		PreviousStartedEventId: L(1),
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
				EventId:   L(10),
				EventType: S(swf.EventTypeExternalWorkflowExecutionSignaled), //n
			},
		},
		PreviousStartedEventId: L(7),
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
				EventId:   L(10),
				EventType: S(swf.EventTypeWorkflowExecutionSignaled),
				WorkflowExecutionSignaledEventAttributes: &swf.WorkflowExecutionSignaledEventAttributes{
					SignalName: S(ContinueSignal),
				},
			},
		},
		PreviousStartedEventId: L(7),
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

func TestWorkflowStartCancel(t *testing.T) {
	ctx := interceptorTestContext()

	decisions := []*swf.Decision{
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeStartChildWorkflowExecution),
			StartChildWorkflowExecutionDecisionAttributes: &swf.StartChildWorkflowExecutionDecisionAttributes{
				WorkflowId: S("dyno-foobar"),
			},
		},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeRequestCancelExternalWorkflowExecution),
			RequestCancelExternalWorkflowExecutionDecisionAttributes: &swf.RequestCancelExternalWorkflowExecutionDecisionAttributes{
				WorkflowId: S("dyno-foobar"),
			},
		},
		&swf.Decision{DecisionType: S(swf.DecisionTypeRecordMarker)},
	}

	decisions = handleStartCancelTypes(decisions, ctx)
	if len(decisions) != 1 {
		t.Fatal("more than 1 decision left after interceptor")
	}
}

func TestWorkflowCancelStart(t *testing.T) {
	ctx := interceptorTestContext()

	decisions := []*swf.Decision{
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeRequestCancelExternalWorkflowExecution),
			RequestCancelExternalWorkflowExecutionDecisionAttributes: &swf.RequestCancelExternalWorkflowExecutionDecisionAttributes{
				WorkflowId: S("dyno-foobar"),
			},
		},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeStartChildWorkflowExecution),
			StartChildWorkflowExecutionDecisionAttributes: &swf.StartChildWorkflowExecutionDecisionAttributes{
				WorkflowId: S("dyno-foobar"),
			},
		},
		&swf.Decision{DecisionType: S(swf.DecisionTypeRecordMarker)},
	}

	decisions = handleStartCancelTypes(decisions, ctx)
	if len(decisions) != 3 {
		t.Fatal("incorrect number of decisions left after interceptor")
	}

}

func TestManyWorkflowStarts(t *testing.T) {
	ctx := interceptorTestContext()

	decisions := []*swf.Decision{
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeStartChildWorkflowExecution),
			StartChildWorkflowExecutionDecisionAttributes: &swf.StartChildWorkflowExecutionDecisionAttributes{
				WorkflowId: S("dyno-foobar"),
			},
		},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeStartChildWorkflowExecution),
			StartChildWorkflowExecutionDecisionAttributes: &swf.StartChildWorkflowExecutionDecisionAttributes{
				WorkflowId: S("dyno-foobar"),
			},
		},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeStartChildWorkflowExecution),
			StartChildWorkflowExecutionDecisionAttributes: &swf.StartChildWorkflowExecutionDecisionAttributes{
				WorkflowId: S("dyno-foobar"),
			},
		},
		&swf.Decision{DecisionType: S(swf.DecisionTypeRecordMarker)},
	}

	decisions = handleStartCancelTypes(decisions, ctx)
	if len(decisions) != 4 {
		t.Fatal("incorrect number of start decisions left after interceptor")
	}
}

func TestStartCancelDifferingWorkflows(t *testing.T) {
	ctx := interceptorTestContext()

	decisions := []*swf.Decision{
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeStartChildWorkflowExecution),
			StartChildWorkflowExecutionDecisionAttributes: &swf.StartChildWorkflowExecutionDecisionAttributes{
				WorkflowId: S("dyno-foobar"),
			},
		},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeRequestCancelExternalWorkflowExecution),
			RequestCancelExternalWorkflowExecutionDecisionAttributes: &swf.RequestCancelExternalWorkflowExecutionDecisionAttributes{
				WorkflowId: S("dyno-baz"),
			},
		},
		&swf.Decision{DecisionType: S(swf.DecisionTypeRecordMarker)},
	}

	decisions = handleStartCancelTypes(decisions, ctx)
	if len(decisions) != 3 {
		t.Fatal("incorrect number of differing workflow decisions left after interceptor")
	}
}

func TestActivityStartCancel(t *testing.T) {
	ctx := interceptorTestContext()

	decisions := []*swf.Decision{
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeScheduleActivityTask),
			ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
				ActivityId: S("foobar"),
			},
		},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeRequestCancelActivityTask),
			RequestCancelActivityTaskDecisionAttributes: &swf.RequestCancelActivityTaskDecisionAttributes{
				ActivityId: S("foobar"),
			},
		},
		&swf.Decision{DecisionType: S(swf.DecisionTypeRecordMarker)},
	}

	decisions = handleStartCancelTypes(decisions, ctx)
	if len(decisions) != 1 {
		t.Fatal("more than 1 decision left after interceptor")
	}

}

func TestManyActivityStarts(t *testing.T) {
	ctx := interceptorTestContext()

	decisions := []*swf.Decision{
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeScheduleActivityTask),
			ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
				ActivityId: S("foobar"),
			},
		},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeScheduleActivityTask),
			ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
				ActivityId: S("foobar"),
			},
		},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeScheduleActivityTask),
			ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
				ActivityId: S("foobar"),
			},
		},
		&swf.Decision{DecisionType: S(swf.DecisionTypeRecordMarker)},
	}

	decisions = handleStartCancelTypes(decisions, ctx)
	if len(decisions) != 4 {
		t.Fatal("incorrect number of start decisions left after interceptor")
	}
}

func TestStartCancelDifferingActivities(t *testing.T) {
	ctx := interceptorTestContext()

	decisions := []*swf.Decision{
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeScheduleActivityTask),
			ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
				ActivityId: S("foobar"),
			},
		},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeRequestCancelActivityTask),
			RequestCancelActivityTaskDecisionAttributes: &swf.RequestCancelActivityTaskDecisionAttributes{
				ActivityId: S("foobar-duex"),
			},
		},
		&swf.Decision{DecisionType: S(swf.DecisionTypeRecordMarker)},
	}

	decisions = handleStartCancelTypes(decisions, ctx)
	if len(decisions) != 3 {
		t.Fatal("incorrect number of differing activities decisions left after interceptor")
	}
}

func TestTimerStartCancel(t *testing.T) {
	ctx := interceptorTestContext()

	decisions := []*swf.Decision{
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeStartTimer),
			StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
				TimerId: S("baz"),
			},
		},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeCancelTimer),
			CancelTimerDecisionAttributes: &swf.CancelTimerDecisionAttributes{
				TimerId: S("baz"),
			},
		},
		&swf.Decision{DecisionType: S(swf.DecisionTypeRecordMarker)},
	}

	decisions = handleStartCancelTypes(decisions, ctx)
	if len(decisions) != 1 {
		t.Fatal("more than 1 timer decision left after interceptor")
	}

}

func TestManyTimerStarts(t *testing.T) {
	ctx := interceptorTestContext()

	decisions := []*swf.Decision{
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeStartTimer),
			StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
				TimerId: S("baz"),
			},
		},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeStartTimer),
			StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
				TimerId: S("baz"),
			},
		},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeStartTimer),
			StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
				TimerId: S("baz"),
			},
		},
		&swf.Decision{DecisionType: S(swf.DecisionTypeRecordMarker)},
	}

	decisions = handleStartCancelTypes(decisions, ctx)
	if len(decisions) != 4 {
		t.Fatal("incorrect number of timer start decisions left after interceptor")
	}
}

func TestStartCancelDifferingTimers(t *testing.T) {
	ctx := interceptorTestContext()

	decisions := []*swf.Decision{
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeStartTimer),
			StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
				TimerId: S("baz"),
			},
		},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeCancelTimer),
			CancelTimerDecisionAttributes: &swf.CancelTimerDecisionAttributes{
				TimerId: S("bar"),
			},
		},
		&swf.Decision{DecisionType: S(swf.DecisionTypeRecordMarker)},
	}

	decisions = handleStartCancelTypes(decisions, ctx)
	if len(decisions) != 3 {
		t.Fatal("incorrect number of differing timer decisions left after interceptor")
	}
}

func TestMixedDecisionInterceptions(t *testing.T) {
	ctx := interceptorTestContext()
	decisions := []*swf.Decision{
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeStartTimer),
			StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
				TimerId: S("baz"),
			},
		},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeCancelTimer),
			CancelTimerDecisionAttributes: &swf.CancelTimerDecisionAttributes{
				TimerId: S("baz"),
			},
		},
		&swf.Decision{DecisionType: S(swf.DecisionTypeRecordMarker)},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeScheduleActivityTask),
			ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
				ActivityId: S("foobar"),
			},
		},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeScheduleActivityTask),
			ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
				ActivityId: S("foobar"),
			},
		},
		&swf.Decision{
			DecisionType: S(swf.DecisionTypeRequestCancelActivityTask),
			RequestCancelActivityTaskDecisionAttributes: &swf.RequestCancelActivityTaskDecisionAttributes{
				ActivityId: S("foobar"),
			},
		},
	}
	decisions = handleStartCancelTypes(decisions, ctx)
	if len(decisions) != 2 {
		t.Fatal("incorrect number of mixed decisions left after interceptor")
	}
}

func TestDedupeWorkflowCompletesExpectsOneCompleteReturned(t *testing.T) {
	// arrange
	outcome := &Outcome{
		State:     "state",
		Data:      "data",
		Decisions: []*swf.Decision{completeDecision(), completeDecision(), completeDecision()},
	}
	interceptor := DedupeWorkflowCompletes()

	// act
	interceptor.AfterDecision(nil, interceptorTestContext(), outcome)

	// assert
	assert.Len(t, outcome.Decisions, 1, "Expected outcome to only have 1 decision because completes were deduped")
}

func TestDedupeWorkflowCancellationsExpectsOneCancelReturned(t *testing.T) {
	// arrange
	outcome := &Outcome{
		State:     "state",
		Data:      "data",
		Decisions: []*swf.Decision{cancelDecision(), cancelDecision(), cancelDecision()},
	}
	interceptor := DedupeWorkflowCancellations()

	// act
	interceptor.AfterDecision(nil, interceptorTestContext(), outcome)

	// assert
	assert.Len(t, outcome.Decisions, 1, "Expected outcome to only have 1 decision because cancellations were deduped")
}

func TestDedupeWorkflowFailuresExpectsOneFailureReturned(t *testing.T) {
	// arrange
	outcome := &Outcome{
		State:     "state",
		Data:      "data",
		Decisions: []*swf.Decision{failDecision(), failDecision(), failDecision()},
	}
	interceptor := DedupeWorkflowFailures()

	// act
	interceptor.AfterDecision(nil, interceptorTestContext(), outcome)

	// assert
	assert.Len(t, outcome.Decisions, 1, "Expected outcome to only have 1 decision because failures were deduped")
}

func TestDedupeDecisionsExpectsLastDuplicateToRemain(t *testing.T) {
	// arrange
	outcome := &Outcome{
		State:     "state",
		Data:      "data",
		Decisions: []*swf.Decision{completeDecision(), failDecision(), completeDecision(), cancelDecision(), completeDecision(), cancelDecision()},
	}
	interceptor := DedupeDecisions(swf.DecisionTypeCompleteWorkflowExecution)

	// act
	interceptor.AfterDecision(nil, interceptorTestContext(), outcome)

	// assert
	assert.Len(t, outcome.Decisions, 4, "Expected outcome to have 4 decisions after deduping"+
		" 'completes' because it started with 6 decisions, 3 of which were 'complete' type")
	assert.Equal(t, []*swf.Decision{failDecision(), cancelDecision(), completeDecision(), cancelDecision()},
		outcome.Decisions, "Expected outcome decisions to match the expected list of decisions that have 'completes' deduped.")
}

func TestDedupeWorkflowCloseDecisionsExpectsDuplicatesRemoved(t *testing.T) {
	// arrange
	outcome := &Outcome{
		State:     "state",
		Data:      "data",
		Decisions: []*swf.Decision{completeDecision(), completeDecision(), cancelDecision(), cancelDecision(), failDecision(), failDecision()},
	}
	interceptor := DedupeWorkflowCloseDecisions()

	// act
	interceptor.AfterDecision(nil, interceptorTestContext(), outcome)

	// assert
	assert.Len(t, outcome.Decisions, 3, "Expected outcome to have 3 decisions after deduping"+
		" because it started with 6 decisions, 3 of which were duplicate close decisions")
	assert.Equal(t, []*swf.Decision{completeDecision(), cancelDecision(), failDecision()},
		outcome.Decisions, "Expected outcome decisions to match the expected list of decisions that have 'close decisions' deduped.")
}

func TestMoveDecisionsToEndExpectsAllDecisionsOfTypeAtEnd(t *testing.T) {
	// arrange
	outcome := &Outcome{
		State:     "state",
		Data:      "data",
		Decisions: []*swf.Decision{completeDecision(), failDecision(), completeDecision(), cancelDecision(), completeDecision(), cancelDecision()},
	}
	interceptor := MoveDecisionsToEnd(swf.DecisionTypeCompleteWorkflowExecution)

	// act
	interceptor.AfterDecision(nil, interceptorTestContext(), outcome)

	// assert
	assert.Equal(t, []*swf.Decision{failDecision(), cancelDecision(), cancelDecision(), completeDecision(), completeDecision(), completeDecision()},
		outcome.Decisions, "Expected outcome decisions to have all 'complete' decisions at the end")
}

func TestMoveDecisionsToEndWhenNoMatchingDecisionsExpectsNoChange(t *testing.T) {
	// arrange
	outcome := &Outcome{
		State:     "state",
		Data:      "data",
		Decisions: []*swf.Decision{failDecision(), cancelDecision(), cancelDecision()},
	}
	interceptor := MoveDecisionsToEnd(swf.DecisionTypeCompleteWorkflowExecution)

	// act
	interceptor.AfterDecision(nil, interceptorTestContext(), outcome)

	// assert
	assert.Equal(t, []*swf.Decision{failDecision(), cancelDecision(), cancelDecision()},
		outcome.Decisions, "Expected outcome decisions to have stayed the same because no matching types are in the list")
}

func TestMoveWorkflowCloseDecisionsToEndExpectsFailDecisionAtEnd(t *testing.T) {
	// arrange
	outcome := &Outcome{
		State:     "state",
		Data:      "data",
		Decisions: []*swf.Decision{failDecision(), timerDecision()},
	}
	interceptor := MoveWorkflowCloseDecisionsToEnd()

	// act
	interceptor.AfterDecision(nil, interceptorTestContext(), outcome)

	// assert
	assert.Equal(t, []*swf.Decision{timerDecision(), failDecision()},
		outcome.Decisions, "Expected fail decision to have moved to the end because it is 'workflow closing' decision")
}

func TestMoveWorkflowCloseDecisionsToEndExpectsCancelDecisionAtEnd(t *testing.T) {
	// arrange
	outcome := &Outcome{
		State:     "state",
		Data:      "data",
		Decisions: []*swf.Decision{cancelDecision(), timerDecision()},
	}
	interceptor := MoveWorkflowCloseDecisionsToEnd()

	// act
	interceptor.AfterDecision(nil, interceptorTestContext(), outcome)

	// assert
	assert.Equal(t, []*swf.Decision{timerDecision(), cancelDecision()},
		outcome.Decisions, "Expected cancel decision to have moved to the end because it is 'workflow closing' decision")
}

func TestMoveWorkflowCloseDecisionsToEndExpectsCompleteDecisionAtEnd(t *testing.T) {
	// arrange
	outcome := &Outcome{
		State:     "state",
		Data:      "data",
		Decisions: []*swf.Decision{completeDecision(), timerDecision()},
	}
	interceptor := MoveWorkflowCloseDecisionsToEnd()

	// act
	interceptor.AfterDecision(nil, interceptorTestContext(), outcome)

	// assert
	assert.Equal(t, []*swf.Decision{timerDecision(), completeDecision()},
		outcome.Decisions, "Expected complete decision to have moved to the end because it is 'workflow closing' decision")
}

func TestRemoveLowerPriorityDecisionsExpectsLowerPriorityRemoved(t *testing.T) {
	// arrange
	outcome := &Outcome{
		State: "state",
		Data:  "data",
		Decisions: []*swf.Decision{completeDecision(), timerDecision(), completeDecision(),
			cancelDecision(), completeDecision(), timerDecision(), cancelDecision()},
	}
	interceptor := RemoveLowerPriorityDecisions(
		swf.DecisionTypeFailWorkflowExecution,
		swf.DecisionTypeCompleteWorkflowExecution,
		swf.DecisionTypeCancelWorkflowExecution)

	// act
	interceptor.AfterDecision(nil, interceptorTestContext(), outcome)

	// assert
	assert.Len(t, outcome.Decisions, 5, "Expected 5 decisions because two cancel decisions should have been removed")
	assert.Equal(t, []*swf.Decision{completeDecision(), timerDecision(), completeDecision(), completeDecision(), timerDecision()},
		outcome.Decisions, "Expected lower priority decisions to have been removed")
}

func TestCloseWorkflowRemoveIncompatibleDecisionInterceptor(t *testing.T) {
	// arrange
	outcome := &Outcome{
		State:     "state",
		Data:      "data",
		Decisions: []*swf.Decision{timerDecision(), scheduleActivityDecision(), completeDecision()},
	}
	interceptor := CloseWorkflowRemoveIncompatibleDecisionInterceptor()

	// act
	interceptor.AfterDecision(nil, interceptorTestContext(), outcome)

	// assert
	assert.Len(t, outcome.Decisions, 1, "Expected outcome to only have 1 decision because incompatables were removed")
}

func scheduleActivityDecision() *swf.Decision {
	return &swf.Decision{
		DecisionType: S(swf.DecisionTypeScheduleActivityTask),
		ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
			ActivityId: S("foobar"),
		},
	}
}

func completeDecision() *swf.Decision {
	return &swf.Decision{
		CompleteWorkflowExecutionDecisionAttributes: &swf.CompleteWorkflowExecutionDecisionAttributes{},
		DecisionType:                                S(swf.DecisionTypeCompleteWorkflowExecution),
	}
}

func cancelDecision() *swf.Decision {
	return &swf.Decision{
		CancelWorkflowExecutionDecisionAttributes: &swf.CancelWorkflowExecutionDecisionAttributes{},
		DecisionType:                              S(swf.DecisionTypeCancelWorkflowExecution),
	}
}

func failDecision() *swf.Decision {
	return &swf.Decision{
		FailWorkflowExecutionDecisionAttributes: &swf.FailWorkflowExecutionDecisionAttributes{},
		DecisionType:                            S(swf.DecisionTypeFailWorkflowExecution),
	}
}

func timerDecision() *swf.Decision {
	return &swf.Decision{
		DecisionType: S(swf.DecisionTypeStartTimer),
		StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
			TimerId: S("baz"),
		},
	}
}

func interceptorTestContext() *FSMContext {
	return NewFSMContext(&FSM{Serializer: &JSONStateSerializer{}},
		swf.WorkflowType{Name: S("foo"), Version: S("1")},
		swf.WorkflowExecution{WorkflowId: S("id"), RunId: S("runid")},
		&EventCorrelator{}, "state", "data", 1)
}
