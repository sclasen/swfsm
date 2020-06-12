package fsm

import (
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/stretchr/testify/assert"

	s "github.com/sclasen/swfsm/sugar"
)

func TestTypedFuncs(t *testing.T) {
	//this will panic when run if the types arent right.
	typedFuncs := Typed(new(TestingType))
	typedFuncs.Decider(TestingDecider)(new(FSMContext), &swf.HistoryEvent{}, new(TestingType))
	typedFuncs.DecisionFunc(TestingDecisionFunc)(new(FSMContext), &swf.HistoryEvent{}, new(TestingType))
	typedFuncs.MultiDecisionFunc(TestingMultiDecisionFunc)(new(FSMContext), &swf.HistoryEvent{}, new(TestingType))
	typedFuncs.PredicateFunc(TestingPredicateFunc)(new(TestingType))
	typedFuncs.StateFunc(TestingStateFunc)(new(FSMContext), &swf.HistoryEvent{}, new(TestingType))
}

type TestingType struct {
	Field string
}

func TestingDecider(ctx *FSMContext, h *swf.HistoryEvent, data *TestingType) Outcome {
	return ctx.Stay(data, ctx.EmptyDecisions())
}

func TestingDecisionFunc(ctx *FSMContext, h *swf.HistoryEvent, data *TestingType) *swf.Decision {
	return &swf.Decision{}
}

func TestingMultiDecisionFunc(ctx *FSMContext, h *swf.HistoryEvent, data *TestingType) []*swf.Decision {
	return []*swf.Decision{&swf.Decision{}}
}

func TestingPredicateFunc(data *TestingType) bool {
	return false
}

func TestingStateFunc(ctx *FSMContext, h *swf.HistoryEvent, data *TestingType) {
}

func TestComposedDecider(t *testing.T) {
	typedFuncs := Typed(new(TestingType))
	composed := NewComposedDecider(
		typedFuncs.Decider(TestingDecider),
		DefaultDecider(),
	)
	composed(new(FSMContext), &swf.HistoryEvent{}, new(TestingType))
}

func ExampleComposedDecider() {

	//to reduce boilerplate you can create reusable components to compose Deciders with,
	//that use functions that have the dataType of your FSM.
	typedFuncs := Typed(new(TestingType))

	//for example. reduced boilerplate for the retry of failed activities.
	//first, you would have one of these typed DecisionFuncs for each activity decision type you create.
	fooActivityDecision := func(ctx *FSMContext, h *swf.HistoryEvent, data *TestingType) *swf.Decision {
		return &swf.Decision{
			DecisionType: aws.String(swf.DecisionTypeScheduleActivityTask),
			ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
				ActivityType: &swf.ActivityType{Name: aws.String("foo-activity"), Version: aws.String("1")},
			},
		}
	}

	barActivityDecision := func(ctx *FSMContext, h *swf.HistoryEvent, data *TestingType) *swf.Decision {
		return &swf.Decision{
			DecisionType: aws.String(swf.DecisionTypeScheduleActivityTask),
			ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
				ActivityType: &swf.ActivityType{Name: aws.String("bar-activity"), Version: aws.String("1")},
			},
		}
	}

	// optionally a type alias for your 'typed' decision fn.
	// if you dont do this the retryFailedActivities below will need to be
	// func(activityName string, activityFn interface{})
	// instead of
	// func(activityName string, activityFn TestingTypeDecisionFunc)
	type TestingTypeDecisionFunc func(*FSMContext, *swf.HistoryEvent, *TestingType) *swf.Decision

	//now the retryFailedActivities function, which can be used for all activity funcs like the above.
	retryFailedActivities := func(activityName string, activityFn TestingTypeDecisionFunc) Decider {
		typedDecisionFn := typedFuncs.DecisionFunc(activityFn)
		return func(ctx *FSMContext, h *swf.HistoryEvent, data interface{}) Outcome {
			switch *h.EventType {
			case swf.EventTypeActivityTaskFailed, swf.EventTypeActivityTaskTimedOut, swf.EventTypeActivityTaskCanceled:
				if *ctx.ActivityInfo(h).Name == activityName {
					decisions := ctx.EmptyDecisions()
					retry := typedDecisionFn(ctx, h, data)
					decisions = append(decisions, retry)
					return ctx.Stay(data, decisions)
				}
			}
			return ctx.Pass()
		}
	}

	//now build a decider out of the parts.
	//the one thing you need to be careful of is having a unit test that executes the following
	//since the type checking can only be done at initialization at runtime here.
	decider := NewComposedDecider(
		retryFailedActivities("foo-activity", fooActivityDecision),
		retryFailedActivities("bar-activity", barActivityDecision),
		DefaultDecider(),
	)

	decider(new(FSMContext), &swf.HistoryEvent{}, new(TestData))

}

