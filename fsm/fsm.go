package fsm

import (
	"errors"
	"fmt"
	"log"
	"reflect"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/swf"
	"github.com/sclasen/swfsm/poller"
	. "github.com/sclasen/swfsm/sugar"
)

//SWFOps is the subset of swf.SWF ops required by the fsm package
type SWFOps interface {
	PollForDecisionTask(*swf.PollForDecisionTaskInput) (*swf.DecisionTask, error)
	PollForActivityTask(*swf.PollForActivityTaskInput) (*swf.ActivityTask, error)
	RespondDecisionTaskCompleted(*swf.RespondDecisionTaskCompletedInput) error
}

// FSM models the decision handling logic a workflow in SWF
type FSM struct {
	//Name of the fsm. Used when emitting logs. Should probably be set to the name of the workflow associated with the fsm.
	Name string
	// Domain of the workflow associated with the FSM.
	Domain string
	// TaskList that the underlying poller will poll for decision tasks.
	TaskList string
	// Identity used in PollForDecisionTaskRequests, can be empty.
	Identity string
	// Client used to make SWF api requests.
	SWF SWFOps
	// Strategy for replication of state to the systems the build the Query side model.
	ReplicationHandler ReplicationHandler
	// DataType of the data struct associated with this FSM.
	// The data is automatically peristed to and loaded from workflow history by the FSM.
	DataType interface{}
	// Serializer used to serialize/deserialise fsm state data to/from workflow history.
	Serializer StateSerializer
	// Serializer used to serialize/deserialise in json the fsm managed marker recorded events to/from workflow history.
	systemSerializer StateSerializer
	//PollerShutdownManager is used when the FSM is managing the polling
	ShutdownManager *poller.ShutdownManager
	//DecisionTaskDispatcher determines the concurrency strategy for processing tasks in your fsm
	DecisionTaskDispatcher DecisionTaskDispatcher
	states                 map[string]*FSMState
	initialState           *FSMState
	completeState          *FSMState
	stop                   chan bool
	stopAck                chan bool
	allowPanics            bool //makes testing easier
}

// StateSerializer is the implementation of FSMSerializer.StateSerializer()
func (f *FSM) StateSerializer() StateSerializer {
	return f.Serializer
}

// AddInitialState adds a state to the FSM and uses it as the initial state when a workflow execution is started.
func (f *FSM) AddInitialState(state *FSMState) {
	f.AddState(state)
	f.initialState = state
}

// InitialState is the implementation of FSMSerializer.InitialState()
func (f *FSM) InitialState() string {
	return f.initialState.Name
}

// AddState adds a state to the FSM.
func (f *FSM) AddState(state *FSMState) {
	if f.states == nil {
		f.states = make(map[string]*FSMState)
	}
	f.states[state.Name] = state
}

// AddCompleteState adds a state to the FSM and uses it as the final state of a workflow.
// it will only receive events if you returned FSMContext.Complete(...) and the workflow was unable to complete.
func (f *FSM) AddCompleteState(state *FSMState) {
	f.AddState(state)
	f.completeState = state
}

// DefaultCompleteState is the complete state used in an FSM if one has not been set.
// It simply responds with a CompleteDecision which attempts to Complete the workflow.
// This state will only get events if you previously attempted to complete the workflow and it failed.
func (f *FSM) DefaultCompleteState() *FSMState {
	return &FSMState{
		Name: CompleteState,
		Decider: func(fsm *FSMContext, h swf.HistoryEvent, data interface{}) Outcome {
			f.log("state=complete at=attempt-completion event=%s", h)
			return fsm.CompleteWorkflow(data)
		},
	}
}

// Init initializaed any optional, unspecified values such as the error state, stop channel, serializer, PollerShutdownManager.
// it gets called by Start(), so you should only call this if you are manually managing polling for tasks, and calling Tick yourself.
func (f *FSM) Init() {
	if f.initialState == nil {
		panic("No Initial State Defined For FSM")
	}

	if f.completeState == nil {
		f.AddCompleteState(f.DefaultCompleteState())
	}

	if f.stop == nil {
		f.stop = make(chan bool, 1)
	}

	if f.stopAck == nil {
		f.stopAck = make(chan bool, 1)
	}

	if f.Serializer == nil {
		f.log("action=start at=no-serializer defaulting-to=JSONSerializer")
		f.Serializer = &JSONStateSerializer{}
	}

	if f.systemSerializer == nil {
		f.log("action=start at=no-system-serializer defaulting-to=JSONSerializer")
		f.systemSerializer = &JSONStateSerializer{}
	}

	if f.ShutdownManager == nil {
		f.ShutdownManager = poller.NewShutdownManager()
	}

	if f.DecisionTaskDispatcher == nil {
		f.DecisionTaskDispatcher = &CallingGoroutineDispatcher{}
	}

}

