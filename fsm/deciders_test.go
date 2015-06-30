package fsm

import (
	"testing"

	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/sclasen/swfsm/enums/swf"
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
			DecisionType: aws.String(enums.DecisionTypeScheduleActivityTask),
			ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
				ActivityType: &swf.ActivityType{Name: aws.String("foo-activity"), Version: aws.String("1")},
			},
		}
	}

	barActivityDecision := func(ctx *FSMContext, h *swf.HistoryEvent, data *TestingType) *swf.Decision {
		return &swf.Decision{
			DecisionType: aws.String(enums.DecisionTypeScheduleActivityTask),
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
			case enums.EventTypeActivityTaskFailed, enums.EventTypeActivityTaskTimedOut, enums.EventTypeActivityTaskCanceled:
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
		case enums.EventTypeWorkflowExecutionStarted:
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
		case enums.EventTypeChildWorkflowExecutionStarted:
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

func TestOnData(t *testing.T) {}

func TestOnSignalReceived(t *testing.T) {}

func TestOnSignalSent(t *testing.T) {}

func TestOnTimerFired(t *testing.T) {}

func TestOnSignalFailed(t *testing.T) {}

func TestOnActivityCompleted(t *testing.T) {}

func TestOnActivityFailed(t *testing.T) {}

func TestAddDecision(t *testing.T) {}

func TestAddDecisions(t *testing.T) {}

func TestUpdateState(t *testing.T) {}

func TestTransition(t *testing.T) {}

func TestCompleteWorkflow(t *testing.T) {}

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

func deciderTestContext() *FSMContext {
	return NewFSMContext(nil,
		swf.WorkflowType{Name: s.S("foo"), Version: s.S("1")},
		swf.WorkflowExecution{WorkflowID: s.S("id"), RunID: s.S("runid")},
		nil, "state", nil, 1)
}
