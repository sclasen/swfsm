package testing

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/sclasen/swfsm/fsm"
	. "github.com/sclasen/swfsm/sugar"
	te "testing"
)

func TestThrottleSignalInterceptor(t *te.T) {

	interceptor := fsm.NewComposedDecisionInterceptor(
		TestThrotteSignalsOnceInterceptor(),
		TestThrotteCancelsOnceInterceptor(),
		TestThrotteChildrenOnceInterceptor(),
		TestThrotteTimersOnceInterceptor(10),
	)

	ctx := interceptorTestContext()

	first := &fsm.Outcome{
		State: "steady",
		Data:  &TestData{},
		Decisions: []*swf.Decision{
			{
				DecisionType: S(swf.DecisionTypeSignalExternalWorkflowExecution),
				SignalExternalWorkflowExecutionDecisionAttributes: &swf.SignalExternalWorkflowExecutionDecisionAttributes{
					SignalName: S("the-signal"),
					WorkflowId: S("to-signal"),
				},
			},
			{
				DecisionType: S(swf.DecisionTypeRequestCancelExternalWorkflowExecution),
				RequestCancelExternalWorkflowExecutionDecisionAttributes: &swf.RequestCancelExternalWorkflowExecutionDecisionAttributes{
					WorkflowId: S("to-cancel"),
				},
			},
			{
				DecisionType: S(swf.DecisionTypeStartChildWorkflowExecution),
				StartChildWorkflowExecutionDecisionAttributes: &swf.StartChildWorkflowExecutionDecisionAttributes{
					WorkflowId:   S("the-child"),
					WorkflowType: &swf.WorkflowType{Name: S("foo"), Version: S("1")},
				},
			},
			{
				DecisionType: S(swf.DecisionTypeStartTimer),
				StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
					TimerId:            S("the-timer"),
					StartToFireTimeout: S("12345"),
				},
			},
		},
	}

	firstTask := &swf.PollForDecisionTaskOutput{
		Events: []*swf.HistoryEvent{
			{
				EventType: S(swf.EventTypeWorkflowExecutionStarted), //triggers the timers interceptor
				EventId:   L(1),
			},
		},
		PreviousStartedEventId: L(0),
	}

	interceptor.AfterDecision(firstTask, ctx, first)

	if *first.Decisions[0].SignalExternalWorkflowExecutionDecisionAttributes.WorkflowId == "to-signal" {
		t.Fatal("signal not intercepted in after", PrettyDecision(*first.Decisions[0]))
	}

	if *first.Decisions[1].RequestCancelExternalWorkflowExecutionDecisionAttributes.WorkflowId == "to-cancel" {
		t.Fatal("cancel not intercepted in after", PrettyDecision(*first.Decisions[1]))
	}

	if *first.Decisions[2].StartChildWorkflowExecutionDecisionAttributes.WorkflowId == "the-child" {
		t.Fatal("child not intercepted in after", PrettyDecision(*first.Decisions[2]))
	}

	if *first.Decisions[3].StartTimerDecisionAttributes.TimerId == "the-timer" {
		t.Fatal("timer not intercepted in after", PrettyDecision(*first.Decisions[3]))
	}

	secondTask := &swf.PollForDecisionTaskOutput{
		Events: []*swf.HistoryEvent{
			{
				EventType: S(swf.EventTypeStartTimerFailed),
				EventId:   L(4),
				StartTimerFailedEventAttributes: &swf.StartTimerFailedEventAttributes{
					TimerId: first.Decisions[3].StartTimerDecisionAttributes.TimerId,
				},
			},
			{
				EventType: S(swf.EventTypeStartChildWorkflowExecutionFailed),
				EventId:   L(3),
				StartChildWorkflowExecutionFailedEventAttributes: &swf.StartChildWorkflowExecutionFailedEventAttributes{
					WorkflowId: S(fmt.Sprintf("fail-on-purpose-%s", "the-child")),
				},
			},
			{
				EventType: S(swf.EventTypeRequestCancelExternalWorkflowExecutionFailed),
				EventId:   L(2),
				RequestCancelExternalWorkflowExecutionFailedEventAttributes: &swf.RequestCancelExternalWorkflowExecutionFailedEventAttributes{
					WorkflowId: S(fmt.Sprintf("fail-on-purpose-%s", "to-cancel")),
				},
			},
			{
				EventType: S(swf.EventTypeSignalExternalWorkflowExecutionFailed),
				EventId:   L(1),
				SignalExternalWorkflowExecutionFailedEventAttributes: &swf.SignalExternalWorkflowExecutionFailedEventAttributes{
					WorkflowId: S(fmt.Sprintf("fail-on-purpose-%s-%s", "to-signal", "the-signal")),
				},
			},
		},
		PreviousStartedEventId: L(0),
	}

	interceptor.BeforeDecision(secondTask, ctx, first)

	if secondTask.Events[0].StartTimerFailedEventAttributes.Cause == nil ||
		*secondTask.Events[0].StartTimerFailedEventAttributes.Cause != swf.StartTimerFailedCauseTimerCreationRateExceeded {
		t.Fatal("timer failed not rewriten ", PrettyHistoryEvent(secondTask.Events[0]))
	}

	if secondTask.Events[1].StartChildWorkflowExecutionFailedEventAttributes.Cause == nil ||
		*secondTask.Events[1].StartChildWorkflowExecutionFailedEventAttributes.Cause != swf.StartChildWorkflowExecutionFailedCauseChildCreationRateExceeded {
		t.Fatal("start child failed not rewriten ", PrettyHistoryEvent(secondTask.Events[1]))
	}

	if secondTask.Events[2].RequestCancelExternalWorkflowExecutionFailedEventAttributes.Cause == nil ||
		*secondTask.Events[2].RequestCancelExternalWorkflowExecutionFailedEventAttributes.Cause != swf.RequestCancelExternalWorkflowExecutionFailedCauseRequestCancelExternalWorkflowExecutionRateExceeded {
		t.Fatal("request cancel failed not rewriten ", PrettyHistoryEvent(secondTask.Events[2]))
	}

	if secondTask.Events[3].SignalExternalWorkflowExecutionFailedEventAttributes.Cause == nil ||
		*secondTask.Events[3].SignalExternalWorkflowExecutionFailedEventAttributes.Cause != swf.SignalExternalWorkflowExecutionFailedCauseSignalExternalWorkflowExecutionRateExceeded {
		t.Fatal("start timer failed not rewriten ", PrettyHistoryEvent(secondTask.Events[3]))
	}

}

func interceptorTestContext() *fsm.FSMContext {
	return fsm.NewFSMContext(&fsm.FSM{Serializer: &fsm.JSONStateSerializer{}},
		swf.WorkflowType{Name: S("foo"), Version: S("1")},
		swf.WorkflowExecution{WorkflowId: S("id"), RunId: S("runid")},
		&fsm.EventCorrelator{}, "state", "data", 1)
}

type TestData struct{}
