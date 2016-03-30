package fsm

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/swf"

	. "github.com/sclasen/swfsm/sugar"
)

// FSMContext is populated by the FSM machinery and passed to Deciders.
type FSMContext struct {
	serialization Serialization
	swf.WorkflowType
	swf.WorkflowExecution
	eventCorrelator *EventCorrelator
	State           string
	stateData       interface{}
	stateVersion    uint64
}

// NewFSMContext constructs an FSMContext.
func NewFSMContext(
	serialization Serialization,
	wfType swf.WorkflowType, wfExec swf.WorkflowExecution,
	eventCorrelator *EventCorrelator,
	state string, stateData interface{}, stateVersion uint64) *FSMContext {
	return &FSMContext{
		serialization:     serialization,
		WorkflowType:      wfType,
		WorkflowExecution: wfExec,
		eventCorrelator:   eventCorrelator,
		State:             state,
		stateData:         stateData,
		stateVersion:      stateVersion,
	}
}

func (f *FSMContext) InitialState() string {
	return f.serialization.InitialState()
}

func (f *FSMContext) StateSerializer() StateSerializer {
	return f.serialization.StateSerializer()
}

// ContinueDecider is a helper func to easily create a ContinueOutcome.
func (f *FSMContext) ContinueDecider(data interface{}, decisions []*swf.Decision) Outcome {
	return Outcome{
		State:     "",
		Data:      data,
		Decisions: decisions,
	}
}

// Stay is a helper func to easily create a StayOutcome.
func (f *FSMContext) Stay(data interface{}, decisions []*swf.Decision) Outcome {
	return Outcome{
		State:     f.State,
		Data:      data,
		Decisions: decisions,
	}
}

// Goto is a helper func to easily create a TransitionOutcome.
func (f *FSMContext) Goto(state string, data interface{}, decisions []*swf.Decision) Outcome {
	return Outcome{
		State:     state,
		Data:      data,
		Decisions: decisions,
	}
}

func (f *FSMContext) Pass() Outcome {
	return Outcome{
		State:     "",
		Data:      f.stateData,
		Decisions: []*swf.Decision{},
	}
}

// CompleteWorkflow is a helper func to easily create a CompleteOutcome that sends a CompleteWorkflow decision.
func (f *FSMContext) CompleteWorkflow(data interface{}, decisions ...*swf.Decision) Outcome {
	if len(decisions) == 0 || *decisions[len(decisions)-1].DecisionType != swf.DecisionTypeCompleteWorkflowExecution {
		decisions = append(decisions, f.CompleteWorkflowDecision(data))
	}
	return Outcome{
		State:     CompleteState,
		Data:      data,
		Decisions: decisions,
	}
}

// ContinueWorkflow is a helper func to easily create a CompleteOutcome that sends a ContinueWorklfow decision.
func (f *FSMContext) ContinueWorkflow(data interface{}, decisions ...*swf.Decision) Outcome {
	if len(decisions) == 0 || *decisions[len(decisions)-1].DecisionType != swf.DecisionTypeContinueAsNewWorkflowExecution {
		decisions = append(decisions, f.ContinueWorkflowDecision(f.State, data))
	}
	return Outcome{
		State:     CompleteState,
		Data:      data,
		Decisions: decisions,
	}
}

// CancelWorkflow is a helper func to easily create a CompleteOutcome that sends a CancelWorklfow decision.
func (f *FSMContext) CancelWorkflow(data interface{}, details *string) Outcome {
	d := &swf.Decision{
		DecisionType: S(swf.DecisionTypeCancelWorkflowExecution),
		CancelWorkflowExecutionDecisionAttributes: &swf.CancelWorkflowExecutionDecisionAttributes{
			Details: details,
		},
	}
	return Outcome{
		State:     CanceledState,
		Data:      data,
		Decisions: f.Decision(d),
	}
}

// FailWorkflow is a helper func to easily create a FailOutcome that sends a FailWorklfow decision.
func (f *FSMContext) FailWorkflow(data interface{}, details *string) Outcome {
	d := &swf.Decision{
		DecisionType: S(swf.DecisionTypeFailWorkflowExecution),
		FailWorkflowExecutionDecisionAttributes: &swf.FailWorkflowExecutionDecisionAttributes{
			Details: details,
		},
	}
	return Outcome{
		State:     FailedState,
		Data:      data,
		Decisions: f.Decision(d),
	}
}

// Decide executes a decider making sure that Activity tasks are being tracked.
func (f *FSMContext) Decide(h *swf.HistoryEvent, data interface{}, decider Decider) Outcome {
	outcome := decider(f, h, data)
	f.eventCorrelator.Track(h)
	return outcome
}