func TestNestedDeciderComposition(t *testing.T) {
	composed := NewComposedDecider(
		NewComposedDecider(
			OnData(func(data interface{}) bool { return true }, Transition("ok"))),
		OnData(func(data interface{}) bool { return true }, Transition("bad")),
		DefaultDecider(),
	)

	ctx := &FSMContext{
		State: "start",
	}

	outcome := composed(ctx, &swf.HistoryEvent{}, new(TestingType))

	if outcome.State != "ok" {
		t.Fatal(outcome)
	}
}

func TestOnStarted(t *testing.T) {
	decider := func(ctx *FSMContext, h *swf.HistoryEvent, data interface{}) Outcome {
		return ctx.Goto("some-state", data, ctx.EmptyDecisions())
	}

	composedDecider := OnStarted(decider)

	for _, et := range s.SWFHistoryEventTypes() {
		ctx := deciderTestContext()
		switch et {
		case swf.EventTypeWorkflowExecutionStarted:
			event := s.EventFromPayload(129, &swf.WorkflowExecutionStartedEventAttributes{})
			data := new(TestData)
			outcome := composedDecider(ctx, event, data)
			expected := decider(ctx, event, data)
			if !reflect.DeepEqual(outcome, expected) {
				t.Fatal("Outcomes not equal", outcome, expected)
			}
		default:
			event := &swf.HistoryEvent{
				EventType: s.S(et),
			}
			if composedDecider(ctx, event, new(TestData)).State != "" {
				t.Fatal("Non nil decision")
			}
		}
	}
}

func TestOnChildStarted(t *testing.T) {
	decider := func(ctx *FSMContext, h *swf.HistoryEvent, data interface{}) Outcome {
		return ctx.Goto("some-state", data, ctx.EmptyDecisions())
	}

	composedDecider := OnChildStarted(decider)

	for _, et := range s.SWFHistoryEventTypes() {
		ctx := deciderTestContext()
		switch et {
		case swf.EventTypeChildWorkflowExecutionStarted:
			event := s.EventFromPayload(129, &swf.ChildWorkflowExecutionStartedEventAttributes{})
			data := new(TestData)
			outcome := composedDecider(ctx, event, data)
			expected := decider(ctx, event, data)
			if !reflect.DeepEqual(outcome, expected) {
				t.Fatal("Outcomes not equal", outcome, expected)
			}
		default:
			event := &swf.HistoryEvent{
				EventType: s.S(et),
			}
			if composedDecider(ctx, event, new(TestData)).State != "" {
				t.Fatal("Non nil decision")
			}
		}
	}
}

func TestOnChildStartedOrAlreadyRunning(t *testing.T) {
	decider := func(ctx *FSMContext, h *swf.HistoryEvent, data interface{}) Outcome {
		return ctx.Goto("some-state", data, ctx.EmptyDecisions())
	}

	composedDecider := OnChildStartedOrAlreadyRunning(decider)

	for _, et := range s.SWFHistoryEventTypes() {
		ctx := deciderTestContext()
		switch et {
		case swf.EventTypeChildWorkflowExecutionStarted:
			event := s.EventFromPayload(129, &swf.ChildWorkflowExecutionStartedEventAttributes{})
			data := new(TestData)
			outcome := composedDecider(ctx, event, data)
			expected := decider(ctx, event, data)
			if !reflect.DeepEqual(outcome, expected) {
				t.Fatal("Outcomes not equal", outcome, expected)
			}
		case swf.EventTypeStartChildWorkflowExecutionFailed:
			{
				event := s.EventFromPayload(129, &swf.StartChildWorkflowExecutionFailedEventAttributes{
					Cause: aws.String(swf.StartChildWorkflowExecutionFailedCauseWorkflowAlreadyRunning),
				})
				data := new(TestData)
				outcome := composedDecider(ctx, event, data)
				expected := decider(ctx, event, data)
				if !reflect.DeepEqual(outcome, expected) {
					t.Fatal("Outcomes not equal", outcome, expected)
				}
			}

			{
				event := s.EventFromPayload(129, &swf.StartChildWorkflowExecutionFailedEventAttributes{
					Cause: aws.String("other failure"),
				})
				if composedDecider(ctx, event, new(TestData)).State != "" {
					t.Fatal("Non nil decision")
				}
			}

		default:
			event := &swf.HistoryEvent{
				EventType: s.S(et),
			}
			if composedDecider(ctx, event, new(TestData)).State != "" {
				t.Fatal("Non nil decision")
			}
		}
	}
}