// Start begins processing DecisionTasks with the FSM. It creates a DecisionTaskPoller and spawns a goroutine that continues polling until Stop() is called and any in-flight polls have completed.
// If you wish to manage polling and calling Tick() yourself, you dont need to start the FSM, just call Init().
func (f *FSM) Start() {
	f.Init()
	poller := poller.NewDecisionTaskPoller(f.SWF, f.Domain, f.Identity, f.TaskList)
	go poller.PollUntilShutdownBy(f.ShutdownManager, fmt.Sprintf("%s-poller", f.Name), f.dispatchTask)
}

func (f *FSM) dispatchTask(decisionTask *swf.DecisionTask) {
	f.DecisionTaskDispatcher.DispatchTask(decisionTask, f.handleDecisionTask)
}

func (f *FSM) handleDecisionTask(decisionTask *swf.DecisionTask) {
	context, decisions, state := f.Tick(decisionTask)
	complete := &swf.RespondDecisionTaskCompletedInput{
		Decisions: decisions,
		TaskToken: decisionTask.TaskToken,
	}

	complete.ExecutionContext = aws.String(state.StateName)

	if err := f.SWF.RespondDecisionTaskCompleted(complete); err != nil {
		f.log("action=tick at=decide-request-failed error=%q", err.Error())
		return
	}

	if f.ReplicationHandler != nil {
		repErr := f.ReplicationHandler(context, decisionTask, complete, state)
		if repErr != nil {
			f.log("action=tick at=replication-handler-failed error=%q", repErr.Error())
		}
	}

}

func (f *FSM) stateFromDecisions(decisions []swf.Decision) string {
	for _, d := range decisions {
		if *d.DecisionType == swf.DecisionTypeRecordMarker && *d.RecordMarkerDecisionAttributes.MarkerName == StateMarker {
			return *d.RecordMarkerDecisionAttributes.Details
		}
	}
	return ""
}

// Serialize uses the FSM.Serializer to serialize data to a string.
// If there is an error in serialization this func will panic, so this should usually only be used inside Deciders
// where the panics are recovered and proper errors are recorded in the workflow.
func (f *FSM) Serialize(data interface{}) string {
	serialized, err := f.Serializer.Serialize(data)
	if err != nil {
		panic(err)
	}
	return serialized
}

// Deserialize uses the FSM.Serializer to deserialize data from a string.
// If there is an error in deserialization this func will panic, so this should usually only be used inside Deciders
// where the panics are recovered and proper errors are recorded in the workflow.
func (f *FSM) Deserialize(serialized string, data interface{}) {
	err := f.Serializer.Deserialize(serialized, data)
	if err != nil {
		panic(err)
	}
	return
}