// EventData will extract a payload from the given HistoryEvent and unmarshall it into the given struct.
func (f *FSMContext) EventData(h *swf.HistoryEvent, data interface{}) {
	f.serialization.EventData(h, data)
}

// ActivityInfo will find information for ActivityTasks being tracked. It can only be used when handling events related to ActivityTasks.
// ActivityTasks are automatically tracked after a EventTypeActivityTaskScheduled event.
// When there is no pending activity related to the event, nil is returned.
func (f *FSMContext) ActivityInfo(h *swf.HistoryEvent) *ActivityInfo {
	return f.eventCorrelator.ActivityInfo(h)
}

// ActivitiesInfo will return a map of scheduledId -> ActivityInfo for all in-flight activities in the workflow.
func (f *FSMContext) ActivitiesInfo() map[string]*ActivityInfo {
	return f.eventCorrelator.Activities
}

// SignalInfo will find information for ActivityTasks being tracked. It can only be used when handling events related to ActivityTasks.
// ActivityTasks are automatically tracked after a EventTypeActivityTaskScheduled event.
// When there is no pending activity related to the event, nil is returned.
func (f *FSMContext) SignalInfo(h *swf.HistoryEvent) *SignalInfo {
	return f.eventCorrelator.SignalInfo(h)
}

// SignalsInfo will return a map of scheduledId -> ActivityInfo for all in-flight activities in the workflow.
func (f *FSMContext) SignalsInfo() map[string]*SignalInfo {
	return f.eventCorrelator.Signals
}

// Serialize will use the current fsm's Serializer to serialize the given struct. It will panic on errors, which is ok in the context of a Decider.
// If you want to handle errors, use Serializer().Serialize(...) instead.
func (f *FSMContext) Serialize(data interface{}) string {
	return f.serialization.Serialize(data)
}

// Serializer returns the current fsm's Serializer.
func (f *FSMContext) Serializer() StateSerializer {
	return f.serialization.StateSerializer()
}

// Deserialize will use the current fsm' Serializer to deserialize the given string into the given struct. It will panic on errors, which is ok in the context of a Decider.
// If you want to handle errors, use Serializer().Deserialize(...) instead.
func (f *FSMContext) Deserialize(serialized string, data interface{}) {
	f.serialization.Deserialize(serialized, data)
}

// EmptyDecisions is a helper to give you an empty Decision slice.
func (f *FSMContext) EmptyDecisions() []*swf.Decision {
	return make([]*swf.Decision, 0)
}

// EmptyDecisions is a helper to give you an empty Decision slice.
func (f *FSMContext) Decision(d *swf.Decision) []*swf.Decision {
	return append(f.EmptyDecisions(), d)
}

func (f *FSMContext) Correlator() *EventCorrelator {
	return f.eventCorrelator
}

func (f *FSMContext) Attempts(h *swf.HistoryEvent) int {
	return f.eventCorrelator.Attempts(h)
}

// ContinueWorkflowDecision will build a ContinueAsNewWorkflow decision that has the expected SerializedState marshalled to json as its input.
// This decision should be used when it is appropriate to Continue your workflow.
// You are unable to ContinueAsNew a workflow that has running activites, so you should assure there are none running before using this.
// As such there is no need to copy over the ActivityCorrelator.
// If the FSM Data Struct is Taggable, its tags will be used on the Continue Decisions
func (f *FSMContext) ContinueWorkflowDecision(continuedState string, data interface{}) *swf.Decision {
	return &swf.Decision{
		DecisionType: aws.String(swf.DecisionTypeContinueAsNewWorkflowExecution),
		ContinueAsNewWorkflowExecutionDecisionAttributes: &swf.ContinueAsNewWorkflowExecutionDecisionAttributes{
			Input: aws.String(f.Serialize(SerializedState{
				StateName:    continuedState,
				StateData:    f.Serialize(data),
				StateVersion: f.stateVersion,
			},
			)),
			TagList: GetTagsIfTaggable(data),
		},
	}
}

// CompleteWorkflowDecision will build a CompleteWorkflowExecutionDecision decision that has the expected SerializedState marshalled to json as its result.
// This decision should be used when it is appropriate to Complete your workflow.
func (f *FSMContext) CompleteWorkflowDecision(data interface{}) *swf.Decision {
	return &swf.Decision{
		DecisionType: aws.String(swf.DecisionTypeCompleteWorkflowExecution),
		CompleteWorkflowExecutionDecisionAttributes: &swf.CompleteWorkflowExecutionDecisionAttributes{
			Result: aws.String(f.Serialize(data)),
		},
	}
}
