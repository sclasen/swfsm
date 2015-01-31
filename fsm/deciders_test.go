package fsm

import (
	"testing"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/swf"
)

func TestTypedFuncs(t *testing.T) {
	//this will panic when run if the types arent right.
	typedFuncs := Typed(new(TestingType))
	typedFuncs.Decider(TestingDecider)(new(FSMContext), swf.HistoryEvent{}, new(TestingType))
	typedFuncs.DecisionFunc(TestingDecisionFunc)(new(FSMContext), swf.HistoryEvent{}, new(TestingType))
	typedFuncs.MultiDecisionFunc(TestingMultiDecisionFunc)(new(FSMContext), swf.HistoryEvent{}, new(TestingType))
	typedFuncs.PredicateFunc(TestingPredicateFunc)(new(TestingType))
	typedFuncs.StateFunc(TestingStateFunc)(new(FSMContext), swf.HistoryEvent{}, new(TestingType))
}

type TestingType struct {
	Field string
}

func TestingDecider(ctx *FSMContext, h swf.HistoryEvent, data *TestingType) Outcome {
	return ctx.Stay(data, ctx.EmptyDecisions())
}

func TestingDecisionFunc(ctx *FSMContext, h swf.HistoryEvent, data *TestingType) swf.Decision {
	return swf.Decision{}
}

func TestingMultiDecisionFunc(ctx *FSMContext, h swf.HistoryEvent, data *TestingType) []swf.Decision {
	return []swf.Decision{swf.Decision{}}
}

func TestingPredicateFunc(data *TestingType) bool {
	return false
}

func TestingStateFunc(ctx *FSMContext, h swf.HistoryEvent, data *TestingType) {
}

func TestComposedDecider(t *testing.T) {
	typedFuncs := Typed(new(TestingType))
	composed := NewComposedDecider(
		typedFuncs.Decider(TestingDecider),
		DefaultDecider(),
	)
	composed(new(FSMContext), swf.HistoryEvent{}, new(TestingType))
}

func ExampleComposedDecider() {

	//to reduce boilerplate you can create reusable components to compose Deciders with,
	//that use functions that have the dataType of your FSM.
	typedFuncs := Typed(new(TestingType))

	//for example. reduced boilerplate for the retry of failed activities.
	//first, you would have one of these typed DecisionFuncs for each activity decision type you create.
	fooActivityDecision := func(ctx *FSMContext, h swf.HistoryEvent, data *TestingType) swf.Decision {
		return swf.Decision{
			DecisionType: aws.String(swf.DecisionTypeScheduleActivityTask),
			ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
				ActivityType: &swf.ActivityType{Name: aws.String("foo-activity"), Version: aws.String("1")},
			},
		}
	}

	barActivityDecision := func(ctx *FSMContext, h swf.HistoryEvent, data *TestingType) swf.Decision {
		return swf.Decision{
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
	type TestingTypeDecisionFunc func(*FSMContext, swf.HistoryEvent, *TestingType) swf.Decision

	//now the retryFailedActivities function, which can be used for all activity funcs like the above.
	retryFailedActivities := func(activityName string, activityFn TestingTypeDecisionFunc) Decider {
		typedDecisionFn := typedFuncs.DecisionFunc(activityFn)
		return func(ctx *FSMContext, h swf.HistoryEvent, data interface{}) Outcome {
			switch *h.EventType {
			case swf.EventTypeActivityTaskFailed, swf.EventTypeActivityTaskTimedOut, swf.EventTypeActivityTaskCanceled:
				if *ctx.ActivityInfo(h).Name == activityName {
					decisions := ctx.EmptyDecisions()
					retry := typedDecisionFn(ctx, h, data)
					decisions = append(decisions, retry)
					return ctx.Stay(data, decisions)
				}
			}
			return Pass
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

	decider(new(FSMContext), swf.HistoryEvent{}, new(TestData))

}