// Tick is called when the DecisionTaskPoller receives a PollForDecisionTaskResponse in its polling loop.
// On errors, a nil *SerializedState is returned, and an error Outcome is included in the Decision list.
// It is exported to facilitate testing.
func (f *FSM) Tick(decisionTask *swf.DecisionTask) (*FSMContext, []swf.Decision, *SerializedState) {
	lastEvents := f.findLastEvents(*decisionTask.PreviousStartedEventID, decisionTask.Events)
	execution := decisionTask.WorkflowExecution
	outcome := new(intermediateOutcome)
	context := NewFSMContext(f,
		*decisionTask.WorkflowType,
		*decisionTask.WorkflowExecution,
		nil,
		"", nil, uint64(0),
	)

	serializedState, err := f.findSerializedState(decisionTask.Events)
	if err != nil {
		f.log("action=tick at=error=find-serialized-state-failed err=%q", err)
		if f.allowPanics {
			panic(err)
		}
		return context, append(outcome.decisions, f.captureSystemError(execution, "FindSerializedStateError", decisionTask.Events, err)...), nil
	}
	eventCorrelator, err := f.findSerializedEventCorrelator(decisionTask.Events)
	if err != nil {
		f.log("action=tick at=error=find-serialized-event-correlator-failed err=%q", err)
		if f.allowPanics {
			panic(err)
		}
		return context, append(outcome.decisions, f.captureSystemError(execution, "FindSerializedStateError", decisionTask.Events, err)...), nil
	}
	context.eventCorrelator = eventCorrelator

	f.log("action=tick at=find-serialized-state state=%s", serializedState.StateName)

	//todo now on any error on processing an event simply sends an error signal to the workflow
	//decider stack should all handle errors
	//if there was no error processing, we recover the state + data from the marker
	if outcome.data == nil && outcome.state == "" {
		data := reflect.New(reflect.TypeOf(f.DataType)).Interface()
		if err = f.Serializer.Deserialize(serializedState.StateData, data); err != nil {
			f.log("action=tick at=error=deserialize-state-failed err=&s", err)
			if f.allowPanics {
				panic(err)
			}
			return context, append(outcome.decisions, f.captureSystemError(execution, "DeserializeStateError", decisionTask.Events, err)...), nil
		}
		f.log("action=tick at=find-current-data data=%v", data)

		outcome.data = data
		outcome.state = serializedState.StateName
		outcome.stateVersion = serializedState.StateVersion
		context.stateVersion = serializedState.StateVersion
	}

	//iterate through events oldest to newest, calling the decider for the current state.
	//if the outcome changes the state use the right FSMState
	for i := len(lastEvents) - 1; i >= 0; i-- {
		e := lastEvents[i]
		f.log("action=tick at=history id=%d type=%s", e.EventID, e.EventType)
		fsmState, ok := f.states[outcome.state]
		if ok {
			context.State = outcome.state
			context.stateData = outcome.data
			anOutcome, err := f.panicSafeDecide(fsmState, context, e, outcome.data)
			if err != nil {
				f.log("at=error error=decision-execution-error err=%q state=%s next-state=%s", err, fsmState.Name, outcome.state)
				if f.allowPanics {
					panic(err)
				}
				//todo make a synthetic error signal event and make the decider handle
				return context, append(outcome.decisions, f.captureDecisionError(execution, e, err)...), nil
			}
			eventCorrelator.Track(e)
			curr := outcome.state
			f.mergeOutcomes(outcome, anOutcome)
			f.log("action=tick at=decided-event state=%s next-state=%s decisions=%d", curr, outcome.state, len(anOutcome.Decisions()))
		} else {
			f.log("action=tick at=error error=marked-state-not-in-fsm state=%s", outcome.state)
			return context, append(outcome.decisions, f.captureSystemError(execution, "MissingFsmStateError", lastEvents[i:], errors.New(outcome.state))...), nil
		}
	}

	f.log("action=tick at=events-processed next-state=%s decisions=%d", outcome.state, len(outcome.decisions))

	for _, d := range outcome.decisions {
		f.log("action=tick at=decide next-state=%s decision=%s", outcome.state, d.DecisionType)
	}

	final, serializedState, err := f.recordStateMarkers(outcome, eventCorrelator)
	if err != nil {
		f.log("action=tick at=error error=state-serialization-error err=%q error-type=system", err)
		if f.allowPanics {
			panic(err)
		}
		return context, append(outcome.decisions, f.captureSystemError(execution, "StateSerializationError", []swf.HistoryEvent{}, err)...), nil
	}

	return context, final, serializedState
}

func (f *FSM) mergeOutcomes(final *intermediateOutcome, intermediate Outcome) {
	final.decisions = append(final.decisions, intermediate.Decisions()...)
	final.data = intermediate.Data()
	if _, ok := intermediate.(StayOutcome); !ok {
		final.state = intermediate.State()
	}
}

//if the outcome is good good if its an error, we capture the error state above

func (f *FSM) panicSafeDecide(state *FSMState, context *FSMContext, event swf.HistoryEvent, data interface{}) (anOutcome Outcome, anErr error) {
	defer func() {
		if !f.allowPanics {
			if r := recover(); r != nil {
				f.log("at=error error=decide-panic-recovery %v", r)
				if err, ok := r.(error); ok && err != nil {
					anErr = err
				} else {
					anErr = errors.New("panic in decider, null error, capture error state")
				}
			}
		} else {
			log.Printf("at=panic-safe-decide-allowing-panic fsm-allow-panics=%t", f.allowPanics)
		}
	}()
	anOutcome = context.Decide(event, data, state.Decider)
	return
}

func (f *FSM) captureDecisionError(execution *swf.WorkflowExecution, event swf.HistoryEvent, err error) []swf.Decision {
	return f.captureError(ErrorSignal, execution, &SerializedDecisionError{
		ErrorEvent: event,
		Error:      err,
	})
}

