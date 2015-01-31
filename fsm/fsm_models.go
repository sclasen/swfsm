package fsm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"

	"code.google.com/p/goprotobuf/proto"
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/kinesis"
	"github.com/awslabs/aws-sdk-go/gen/swf"
)

// constants used as marker names or signal names
const (
	StateMarker       = "FSM.State"
	CorrelatorMarker  = "FSM.Correlator"
	ErrorSignal       = "FSM.Error"
	SystemErrorSignal = "FSM.SystemError"
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

// Outcome represents the minimum data needed to be returned by a Decider.
type Outcome interface {
	// Data returns the data for this Outcome.
	Data() interface{}
	// Decisions returns the list of Decisions for this Outcome.
	Decisions() []swf.Decision
	// State returns the state to transition to. An empty string means no
	// transition.
	State() string
}

// Pass is nil, a sentinel value to represent 'no outcome'
var Pass Outcome

// ContinueOutcome is an Outcome used to contribute decisions and data to a
// composed Decider.
type ContinueOutcome struct {
	data      interface{}
	decisions []swf.Decision
}

// Data returns the data for this Outcome.
func (s ContinueOutcome) Data() interface{} { return s.data }

// Decisions returns the list of Decisions for this Outcome.
func (s ContinueOutcome) Decisions() []swf.Decision { return s.decisions }

// State returns the next state for the ContinueOutcome, which is always empty.
func (s ContinueOutcome) State() string { return "" }

// TransitionOutcome is an Outcome in which the FSM will transtion to a new state.
type TransitionOutcome struct {
	data      interface{}
	state     string
	decisions []swf.Decision
}

// Data returns the data for this Outcome.
func (t TransitionOutcome) Data() interface{} { return t.data }

// Decisions returns the list of Decisions for this Outcome.
func (t TransitionOutcome) Decisions() []swf.Decision { return t.decisions }

// State returns the next state for this TransitionOutcome.
func (t TransitionOutcome) State() string { return t.state }

// StayOutcome is an Outcome in which the FSM will remain in the same state.
type StayOutcome struct {
	data      interface{}
	decisions []swf.Decision
}

// Data returns the data for this Outcome.
func (s StayOutcome) Data() interface{} { return s.data }

// Decisions returns the list of Decisions for this Outcome.
func (s StayOutcome) Decisions() []swf.Decision { return s.decisions }

// State returns the next state for the StayOutcome, which is always empty.
func (s StayOutcome) State() string { return "" }

// CompleteOutcome will send a CompleteWorkflowExecutionDecision, and transition to a 'managed' fsm state
// that will respond to any further events by attempting to Complete the workflow. This can happen only if there were
// unhandled decisions
type CompleteOutcome struct {
	data      interface{}
	decisions []swf.Decision
}

// Data returns the data for this Outcome.
func (t CompleteOutcome) Data() interface{} { return t.data }

// Decisions returns the list of Decisions for this Outcome.
func (t CompleteOutcome) Decisions() []swf.Decision { return t.decisions }

// State returns the next state for the CompleteOutcome, which is always
// "complete".
func (t CompleteOutcome) State() string { return CompleteState }

// ErrorOutcome can be used to purposefully put the workflow into an error state.
type ErrorOutcome struct {
	state     string
	data      interface{}
	decisions []swf.Decision
}

// Data returns the data for this Outcome.
func (e ErrorOutcome) Data() interface{} { return e.data }

// Decisions returns the list of Decisions for this Outcome.
func (e ErrorOutcome) Decisions() []swf.Decision { return e.decisions }

// State returns the next state for the ErrorOutcome, which is always "error".
func (e ErrorOutcome) State() string { return "error" }

type intermediateOutcome struct {
	stateVersion uint64
	state        string
	data         interface{}
	decisions    []swf.Decision
}

//KinesisReplicator lets you customize the retry logic around Replicating State to Kinesis.
type KinesisReplicator func(fsm, workflowID string, put func() (*kinesis.PutRecordOutput, error)) (*kinesis.PutRecordOutput, error)

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
	ErrorEventID        int64       `json:"errorEventIds"`
	UnprocessedEventIDs []int64     `json:"unprocessedEventIds"`
	StateName           string      `json:"stateName"`
	StateData           interface{} `json:"stateData"`
	Error               interface{} `json:"error"`
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

// FSMSerializer is the contract for de/serializing state inside an FSM, typically implemented by the FSM itself
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
	state string, stateData interface{}, stateVersion uint64,
) *FSMContext {
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

// ContinueDecision is a helper func to easily create a ContinueOutcome.
func (f *FSMContext) ContinueDecision(data interface{}, decisions []swf.Decision) Outcome {
	return ContinueOutcome{
		data:      data,
		decisions: decisions,
	}
}

// Stay is a helper func to easily create a StayOutcome.
func (f *FSMContext) Stay(data interface{}, decisions []swf.Decision) Outcome {
	return StayOutcome{
		data:      data,
		decisions: decisions,
	}
}

// Goto is a helper func to easily create a TransitionOutcome.
func (f *FSMContext) Goto(state string, data interface{}, decisions []swf.Decision) Outcome {
	return TransitionOutcome{
		state:     state,
		data:      data,
		decisions: decisions,
	}
}

// Complete is a helper func to easily create a CompleteOutcome.
func (f *FSMContext) Complete(data interface{}, decisions ...swf.Decision) Outcome {
	final := append(decisions, f.CompletionDecision(data))
	return CompleteOutcome{
		data:      data,
		decisions: final,
	}
}

// Goto is a helper func to easily create an ErrorOutcome.
func (f *FSMContext) Error(data interface{}, decisions []swf.Decision) Outcome {
	return ErrorOutcome{
		state:     ErrorState,
		data:      data,
		decisions: decisions,
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
	return f.eventCorrelator.ActivityType(h)
}

// ActivitiesInfo will return a map of activityId -> ActivityInfo for all in-flight activities in the workflow.
func (f *FSMContext) ActivitiesInfo() map[string]*ActivityInfo {
	return f.eventCorrelator.Activities
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
func (f *FSMContext) ContinueWorkflowDecision(continuedState string) swf.Decision {
	return swf.Decision{
		DecisionType: aws.String(swf.DecisionTypeContinueAsNewWorkflowExecution),
		ContinueAsNewWorkflowExecutionDecisionAttributes: &swf.ContinueAsNewWorkflowExecutionDecisionAttributes{
			Input: aws.String(f.Serialize(SerializedState{
				StateName:    continuedState,
				StateData:    f.Serialize(f.stateData),
				StateVersion: f.stateVersion,
			},
			)),
		},
	}
}

// CompletionDecision will build a CompleteWorkflowExecutionDecision decision that has the expected SerializedState marshalled to json as its result.
// This decision should be used when it is appropriate to Complete your workflow.
func (f *FSMContext) CompletionDecision(data interface{}) swf.Decision {
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