func TestOnData(t *testing.T) {
	decider := Transition("some-state")
	typed := Typed(new(TestingType))
	data := &TestingType{Field: "yes"}
	ctx := deciderTestContext()

	predicate := typed.PredicateFunc(func(data *TestingType) bool {
		return data.Field == "yes"
	})

	composedDecider := OnData(predicate, decider)

	event := &swf.HistoryEvent{
		EventId:        s.L(129),
		EventTimestamp: aws.Time(time.Now()),
		EventType:      s.S(swf.EventTypeWorkflowExecutionStarted),
	}

	outcome := composedDecider(ctx, event, data)
	expected := decider(ctx, event, data)

	if !reflect.DeepEqual(outcome, expected) {
		t.Fatal("Outcomes not equal", outcome, expected)
	}

	data.Field = "nope"
	outcome = composedDecider(ctx, event, data)

	if reflect.DeepEqual(outcome, expected) {
		t.Fatal("Outcomes should not be equal", outcome, expected)
	}
}

func TestOnDataUnless(t *testing.T) {
	decider := Transition("some-state")
	typed := Typed(new(TestingType))
	data := &TestingType{Field: "no"}
	ctx := deciderTestContext()

	predicate := typed.PredicateFunc(func(data *TestingType) bool {
		return data.Field == "yes"
	})

	composedDecider := OnDataUnless(predicate, decider)

	event := &swf.HistoryEvent{
		EventId:        s.L(129),
		EventTimestamp: aws.Time(time.Now()),
		EventType:      s.S(swf.EventTypeWorkflowExecutionStarted),
	}

	outcome := composedDecider(ctx, event, data)
	expected := decider(ctx, event, data)

	if !reflect.DeepEqual(outcome, expected) {
		t.Fatal("Outcomes not equal", outcome, expected)
	}

	data.Field = "yes"
	outcome = composedDecider(ctx, event, data)

	if reflect.DeepEqual(outcome, expected) {
		t.Fatal("Outcomes should not be equal", outcome, expected)
	}
}

func TestOnSignalReceived(t *testing.T) {
	signal := "the-signal"
	ctx := deciderTestContext()
	decider := Transition("some-state")
	composedDecider := OnSignalReceived(signal, decider)

	for _, e := range s.SWFHistoryEventTypes() {
		switch e {
		case swf.EventTypeWorkflowExecutionSignaled:
			event := &swf.HistoryEvent{
				EventType: s.S(e),
				EventId:   s.L(129),
				WorkflowExecutionSignaledEventAttributes: &swf.WorkflowExecutionSignaledEventAttributes{
					SignalName: s.S("the-signal"),
				},
			}
			data := &TestingType{Field: "yes"}
			outcome := composedDecider(ctx, event, data)
			expected := decider(ctx, event, data)
			if !reflect.DeepEqual(outcome, expected) {
				t.Fatal("Outcomes not equal", outcome, expected)
			}
		default:
			event := &swf.HistoryEvent{
				EventType: s.S(e),
			}
			outcome := composedDecider(ctx, event, nil)
			if !reflect.DeepEqual(outcome, ctx.Pass()) {
				t.Fatal("Outcome does not equal Pass", outcome)
			}
		}
	}
}

