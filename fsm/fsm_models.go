package fsm

import (
	"bytes"
	"encoding/json"
	"strings"

	"encoding/gob"

	"fmt"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/swf"
)

// constants used as marker names or signal names
const (
	StateMarker       = "FSM.State"
	CorrelatorMarker  = "FSM.Correlator"
	ErrorMarker       = "FSM.Error"
	RepiarStateSignal = "FSM.RepairState"
	ContinueTimer     = "FSM.ContinueWorkflow"
	ContinueSignal    = "FSM.ContinueWorkflow"
	CompleteState     = "complete"
	CanceledState     = "canceled"
	FailedState       = "failed"
	ErrorState        = "error"
	//the FSM was not configured with a state named in an outcome.
	FSMErrorMissingState = "ErrorMissingFsmState"
	//the FSM encountered an erryor while serializaing stateData
	FSMErrorStateSerialization = "ErrorStateSerialization"
	//the FSM encountered an erryor while deserializaing stateData
	FSMErrorStateDeserialization = "ErrorStateDeserialization"
	//the FSM encountered an erryor while deserializaing stateData
	FSMErrorCorrelationDeserialization = "ErrorCorrelationDeserialization"
	//Signal sent when a Long Lived Worker Start()
	ActivityStartedSignal = "FSM.ActivityStarted"
	//Signal send when long Lived worker sends an update from Work()
	ActivityUpdatedSignal = "FSM.ActivityUpdated"
)

// Decider decides an Outcome based on an event and the current data for an
// FSM. You can assert the interface{} parameter that is passed to the Decider
// as the type of the DataType field in the FSM. Alternatively, you can use
// TypedFuncs to create a typed decider to avoid having to do the assertion.
type Decider func(*FSMContext, *swf.HistoryEvent, interface{}) Outcome

//Outcome is the result of a Decider processing a HistoryEvent
type Outcome struct {
	//State is the desired next state in the FSM. the empty string ("") is a signal that you wish decision processing to continue
	//if the FSM machinery recieves the empty string as the state of a final outcome, it will substitute the current state.
	State     string
	Data      interface{}
	Decisions []*swf.Decision
}

// FSMState defines the behavior of one state of an FSM
type FSMState struct {
	// Name is the name of the state. When returning an Outcome, the NextState should match the Name of an FSMState in your FSM.
	Name string
	// Decider decides an Outcome given the current state, data, and an event.
	Decider Decider
}

//DecisionErrorHandler is the error handling contract for panics that occur in Deciders.
//If your DecisionErrorHandler does not return a non nil Outcome, any further attempt to process the decisionTask is abandoned and the task will time out.
type DecisionErrorHandler func(ctx *FSMContext, event *swf.HistoryEvent, stateBeforeEvent interface{}, stateAfterError interface{}, err error) (*Outcome, error)

//FSMErrorHandler is the error handling contract for errors in the FSM machinery itself.
//These are generally a misconfiguration of your FSM or mismatch between struct and serialized form and cant be resolved without config/code changes
//the paramaters to each method provide all availabe info at the time of the error so you can diagnose issues.
//Note that this is a diagnostic interface that basically leaks implementation details, and as such may change from release to release.
type FSMErrorReporter interface {
	ErrorFindingStateData(decisionTask *swf.PollForDecisionTaskOutput, err error)
	ErrorFindingCorrelator(decisionTask *swf.PollForDecisionTaskOutput, err error)
	ErrorMissingFSMState(decisionTask *swf.PollForDecisionTaskOutput, outcome Outcome)
	ErrorDeserializingStateData(decisionTask *swf.PollForDecisionTaskOutput, serializedStateData string, err error)
	ErrorSerializingStateData(decisionTask *swf.PollForDecisionTaskOutput, outcome Outcome, eventCorrelator EventCorrelator, err error)
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

// Serialization is the contract for de/serializing state inside an FSM, typically implemented by the FSM itself
// but serves to break the circular dep between FSMContext and FSM.
type Serialization interface {
	EventData(h *swf.HistoryEvent, data interface{})
	Serialize(data interface{}) string
	StateSerializer() StateSerializer
	Deserialize(serialized string, data interface{})
	InitialState() string
}

// FSM Data types that implement this interface will have the resulting tags used by
// FSMClient when starting workflows and by the FSMContext when calling ContinueWorkflow()
// it is []*string since thats what SWF api takes atm.
type Taggable interface {
	Tags() []*string
}

func GetTagsIfTaggable(data interface{}) []*string {
	var tags []*string
	if t, ok := data.(Taggable); ok {
		tags = t.Tags()
	}
	return tags
}

// SerializedState is a wrapper struct that allows serializing the current state and current data for the FSM in
// a MarkerRecorded event in the workflow history. We also maintain an epoch, which counts the number of times a workflow has
// been continued, and the StartedId of the DecisionTask that generated this state.  The epoch + the id provide a total ordering
// of state over the lifetime of different runs of a workflow.
type SerializedState struct {
	StateVersion uint64 `json:"stateVersion"`
	StateName    string `json:"stateName"`
	StateData    string `json:"stateData"`
	WorkflowId   string `json:"workflowId"`
}

//ErrorState is used as the input to a marker that signifies that the workflow is in an error state.
type SerializedErrorState struct {
	Details                    string
	EarliestUnprocessedEventId int64
	LatestUnprocessedEventId   int64
	ErrorEvent                 *swf.HistoryEvent
}

//Payload of Signals ActivityStartedSignal and ActivityUpdatedSignal
type SerializedActivityState struct {
	ActivityId string
	Input      *string
}

// StartFSMWorkflowInput should be used to construct the input for any StartWorkflowExecutionRequests.
// This panics on errors cause really this should never err.
func StartFSMWorkflowInput(serializer Serialization, data interface{}) *string {
	ss := new(SerializedState)
	stateData := serializer.Serialize(data)
	ss.StateData = stateData
	serialized := serializer.Serialize(ss)
	return aws.String(serialized)
}

//Stasher is used to take snapshots of StateData between each event so that we can have shap
type Stasher struct {
	dataType interface{}
}

func NewStasher(dataType interface{}) *Stasher {
	gob.Register(dataType)
	return &Stasher{
		dataType: dataType,
	}
}

func (s *Stasher) Stash(data interface{}) *bytes.Buffer {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(data)
	if err != nil {
		panic(fmt.Sprintf("at=stash type=%s error=%q", reflect.TypeOf(s.dataType), err))
	}
	return buf
}

func (s *Stasher) Unstash(stashed *bytes.Buffer, into interface{}) {
	dec := gob.NewDecoder(stashed)
	err := dec.Decode(into)
	if err != nil {
		panic(fmt.Sprintf("at=unstash type=%s error=%q", reflect.TypeOf(s.dataType), err))
	}
}
