package fsm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"

	"code.google.com/p/goprotobuf/proto"
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/swf"
)

// constants used as marker names or signal names
const (
	StateMarker       = "FSM.State"
	CorrelatorMarker  = "FSM.Correlator"
	ErrorSignal       = "FSM.Error"
	SystemErrorSignal = "FSM.SystemError"
	RepiarStateSignal = "FSM.RepairState"
	ContinueTimer     = "FSM.ContinueWorkflow"
	ContinueSignal    = "FSM.ContinueWorkflow"
	CompleteState     = "complete"
	ErrorState        = "error"
)

// Decider decides an Outcome based on an event and the current data for an
// FSM. You can assert the interface{} parameter that is passed to the Decider
// as the type of the DataType field in the FSM. Alternatively, you can use the
// TypedDecider to avoid having to do the assertion.
type Decider func(*FSMContext, swf.HistoryEvent, interface{}) Outcome

type Outcome struct {
	State     string
	Data      interface{}
	Decisions []swf.Decision
	Continue  bool
}

// FSMState defines the behavior of one state of an FSM
type FSMState struct {
	// Name is the name of the state. When returning an Outcome, the NextState should match the Name of an FSMState in your FSM.
	Name string
	// Decider decides an Outcome given the current state, data, and an event.
	Decider Decider
}

// ReplicationData is the part of SerializedState that will be replicated onto Kinesis streams.
type ReplicationData struct {
	StateVersion uint64 `json:"stateVersion"`
	StateName    string `json:"stateName"`
	StateData    string `json:"stateData"`
}

// SerializedDecisionError is a wrapper struct that allows serializing the context in which an error in a Decider occurred
// into a WorkflowSignaledEvent in the workflow history.
type SerializedDecisionError struct {
	ErrorEvent swf.HistoryEvent `json:"errorEvent"`
	Error      interface{}      `json:"error"`
}

// SerializedSystemError is a wrapper struct that allows serializing the context in which an error internal to FSM processing has occurred
// into a WorkflowSignaledEvent in the workflow history. These errors are generally in finding the current state and data for a workflow, or
// in serializing and deserializing said state.
type SerializedSystemError struct {
	ErrorType           string      `json:"errorType"`
	Error               interface{} `json:"error"`
	UnprocessedEventIDs []int64     `json:"unprocessedEventIds"`
}

// StateSerializer defines the interface for serializing state to and deserializing state from the workflow history.
type StateSerializer interface {
	Serialize(state interface{}) (string, error)
	Deserialize(serialized string, state interface{}) error
}

// JSONStateSerializer is a StateSerializer that uses go json serialization.
type JSONStateSerializer struct{}

// Serialize serializes the given struct to a json string.
func (j JSONStateSerializer) Serialize(state interface{}) (string, error) {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(state); err != nil {
		return "", err
	}
	return b.String(), nil
}

// Deserialize unmarshalls the given (json) string into the given struct
func (j JSONStateSerializer) Deserialize(serialized string, state interface{}) error {
	err := json.NewDecoder(strings.NewReader(serialized)).Decode(state)
	return err
}

// ProtobufStateSerializer is a StateSerializer that uses base64 encoded protobufs.
type ProtobufStateSerializer struct{}

// Serialize serializes the given struct into bytes with protobuf, then base64 encodes it.  The struct passed to Serialize must satisfy proto.Message.
func (p ProtobufStateSerializer) Serialize(state interface{}) (string, error) {
	bin, err := proto.Marshal(state.(proto.Message))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bin), nil
}

// Deserialize base64 decodes the given string then unmarshalls the bytes into the struct using protobuf. The struct passed to Deserialize must satisfy proto.Message.
func (p ProtobufStateSerializer) Deserialize(serialized string, state interface{}) error {
	bin, err := base64.StdEncoding.DecodeString(serialized)
	if err != nil {
		return err
	}
	err = proto.Unmarshal(bin, state.(proto.Message))

	return err
}

// Serialization is the contract for de/serializing state inside an FSM, typically implemented by the FSM itself
// but serves to break the circular dep between FSMContext and FSM.
type Serialization interface {
	EventData(h swf.HistoryEvent, data interface{})
	Serialize(data interface{}) string
	StateSerializer() StateSerializer
	Deserialize(serialized string, data interface{})
	InitialState() string
}

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

// ContinueDecider is a helper func to easily create a ContinueOutcome.
func (f *FSMContext) ContinueDecider(data interface{}, decisions []swf.Decision) Outcome {
	return Outcome{
		State:     f.State,
		Data:      data,
		Decisions: decisions,
		Continue:  true,
	}
}

// Stay is a helper func to easily create a StayOutcome.
func (f *FSMContext) Stay(data interface{}, decisions []swf.Decision) Outcome {
	return Outcome{
		State:     f.State,
		Data:      data,
		Decisions: decisions,
	}
}