func TestOnSignalSent(t *testing.T) {
	signal := "the-signal"
	decider := Transition("some-state")
	composedDecider := OnSignalSent(signal, decider)

	ctx := testContextWithSignal(123, &swf.SignalExternalWorkflowExecutionInitiatedEventAttributes{
		SignalName: s.S(signal),
		WorkflowId: s.S("the-workflow"),
	})()

	for _, e := range s.SWFHistoryEventTypes() {
		switch e {
		case swf.EventTypeExternalWorkflowExecutionSignaled:
			event := &swf.HistoryEvent{
				EventType: s.S(e),
				EventId:   s.L(129),
				ExternalWorkflowExecutionSignaledEventAttributes: &swf.ExternalWorkflowExecutionSignaledEventAttributes{
					InitiatedEventId: s.L(123),
				},
			}

			data := &TestingType{Field: "yes"}
			outcome := composedDecider(ctx, event, data)
			expected := decider(ctx, event, data)
			if !reflect.DeepEqual(outcome, expected) {
				t.Fatal("Outcomes not equal", outcome, expected)
			}
		default:
			event := &swf.HistoryEvent{
				EventType: s.S(e),
			}
			outcome := composedDecider(ctx, event, nil)
			if !reflect.DeepEqual(outcome, ctx.Pass()) {
				t.Fatal("Outcome does not equal Pass", outcome)
			}
		}
	}
}

func TestOnTimerFired(t *testing.T) {}

func TestOnSignalFailed(t *testing.T) {
	signal := "the-signal"
	decider := Transition("some-state")
	composedDecider := OnSignalFailed(signal, decider)

	ctx := testContextWithSignal(123, &swf.SignalExternalWorkflowExecutionInitiatedEventAttributes{
		SignalName: s.S(signal),
		WorkflowId: s.S("the-workflow"),
	})()

	for _, e := range s.SWFHistoryEventTypes() {
		switch e {
		case swf.EventTypeSignalExternalWorkflowExecutionFailed:
			event := &swf.HistoryEvent{
				EventType: s.S(e),
				EventId:   s.L(129),
				SignalExternalWorkflowExecutionFailedEventAttributes: &swf.SignalExternalWorkflowExecutionFailedEventAttributes{
					InitiatedEventId: s.L(123),
				},
			}

			data := &TestingType{Field: "yes"}
			outcome := composedDecider(ctx, event, data)
			expected := decider(ctx, event, data)
			if !reflect.DeepEqual(outcome, expected) {
				t.Fatal("Outcomes not equal", outcome, expected)
			}
		default:
			event := &swf.HistoryEvent{
				EventType: s.S(e),
			}
			outcome := composedDecider(ctx, event, nil)
			if !reflect.DeepEqual(outcome, ctx.Pass()) {
				t.Fatal("Outcome does not equal Pass", outcome)
			}
		}
	}
}

func TestOnActivityCompleted(t *testing.T) {
	activity := "test-activity"
	decider := Transition("some-state")
	composedDecider := OnActivityCompleted(activity, decider)

	ctx := testContextWithActivity(123, &swf.ActivityTaskScheduledEventAttributes{
		ActivityId: s.S("the-id"),
		ActivityType: &swf.ActivityType{
			Name:    s.S(activity),
			Version: s.S("1"),
		},
	},
	)

	for _, e := range s.SWFHistoryEventTypes() {
		switch e {
		case swf.EventTypeActivityTaskCompleted:
			event := &swf.HistoryEvent{
				EventType:                            s.S(e),
				EventId:                              s.L(129),
				ActivityTaskCompletedEventAttributes: &swf.ActivityTaskCompletedEventAttributes{ScheduledEventId: s.L(123)},
			}
			data := &TestingType{Field: "yes"}
			outcome := composedDecider(ctx(), event, data)
			expected := decider(ctx(), event, data)
			if !reflect.DeepEqual(outcome, expected) {
				t.Fatal("Outcomes not equal", outcome, expected)
			}
		default:
			event := &swf.HistoryEvent{
				EventType: s.S(e),
			}
			outcome := composedDecider(ctx(), event, nil)
			if !reflect.DeepEqual(outcome, ctx().Pass()) {
				t.Fatal("Outcome does not equal Pass", outcome)
			}
		}
	}
}