func (f *FSM) captureSystemError(execution *swf.WorkflowExecution, errorType string, lastEvents []swf.HistoryEvent, err error) []swf.Decision {
	return f.captureError(SystemErrorSignal, execution, &SerializedSystemError{
		ErrorType:           errorType,
		UnprocessedEventIDs: f.eventIDs(lastEvents),
		Error:               err,
	})
}

func (f *FSM) eventIDs(events []swf.HistoryEvent) []int64 {
	ids := make([]int64, len(events))
	for _, e := range events {
		ids = append(ids, *e.EventID)
	}
	return ids
}

func (f *FSM) captureError(signal string, execution *swf.WorkflowExecution, error interface{}) []swf.Decision {
	decisions := f.EmptyDecisions()
	r, err := f.recordMarker(signal, error)
	if err != nil {
		//really bail
		panic(fmt.Sprintf("giving up, can't even create a RecordMarker decsion: %s", err))
	}
	d := swf.Decision{
		DecisionType: aws.String(swf.DecisionTypeSignalExternalWorkflowExecution),
		SignalExternalWorkflowExecutionDecisionAttributes: &swf.SignalExternalWorkflowExecutionDecisionAttributes{
			WorkflowID: execution.WorkflowID,
			RunID:      execution.RunID,
			SignalName: aws.String(signal),
			Input:      r.RecordMarkerDecisionAttributes.Details,
		},
	}
	return append(decisions, d)
}

// EventData works in combination with the FSM.Serializer to provide
// deserialization of data sent in a HistoryEvent. It is sugar around extracting the event payload from the proper
// field of the proper Attributes struct on the HistoryEvent
func (f *FSM) EventData(event swf.HistoryEvent, eventData interface{}) {

	if eventData != nil {
		var serialized string
		switch *event.EventType {
		case swf.EventTypeActivityTaskCompleted:
			serialized = *event.ActivityTaskCompletedEventAttributes.Result
		case swf.EventTypeChildWorkflowExecutionFailed:
			serialized = *event.ActivityTaskFailedEventAttributes.Details
		case swf.EventTypeWorkflowExecutionCompleted:
			serialized = *event.WorkflowExecutionCompletedEventAttributes.Result
		case swf.EventTypeChildWorkflowExecutionCompleted:
			serialized = *event.ChildWorkflowExecutionCompletedEventAttributes.Result
		case swf.EventTypeWorkflowExecutionSignaled:
			serialized = *event.WorkflowExecutionSignaledEventAttributes.Input
		case swf.EventTypeWorkflowExecutionStarted:
			serialized = *event.WorkflowExecutionStartedEventAttributes.Input
		case swf.EventTypeWorkflowExecutionContinuedAsNew:
			serialized = *event.WorkflowExecutionContinuedAsNewEventAttributes.Input
		}
		if serialized != "" {
			f.Deserialize(serialized, eventData)
		} else {
			panic(fmt.Sprintf("event payload was empty for %s", PrettyHistoryEvent(event)))
		}
	}

}

func (f *FSM) log(format string, data ...interface{}) {
	actualFormat := fmt.Sprintf("component=FSM name=%s %s", f.Name, format)
	log.Printf(actualFormat, data...)
}

func (f *FSM) findSerializedState(events []swf.HistoryEvent) (*SerializedState, error) {
	for _, event := range events {
		if f.isStateMarker(event) {
			state := &SerializedState{}
			err := f.Serializer.Deserialize(*event.MarkerRecordedEventAttributes.Details, state)
			return state, err
		} else if *event.EventType == swf.EventTypeWorkflowExecutionStarted {
			log.Println(event)
			state := &SerializedState{}
			err := f.Serializer.Deserialize(*event.WorkflowExecutionStartedEventAttributes.Input, state)
			if err == nil {
				if state.StateName == "" {
					state.StateName = f.initialState.Name
				}
			}
			return state, err
		}
	}
	return nil, errors.New("Cant Find Current Data")
}

func (f *FSM) findSerializedEventCorrelator(events []swf.HistoryEvent) (*EventCorrelator, error) {
	for _, event := range events {
		if f.isCorrelatorMarker(event) {
			correlator := &EventCorrelator{}
			err := f.Serializer.Deserialize(*event.MarkerRecordedEventAttributes.Details, correlator)
			return correlator, err
		}
	}
	return &EventCorrelator{}, nil
}