// Goto is a helper func to easily create a TransitionOutcome.
func (f *FSMContext) Goto(state string, data interface{}, decisions []swf.Decision) Outcome {
	return Outcome{
		State:     state,
		Data:      data,
		Decisions: decisions,
	}
}

func (f *FSMContext) Pass() Outcome {
	return Outcome{
		State:     f.State,
		Data:      f.stateData,
		Decisions: []swf.Decision{},
		Continue:  true,
	}
}

// CompleteWorkflow is a helper func to easily create a CompleteOutcome that sends a CompleteWorkflow decision.
func (f *FSMContext) CompleteWorkflow(data interface{}, decisions ...swf.Decision) Outcome {
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
func (f *FSMContext) ContinueWorkflow(data interface{}, decisions ...swf.Decision) Outcome {
	if len(decisions) == 0 || *decisions[len(decisions)-1].DecisionType != swf.DecisionTypeContinueAsNewWorkflowExecution {
		decisions = append(decisions, f.ContinueWorkflowDecision(f.State, data))
	}
	return Outcome{
		State:     CompleteState,
		Data:      data,
		Decisions: decisions,
	}
}

// Decide executes a decider making sure that Activity tasks are being tracked.
func (f *FSMContext) Decide(h swf.HistoryEvent, data interface{}, decider Decider) Outcome {
	outcome := decider(f, h, data)
	f.eventCorrelator.Track(h)
	return outcome
}

// EventData will extract a payload from the given HistoryEvent and unmarshall it into the given struct.
func (f *FSMContext) EventData(h swf.HistoryEvent, data interface{}) {
	f.serialization.EventData(h, data)
}

// ActivityInfo will find information for ActivityTasks being tracked. It can only be used when handling events related to ActivityTasks.
// ActivityTasks are automatically tracked after a EventTypeActivityTaskScheduled event.
// When there is no pending activity related to the event, nil is returned.
func (f *FSMContext) ActivityInfo(h swf.HistoryEvent) *ActivityInfo {
	return f.eventCorrelator.ActivityInfo(h)
}

// ActivitiesInfo will return a map of scheduledID -> ActivityInfo for all in-flight activities in the workflow.
func (f *FSMContext) ActivitiesInfo() map[string]*ActivityInfo {
	return f.eventCorrelator.Activities
}

// SignalInfo will find information for ActivityTasks being tracked. It can only be used when handling events related to ActivityTasks.
// ActivityTasks are automatically tracked after a EventTypeActivityTaskScheduled event.
// When there is no pending activity related to the event, nil is returned.
func (f *FSMContext) SignalInfo(h swf.HistoryEvent) *SignalInfo {
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
func (f *FSMContext) EmptyDecisions() []swf.Decision {
	return make([]swf.Decision, 0)
}

// ContinueWorkflowDecision will build a ContinueAsNewWorkflow decision that has the expected SerializedState marshalled to json as its input.
// This decision should be used when it is appropriate to Continue your workflow.
// You are unable to ContinueAsNew a workflow that has running activites, so you should assure there are none running before using this.
// As such there is no need to copy over the ActivityCorrelator.
func (f *FSMContext) ContinueWorkflowDecision(continuedState string, data interface{}) swf.Decision {
	return swf.Decision{
		DecisionType: aws.String(swf.DecisionTypeContinueAsNewWorkflowExecution),
		ContinueAsNewWorkflowExecutionDecisionAttributes: &swf.ContinueAsNewWorkflowExecutionDecisionAttributes{
			Input: aws.String(f.Serialize(SerializedState{
				StateName:    continuedState,
				StateData:    f.Serialize(data),
				StateVersion: f.stateVersion,
			},
			)),
		},
	}
}

// CompleteWorkflowDecision will build a CompleteWorkflowExecutionDecision decision that has the expected SerializedState marshalled to json as its result.
// This decision should be used when it is appropriate to Complete your workflow.
func (f *FSMContext) CompleteWorkflowDecision(data interface{}) swf.Decision {
	return swf.Decision{
		DecisionType: aws.String(swf.DecisionTypeCompleteWorkflowExecution),
		CompleteWorkflowExecutionDecisionAttributes: &swf.CompleteWorkflowExecutionDecisionAttributes{
			Result: aws.String(f.Serialize(data)),
		},
	}
}

// SerializedState is a wrapper struct that allows serializing the current state and current data for the FSM in
// a MarkerRecorded event in the workflow history. We also maintain an epoch, which counts the number of times a workflow has
// been continued, and the StartedId of the DecisionTask that generated this state.  The epoch + the id provide a total ordering
// of state over the lifetime of different runs of a workflow.
type SerializedState struct {
	StateVersion uint64 `json:"stateVersion"`
	StateName    string `json:"stateName"`
	StateData    string `json:"stateData"`
}