func TestOnActivityFailedTimedOutCanceled(t *testing.T) {
	activity := "test-activity"
	decider := Transition("some-state")
	composedDecider := OnActivityFailedTimedOutCanceled(activity, decider)

	ctx := testContextWithActivity(123, &swf.ActivityTaskScheduledEventAttributes{
		ActivityId: s.S("the-id"),
		ActivityType: &swf.ActivityType{
			Name:    s.S(activity),
			Version: s.S("1"),
		},
	},
	)

	for _, e := range s.SWFHistoryEventTypes() {
		switch e {
		case swf.EventTypeActivityTaskFailed, swf.EventTypeActivityTaskTimedOut, swf.EventTypeActivityTaskCanceled:
			event := &swf.HistoryEvent{
				EventType:                           s.S(e),
				EventId:                             s.L(129),
				ActivityTaskCanceledEventAttributes: &swf.ActivityTaskCanceledEventAttributes{ScheduledEventId: s.L(123)},
				ActivityTaskFailedEventAttributes:   &swf.ActivityTaskFailedEventAttributes{ScheduledEventId: s.L(123)},
				ActivityTaskTimedOutEventAttributes: &swf.ActivityTaskTimedOutEventAttributes{ScheduledEventId: s.L(123)},
			}
			data := &TestingType{Field: "yes"}
			outcome := composedDecider(ctx(), event, data)
			expected := decider(ctx(), event, data)
			if !reflect.DeepEqual(outcome, expected) {
				t.Fatal("Outcomes not equal", outcome, expected)
			}
		default:
			event := &swf.HistoryEvent{
				EventType: s.S(e),
			}
			outcome := composedDecider(ctx(), event, nil)
			if !reflect.DeepEqual(outcome, ctx().Pass()) {
				t.Fatal("Outcome does not equal Pass", outcome)
			}
		}
	}
}

func TestOnActivityFailed(t *testing.T) {}

func TestOnChildStartFailed(t *testing.T) {}

func TestOnChildCompleted(t *testing.T) {}

func TestOnStartTimerFailed(t *testing.T) {}

func TestAddDecision(t *testing.T) {}

func TestAddDecisions(t *testing.T) {}

func TestUpdateState(t *testing.T) {}

func TestTransition(t *testing.T) {}

func TestCompleteWorkflow(t *testing.T) {}

func TestFailWorkflow(t *testing.T) {
	// arrange
	data := &TestingType{"Some data"}
	fsmContext := NewFSMContext(nil,
		swf.WorkflowType{Name: s.S("foo"), Version: s.S("1")},
		swf.WorkflowExecution{WorkflowId: s.S("id"), RunId: s.S("runid")},
		nil, "state", nil, 1)
	details := "Some failure message"
	// act
	failDecider := FailWorkflow(s.S(details))
	outcome := failDecider(fsmContext, nil, data)

	// assert
	assert.Equal(t, data, outcome.Data, "Expected data to be passed into failed outcome")
	failDecision := FindDecision(outcome.Decisions, failWorkflowPredicate)
	assert.NotNil(t, failDecision, "Expected to find a fail workflow decision in the outcome")
	assert.Equal(t, details, *failDecision.FailWorkflowExecutionDecisionAttributes.Details,
		"Expected details in the fail decision to match what was passed in")
}

func TestStay(t *testing.T) {}

func testContextWithActivity(scheduledEventId int, event *swf.ActivityTaskScheduledEventAttributes) func() *FSMContext {
	return func() *FSMContext {
		correlator := &EventCorrelator{}
		correlator.Serializer = JSONStateSerializer{}
		correlator.Track(s.EventFromPayload(scheduledEventId, event))
		ctx := deciderTestContext()
		ctx.eventCorrelator = correlator
		return ctx
	}
}

func testContextWithSignal(scheduledEventId int, event *swf.SignalExternalWorkflowExecutionInitiatedEventAttributes) func() *FSMContext {
	return func() *FSMContext {
		correlator := &EventCorrelator{}
		correlator.Track(&swf.HistoryEvent{
			EventId:   s.I(scheduledEventId),
			EventType: s.S(swf.EventTypeSignalExternalWorkflowExecutionInitiated),
			SignalExternalWorkflowExecutionInitiatedEventAttributes: event,
		})

		return NewFSMContext(nil,
			swf.WorkflowType{Name: s.S("foo"), Version: s.S("1")},
			swf.WorkflowExecution{WorkflowId: s.S("id"), RunId: s.S("runid")},
			correlator, "state", nil, 1)
	}
}

func deciderTestContext() *FSMContext {
	return NewFSMContext(nil,
		swf.WorkflowType{Name: s.S("foo"), Version: s.S("1")},
		swf.WorkflowExecution{WorkflowId: s.S("id"), RunId: s.S("runid")},
		nil, "state", nil, 1)
}