func (f *FSM) findLastEvents(prevStarted int64, events []swf.HistoryEvent) []swf.HistoryEvent {
	var lastEvents []swf.HistoryEvent

	for _, event := range events {
		if *event.EventID == prevStarted {
			return lastEvents
		}
		switch *event.EventType {
		case swf.EventTypeDecisionTaskCompleted, swf.EventTypeDecisionTaskScheduled,
			swf.EventTypeDecisionTaskStarted:
			//no-op, dont even process these?
		case swf.EventTypeMarkerRecorded:
			if !f.isStateMarker(event) {
				lastEvents = append(lastEvents, event)
			}
		default:
			lastEvents = append(lastEvents, event)
		}

	}

	return lastEvents
}

func (f *FSM) recordStateMarkers(outcome *intermediateOutcome, eventCorrelator *EventCorrelator) ([]swf.Decision, *SerializedState, error) {
	serializedData, err := f.Serializer.Serialize(outcome.data)

	state := &SerializedState{
		StateVersion: outcome.stateVersion + 1, //increment the version here only.
		StateName:    outcome.state,
		StateData:    serializedData,
	}
	serializedMarker, err := f.systemSerializer.Serialize(state)

	if err != nil {
		return nil, state, err
	}

	serializedCorrelator, err := f.systemSerializer.Serialize(eventCorrelator)

	if err != nil {
		return nil, state, err
	}

	d := f.recordStringMarker(StateMarker, serializedMarker)
	c := f.recordStringMarker(CorrelatorMarker, serializedCorrelator)
	decisions := f.EmptyDecisions()
	decisions = append(decisions, d, c)
	decisions = append(decisions, outcome.decisions...)
	return decisions, state, nil
}

func (f *FSM) recordMarker(markerName string, details interface{}) (swf.Decision, error) {
	serialized, err := f.Serializer.Serialize(details)
	if err != nil {
		return swf.Decision{}, err
	}

	return f.recordStringMarker(markerName, serialized), nil
}

func (f *FSM) recordStringMarker(markerName string, details string) swf.Decision {
	return swf.Decision{
		DecisionType: aws.String(swf.DecisionTypeRecordMarker),
		RecordMarkerDecisionAttributes: &swf.RecordMarkerDecisionAttributes{
			MarkerName: aws.String(markerName),
			Details:    aws.String(details),
		},
	}
}

// Stop causes the DecisionTask select loop to exit, and to stop the DecisionTaskPoller
func (f *FSM) Stop() {
	f.stop <- true
}

func (f *FSM) isStateMarker(e swf.HistoryEvent) bool {
	return *e.EventType == swf.EventTypeMarkerRecorded && *e.MarkerRecordedEventAttributes.MarkerName == StateMarker
}

func (f *FSM) isCorrelatorMarker(e swf.HistoryEvent) bool {
	return *e.EventType == swf.EventTypeMarkerRecorded && *e.MarkerRecordedEventAttributes.MarkerName == CorrelatorMarker
}

func (f *FSM) isErrorSignal(e swf.HistoryEvent) bool {
	if *e.EventType == swf.EventTypeWorkflowExecutionSignaled {
		switch *e.WorkflowExecutionSignaledEventAttributes.SignalName {
		case SystemErrorSignal, ErrorSignal:
			return true
		default:
			return false
		}
	} else {
		return false
	}
}

// EmptyDecisions is a helper method to give you an empty decisions array for use in your Deciders.
func (f *FSM) EmptyDecisions() []swf.Decision {
	return make([]swf.Decision, 0)
}

// StartFSMWorkflowInput should be used to construct the input for any StartWorkflowExecutionRequests.
// This panics on errors cause really this should never err.
func StartFSMWorkflowInput(serializer StateSerializer, data interface{}) aws.StringValue {
	ss := new(SerializedState)
	stateData, err := serializer.Serialize(data)
	if err != nil {
		panic(err)
	}

	ss.StateData = stateData
	serialized, err := serializer.Serialize(ss)
	if err != nil {
		panic(err)
	}
	return aws.String(serialized)
}

//ContinueFSMWorkflowInput should be used to construct the input for any ContinueAsNewWorkflowExecution decisions.
func ContinueFSMWorkflowInput(ctx *FSMContext, data interface{}) aws.StringValue {
	ss := new(SerializedState)
	stateData := ctx.Serialize(data)

	ss.StateData = stateData
	ss.StateName = ctx.serialization.InitialState()
	ss.StateVersion = ctx.stateVersion

	return aws.String(ctx.Serialize(ss))
}
