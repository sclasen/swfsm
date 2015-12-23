package fsm

import (
	"fmt"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/juju/errors"
	. "github.com/sclasen/swfsm/log"
	"github.com/sclasen/swfsm/poller"
	s "github.com/sclasen/swfsm/sugar"
)

//SWFOps is the subset of swf.SWF ops required by the fsm package
type SWFOps interface {
	PollForDecisionTaskPages(*swf.PollForDecisionTaskInput, func(*swf.PollForDecisionTaskOutput, bool) bool) error
	RespondDecisionTaskCompleted(*swf.RespondDecisionTaskCompletedInput) (*swf.RespondDecisionTaskCompletedOutput, error)
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
	// DecisionInterceptor fsm will call BeforeDecision/AfterDecision
	DecisionInterceptor DecisionInterceptor
	//DecisionErrorHandler  is called whenever there is a panic in your decider.
	//if it returns a nil *Outcome, the attempt to handle the DecisionTask is abandoned.
	//fsm will then mark the workflow as being in error, by recording 3 markers. state, correlator and error
	//the error marker  contains an ErrorState which tracks the range of unprocessed events since the error occurred.
	//on subsequent decision tasks if the fsm detects an error state, it will get the ErrorEvent from the ErrorState
	//and call the DecisionErrorHandler again.
	//
	//If there are errors here a new ErrorMarker with the increased range of unprocessed events
	//will be recorded.
	//If there is a good outcome, then we use that as the starting point from which to grab and Decide on the range of unprocessed
	//events. If this works out fine, we then process the initiating decisionTask range of events.
	DecisionErrorHandler DecisionErrorHandler
	//FSMErrorReporter  is called whenever there is an error within the FSM, usually indicating bad state or configuration of your FSM.
	FSMErrorReporter FSMErrorReporter
	states           map[string]*FSMState
	errorHandlers    map[string]DecisionErrorHandler
	initialState     *FSMState
	completeState    *FSMState
	canceledState    *FSMState
	stop             chan bool
	stopAck          chan bool
	allowPanics      bool //makes testing easier
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

// AddCanceledState adds a state to the FSM and uses it as the final state of a workflow.
// it will only receive events if you returned FSMContext.CancelWorkflow(...) and the workflow was unable to cancel.
func (f *FSM) AddCanceledState(state *FSMState) {
	f.AddState(state)
	f.canceledState = state
}

// AddInitialStateWithHandler adds a state to the FSM and uses it as the initial state when a workflow execution is started.
// it uses the FSM DefaultDecisionErrorHandler, which defaults to FSM.DefaultDecisionErrorHandler if unset.
func (f *FSM) AddInitialStateWithHandler(state *FSMState, handler DecisionErrorHandler) {
	f.AddState(state)
	f.AddErrorHandler(state.Name, handler)
	f.initialState = state
}

// AddErrorHandler adds a DecisionErrorHandler  to a state in the FSM.
func (f *FSM) AddErrorHandler(state string, handler DecisionErrorHandler) {
	if f.errorHandlers == nil {
		f.errorHandlers = make(map[string]DecisionErrorHandler)
	}
	f.errorHandlers[state] = handler
}

// AddCompleteStateWithHandler adds a state to the FSM and uses it as the final state of a workflow.
// it will only receive events if you returned FSMContext.Complete(...) and the workflow was unable to complete.
// It also adds a DecisionErrorHandler to the state.
func (f *FSM) AddCompleteStateWithHandler(state *FSMState, handler DecisionErrorHandler) {
	f.AddState(state)
	f.AddErrorHandler(state.Name, handler)
	f.completeState = state
}

// DefaultCompleteState is the complete state used in an FSM if one has not been set.
// It simply responds with a CompleteDecision which attempts to Complete the workflow.
// This state will only get events if you previously attempted to complete the workflow and it failed.
func (f *FSM) DefaultCompleteState() *FSMState {
	return &FSMState{
		Name: CompleteState,
		Decider: func(fsm *FSMContext, h *swf.HistoryEvent, data interface{}) Outcome {
			f.log("state=complete at=attempt-completion event=%s", h)
			return fsm.CompleteWorkflow(data)
		},
	}
}

// DefaultCanceledState is the canceled state used in an FSM if one has not been set.
// It simply responds with a CancelWorkflow which attempts to Cancel the workflow.
// This state will only get events if you previously attempted to cancel the workflow and it failed.
func (f *FSM) DefaultCanceledState() *FSMState {
	return &FSMState{
		Name: CanceledState,
		Decider: func(fsm *FSMContext, h *swf.HistoryEvent, data interface{}) Outcome {
			f.log("state=complete at=attempt-cancel event=%s", h)
			return fsm.CancelWorkflow(data, s.S("default-canceled-state"))
		},
	}
}

// DefaultDecisionErrorHandler is the DefaultDecisionErrorHandler
func (f *FSM) DefaultDecisionErrorHandler(ctx *FSMContext, event *swf.HistoryEvent, stateBeforeEvent interface{}, stateAfterError interface{}, err error) (*Outcome, error) {
	f.log("action=tick workflow=%s workflow-id=%s at=decider-error err=%q", s.LS(ctx.WorkflowType.Name), s.LS(ctx.WorkflowId), err)
	return nil, err
}

// ErrorFindingStateData is part of the FSM implementation of FSMErrorReporter
func (f *FSM) ErrorFindingStateData(decisionTask *swf.PollForDecisionTaskOutput, err error) {
	f.log("action=tick workflow=%s workflow-id=%s at=error=find-serialized-state-failed err=%q", s.LS(decisionTask.WorkflowType.Name), s.LS(decisionTask.WorkflowExecution.WorkflowId), err)
}

// ErrorFindingCorrelator is part of the FSM implementation of FSMErrorReporter
func (f *FSM) ErrorFindingCorrelator(decisionTask *swf.PollForDecisionTaskOutput, err error) {
	f.log("action=tick workflow=%s workflow-id=%s at=error=find-serialized-event-correlator-failed err=%q", s.LS(decisionTask.WorkflowType.Name), s.LS(decisionTask.WorkflowExecution.WorkflowId), err)
}

// ErrorMissingFSMState is part of the FSM implementation of FSMErrorReporter
func (f *FSM) ErrorMissingFSMState(decisionTask *swf.PollForDecisionTaskOutput, outcome Outcome) {
	f.log("action=tick workflow=%s workflow-id=%s at=error error=marked-state-not-in-fsm state=%s", s.LS(decisionTask.WorkflowType.Name), s.LS(decisionTask.WorkflowExecution.WorkflowId), outcome.State)
}

// ErrorDeserializingStateData is part of the FSM implementation of FSMErrorReporter
func (f *FSM) ErrorDeserializingStateData(decisionTask *swf.PollForDecisionTaskOutput, serializedStateData string, err error) {
	f.log("action=tick workflow=%s workflow-id=%s at=error=deserialize-state-failed err=&s", s.LS(decisionTask.WorkflowType.Name), s.LS(decisionTask.WorkflowExecution.WorkflowId), err)
}

// ErrorSerializingStateData is part of the FSM implementation of FSMErrorReporter
func (f *FSM) ErrorSerializingStateData(decisionTask *swf.PollForDecisionTaskOutput, outcome Outcome, eventCorrelator EventCorrelator, err error) {
	f.log("action=tick workflow=%s workflow-id=%s at=error error=state-serialization-error err=%q error-type=system", s.LS(decisionTask.WorkflowType.Name), s.LS(decisionTask.WorkflowExecution.WorkflowId), err)

}

// Init initializes any optional, unspecified values such as the error state, stop channel, serializer, PollerShutdownManager.
// it gets called by Start(), so you should only call this if you are manually managing polling for tasks, and calling Tick yourself.
func (f *FSM) Init() {
	if f.initialState == nil {
		panic("No Initial State Defined For FSM")
	}

	if f.completeState == nil {
		f.AddCompleteState(f.DefaultCompleteState())
	}

	if f.canceledState == nil {
		f.AddCanceledState(f.DefaultCanceledState())
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

	if f.DecisionErrorHandler == nil {
		f.DecisionErrorHandler = f.DefaultDecisionErrorHandler
	}

	if f.FSMErrorReporter == nil {
		f.FSMErrorReporter = f
	}

}

// Start begins processing DecisionTasks with the FSM. It creates a DecisionTaskPoller and spawns a goroutine that continues polling until Stop() is called and any in-flight polls have completed.
// If you wish to manage polling and calling Tick() yourself, you dont need to start the FSM, just call Init().
func (f *FSM) Start() {
	f.Init()
	poller := poller.NewDecisionTaskPoller(f.SWF, f.Domain, f.Identity, f.TaskList)
	go poller.PollUntilShutdownBy(f.ShutdownManager, fmt.Sprintf("%s-poller", f.Name), f.dispatchTask, f.taskReady)
}

// signals the poller to stop reading decision task pages once we have marker events
func (f *FSM) taskReady(task *swf.PollForDecisionTaskOutput) bool {
	var state, correlator bool
	for _, e := range task.Events {
		if f.isStateMarker(e) {
			state = true
		}

		if f.isCorrelatorMarker(e) {
			correlator = true
		}

		if state && correlator {
			f.log("workflow=%q fn=taskReady at=state-and-correlator-and-prev-found eventid=%s",
				s.LS(task.WorkflowExecution.WorkflowId), s.LL(e.EventId))
			return true
		}

		if e.EventType != nil && *e.EventType == swf.EventTypeWorkflowExecutionStarted {
			f.log("workflow=%q fn=taskReady at=start-event eventid=%s",
				s.LS(task.WorkflowExecution.WorkflowId), s.LL(e.EventId))
			return true
		}
	}
	return false
}

func (f *FSM) dispatchTask(decisionTask *swf.PollForDecisionTaskOutput) {
	f.DecisionTaskDispatcher.DispatchTask(decisionTask, f.handleDecisionTask)
}

func (f *FSM) handleDecisionTask(decisionTask *swf.PollForDecisionTaskOutput) {
	context, decisions, state, err := f.Tick(decisionTask)
	if err != nil {
		f.log("workflow=%s workflow-id=%s run-id=%s action=tick at=tick-error status=abandoning-task error=%q", s.LS(decisionTask.WorkflowType.Name), s.LS(decisionTask.WorkflowExecution.WorkflowId), s.LS(decisionTask.WorkflowExecution.RunId), err.Error())
		return
	}
	complete := &swf.RespondDecisionTaskCompletedInput{
		Decisions: decisions,
		TaskToken: decisionTask.TaskToken,
	}

	complete.ExecutionContext = aws.String(state.StateName)

	if _, err := f.SWF.RespondDecisionTaskCompleted(complete); err != nil {
		f.log("workflow=%s workflow-id=%s action=tick at=decide-request-failed error=%q", s.LS(decisionTask.WorkflowType.Name), s.LS(decisionTask.WorkflowExecution.WorkflowId), s.LS(decisionTask.WorkflowExecution.RunId), err.Error())
		return
	}

	if f.ReplicationHandler != nil {
		repErr := f.ReplicationHandler(context, decisionTask, complete, state)
		if repErr != nil {
			f.log("workflow=%s workflow-id=%s action=tick at=replication-handler-failed error=%q", s.LS(decisionTask.WorkflowType.Name), s.LS(decisionTask.WorkflowExecution.WorkflowId), s.LS(decisionTask.WorkflowExecution.RunId), repErr.Error())
		}
	}

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
func (f *FSM) Tick(decisionTask *swf.PollForDecisionTaskOutput) (*FSMContext, []*swf.Decision, *SerializedState, error) {
	//BeforeDecision interceptor invocation
	if f.DecisionInterceptor != nil {
		f.DecisionInterceptor.BeforeTask(decisionTask)
	}
	lastEvents := f.findLastEvents(*decisionTask.PreviousStartedEventId, decisionTask.Events)
	outcome := new(Outcome)
	context := NewFSMContext(f,
		*decisionTask.WorkflowType,
		*decisionTask.WorkflowExecution,
		nil,
		"", nil, uint64(0),
	)

	serializedState, err := f.findSerializedState(decisionTask.Events)
	if err != nil {
		f.FSMErrorReporter.ErrorFindingStateData(decisionTask, err)
		if f.allowPanics {
			panic(err)
		}
		return nil, nil, nil, errors.Trace(err)
	}
	eventCorrelator, err := f.findSerializedEventCorrelator(decisionTask.Events)
	if err != nil {
		f.FSMErrorReporter.ErrorFindingCorrelator(decisionTask, err)
		if f.allowPanics {
			panic(err)
		}
		return nil, nil, nil, errors.Trace(err)
	}
	context.eventCorrelator = eventCorrelator

	f.clog(context, "action=tick at=find-serialized-state state=%s", serializedState.StateName)

	if outcome.Data == nil && outcome.State == "" {
		data := f.zeroStateData()
		if err = f.Serializer.Deserialize(serializedState.StateData, data); err != nil {
			f.FSMErrorReporter.ErrorDeserializingStateData(decisionTask, serializedState.StateData, err)
			if f.allowPanics {
				panic(err)
			}
			return nil, nil, nil, errors.Trace(err)
		}
		f.clog(context, "action=tick at=find-current-data data=%v", data)
		outcome.Data = data
		outcome.State = serializedState.StateName
		context.stateVersion = serializedState.StateVersion
		// BeforeDecisionContext interceptor invocation
		if f.DecisionInterceptor != nil {
			before := &Outcome{Data: outcome.Data, Decisions: outcome.Decisions, State: outcome.State}
			f.DecisionInterceptor.BeforeDecision(decisionTask, context, before)
			outcome.State = before.State
			outcome.Decisions = before.Decisions
			outcome.Data = before.Data
		}
	}

	errorState, err := f.findSerializedErrorState(decisionTask.Events)
	if errorState != nil {
		recovery, err := f.ErrorStateTick(decisionTask, errorState, context, outcome.Data)
		if recovery != nil {
			outcome = recovery
		} else {
			logf(context, "at=error error=error-recovery-failed cause=%s", err)
			//bump the unprocessed window, and re-record the error marker
			errorState.LatestUnprocessedEventId = *decisionTask.StartedEventId
			final, serializedState, err := f.recordStateMarkers(context, outcome, eventCorrelator, errorState)
			//update Error State Marker and exit with 3 marker decisions
			return context, final, serializedState, err
		}
	}

	//iterate through events oldest to newest, calling the decider for the current state.
	//if the outcome changes the state use the right FSMState
	for i := len(lastEvents) - 1; i >= 0; i-- {
		e := lastEvents[i]
		f.clog(context, "action=tick at=history id=%d type=%s", *e.EventId, *e.EventType)
		fsmState, ok := f.states[outcome.State]
		if ok {
			context.State = outcome.State
			context.stateData = outcome.Data
			//stash a copy of the state before the decision in case we need to call the error handler
			stashed := f.Serialize(outcome.Data)
			anOutcome, err := f.panicSafeDecide(fsmState, context, e, outcome.Data)
			if err != nil {
				stashedData := f.zeroStateData()
				f.Deserialize(stashed, stashedData)
				handler := f.errorHandlers[fsmState.Name]
				if handler == nil {
					handler = f.DecisionErrorHandler
				}
				rescued, notRescued := handler(context, e, stashedData, outcome.Data, err)
				if rescued != nil {
					anOutcome = *rescued
				} else {
					notRescuedSerialized := ""
					if notRescued != nil {
						notRescuedSerialized = notRescued.Error()
					}
					errorState := &SerializedErrorState{
						Details:                    notRescuedSerialized,
						ErrorEvent:                 e,
						EarliestUnprocessedEventId: *decisionTask.PreviousStartedEventId + 1,
						LatestUnprocessedEventId:   *decisionTask.StartedEventId,
					}
					final, serializedState, err := f.recordStateMarkers(context, outcome, eventCorrelator, errorState)
					if err != nil {
						f.FSMErrorReporter.ErrorSerializingStateData(decisionTask, *outcome, *eventCorrelator, err)
						if f.allowPanics {
							panic(err)
						}
						return nil, nil, nil, errors.Trace(err)
					}
					return context, final, serializedState, nil
				}
			}
			//NOTE this call is handled in fsmContext.Decide. The double call causes nil panics
			//eventCorrelator.Track(e)
			curr := outcome.State
			f.mergeOutcomes(outcome, anOutcome)
			f.clog(context, "action=tick at=decided-event state=%s next-state=%s decisions=%d", curr, outcome.State, len(anOutcome.Decisions))
		} else {
			f.FSMErrorReporter.ErrorMissingFSMState(decisionTask, *outcome)
			return nil, nil, nil, errors.New("marked-state-not-in-fsm state=" + outcome.State)
		}
	}

	f.clog(context, "action=tick at=events-processed next-state=%s decisions=%d", outcome.State, len(outcome.Decisions))

	for _, d := range outcome.Decisions {
		f.clog(context, "action=tick at=decide next-state=%s decision=%s", outcome.State, *d.DecisionType)
	}
	//AfterDecision interceptor invocation
	if f.DecisionInterceptor != nil {
		context.State = outcome.State
		context.stateData = outcome.Data
		after := &Outcome{Data: outcome.Data, Decisions: outcome.Decisions, State: outcome.State}
		f.DecisionInterceptor.AfterDecision(decisionTask, context, after)
		outcome.State = after.State
		outcome.Decisions = after.Decisions
		outcome.Data = after.Data
	}

	final, serializedState, err := f.recordStateMarkers(context, outcome, context.eventCorrelator, nil)
	if err != nil {
		f.FSMErrorReporter.ErrorSerializingStateData(decisionTask, *outcome, *eventCorrelator, err)
		if f.allowPanics {
			panic(err)
		}
		return nil, nil, nil, errors.Trace(err)
	}

	return context, final, serializedState, nil
}

// ErrorStateTick is called when the DecisionTaskPoller receives a PollForDecisionTaskResponse in its polling loop
// that contains an error marker in its history.
func (f *FSM) ErrorStateTick(decisionTask *swf.PollForDecisionTaskOutput, error *SerializedErrorState, context *FSMContext, data interface{}) (*Outcome, error) {
	handler := f.errorHandlers[context.State]
	if handler == nil {
		handler = f.DecisionErrorHandler
	}
	handled, notHandled := handler(context, error.ErrorEvent, data, data, nil)
	if handled == nil {
		return nil, notHandled
	}

	//todo we are assuming all history events in the range
	//error.EarliestUnprocessedEventId to error.LatestUnprocessedEventId
	//are in the decisionTaks.History
	filteredDecisionTask := new(swf.PollForDecisionTaskOutput)
	s, e := f.systemSerializer.Serialize(decisionTask)
	if e != nil {
		return nil, e
	}
	e = f.systemSerializer.Deserialize(s, filteredDecisionTask)
	if e != nil {
		return nil, e
	}

	filtered := make([]*swf.HistoryEvent, 0)
	for _, h := range decisionTask.Events {
		if f.isErrorMarker(h) {
			continue
		}
		filtered = append(filtered, h)
	}
	filteredDecisionTask.Events = filtered
	filteredDecisionTask.StartedEventId = &error.LatestUnprocessedEventId
	filteredDecisionTask.PreviousStartedEventId = &error.EarliestUnprocessedEventId

	_, decisions, serializedState, err := f.Tick(filteredDecisionTask)
	if err != nil {
		data := f.zeroStateData()
		f.Deserialize(serializedState.StateData, data)

		return &Outcome{
			State:     serializedState.StateName,
			Decisions: decisions,
			Data:      data,
		}, nil

	}

	return nil, err
}

func (f *FSM) mergeOutcomes(final *Outcome, intermediate Outcome) {
	final.Decisions = append(final.Decisions, intermediate.Decisions...)
	final.Data = intermediate.Data
	if intermediate.State != "" {
		final.State = intermediate.State
	}
}

func (f *FSM) panicSafeDecide(state *FSMState, context *FSMContext, event *swf.HistoryEvent, data interface{}) (anOutcome Outcome, anErr error) {
	defer func() {
		if !f.allowPanics {
			if r := recover(); r != nil {
				f.log("at=error error=decide-panic-recovery %v", r)
				if err, ok := r.(error); ok && err != nil {
					anErr = errors.Trace(err)
				} else {
					anErr = errors.New(fmt.Sprintf("panic in decider: %#v", r))
				}
			}
		} else {
			Log.Printf("at=panic-safe-decide-allowing-panic fsm-allow-panics=%t", f.allowPanics)
		}
	}()
	anOutcome = context.Decide(event, data, state.Decider)
	return
}

// EventData works in combination with the FSM.Serializer to provide
// deserialization of data sent in a HistoryEvent. It is sugar around extracting the event payload from the proper
// field of the proper Attributes struct on the HistoryEvent
func (f *FSM) EventData(event *swf.HistoryEvent, eventData interface{}) {

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
			switch *event.WorkflowExecutionSignaledEventAttributes.SignalName {
			case ActivityStartedSignal, ActivityUpdatedSignal:
				state := new(SerializedActivityState)
				f.systemSerializer.Deserialize(*event.WorkflowExecutionSignaledEventAttributes.Input, state)
				if state.Input != nil {
					serialized = *state.Input
				}
			default:
				serialized = *event.WorkflowExecutionSignaledEventAttributes.Input
			}
		case swf.EventTypeWorkflowExecutionStarted:
			serialized = *event.WorkflowExecutionStartedEventAttributes.Input
		case swf.EventTypeWorkflowExecutionContinuedAsNew:
			serialized = *event.WorkflowExecutionContinuedAsNewEventAttributes.Input
		}
		if serialized != "" {
			f.Deserialize(serialized, eventData)
		} else {
			panic(fmt.Sprintf("event payload was empty for %s", s.PrettyHistoryEvent(event)))
		}
	}

}

func (f *FSM) log(format string, data ...interface{}) {
	actualFormat := fmt.Sprintf("component=FSM name=%s %s", f.Name, format)
	Log.Printf(actualFormat, data...)
}

func (f *FSM) clog(ctx *FSMContext, format string, data ...interface{}) {
	actualFormat := fmt.Sprintf("component=FSM name=%s type=%s id=%s %s", f.Name, s.LS(ctx.WorkflowType.Name), s.LS(ctx.WorkflowId), format)
	Log.Printf(actualFormat, data...)
}

func (f *FSM) findSerializedState(events []*swf.HistoryEvent) (*SerializedState, error) {
	for _, event := range events {
		if state, err := f.statefulHistoryEventToSerializedState(event); state != nil || err != nil {
			return state, err
		}
	}
	return nil, errors.New("Cant Find Current Data")
}

func (f *FSM) statefulHistoryEventToSerializedState(event *swf.HistoryEvent) (*SerializedState, error) {
	if f.isStateMarker(event) {
		state := &SerializedState{}
		err := f.systemSerializer.Deserialize(*event.MarkerRecordedEventAttributes.Details, state)
		return state, err
	} else if *event.EventType == swf.EventTypeWorkflowExecutionStarted {
		state := &SerializedState{}
		err := f.Serializer.Deserialize(*event.WorkflowExecutionStartedEventAttributes.Input, state)
		if err == nil {
			if state.StateName == "" {
				state.StateName = f.initialState.Name
			}
		}
		return state, err

	}
	return nil, nil
}

func (f *FSM) findSerializedEventCorrelator(events []*swf.HistoryEvent) (*EventCorrelator, error) {
	for _, event := range events {
		if f.isCorrelatorMarker(event) {
			correlator := &EventCorrelator{
				Serializer: f.systemSerializer,
			}
			err := f.Serializer.Deserialize(*event.MarkerRecordedEventAttributes.Details, correlator)
			return correlator, err
		}
	}
	return &EventCorrelator{
		Serializer: f.systemSerializer,
	}, nil
}

func (f *FSM) findSerializedErrorState(events []*swf.HistoryEvent) (*SerializedErrorState, error) {
	for _, event := range events {
		if f.isErrorMarker(event) {
			errState := &SerializedErrorState{}
			err := f.Serializer.Deserialize(*event.MarkerRecordedEventAttributes.Details, errState)
			return errState, err
		}
	}
	return nil, nil
}

func (f *FSM) findLastEvents(prevStarted int64, events []*swf.HistoryEvent) []*swf.HistoryEvent {
	var lastEvents []*swf.HistoryEvent

	for _, event := range events {
		if *event.EventId == prevStarted {
			return lastEvents
		}
		switch *event.EventType {
		case swf.EventTypeDecisionTaskCompleted, swf.EventTypeDecisionTaskScheduled,
			swf.EventTypeDecisionTaskStarted:
			//no-op, dont even process these?
		case swf.EventTypeMarkerRecorded:
			if !f.isStateMarker(event) && !f.isCorrelatorMarker(event) {
				lastEvents = append(lastEvents, event)
			}
		default:
			lastEvents = append(lastEvents, event)
		}

	}

	return lastEvents
}

func (f *FSM) recordStateMarkers(context *FSMContext, outcome *Outcome, eventCorrelator *EventCorrelator, errorState *SerializedErrorState) ([]*swf.Decision, *SerializedState, error) {
	serializedData, err := f.Serializer.Serialize(outcome.Data)

	state := &SerializedState{
		StateVersion: context.stateVersion + 1, //increment the version here only.
		StateName:    outcome.State,
		StateData:    serializedData,
		WorkflowId:   *context.WorkflowId,
	}
	serializedMarker, err := f.systemSerializer.Serialize(state)

	if err != nil {
		return nil, state, errors.Trace(err)
	}

	serializedCorrelator, err := f.systemSerializer.Serialize(eventCorrelator)

	if err != nil {
		return nil, state, errors.Trace(err)
	}

	d := f.recordStringMarker(StateMarker, serializedMarker)
	c := f.recordStringMarker(CorrelatorMarker, serializedCorrelator)
	decisions := f.EmptyDecisions()
	decisions = append(decisions, d, c)

	if errorState != nil {
		serializedError, err := f.systemSerializer.Serialize(*errorState)

		if err != nil {
			return nil, state, errors.Trace(err)
		}
		e := f.recordStringMarker(ErrorMarker, serializedError)
		decisions = append(decisions, e)
	}

	decisions = append(decisions, outcome.Decisions...)
	return decisions, state, nil
}

func (f *FSM) recordMarker(markerName string, details interface{}) (*swf.Decision, error) {
	serialized, err := f.Serializer.Serialize(details)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return f.recordStringMarker(markerName, serialized), nil
}

func (f *FSM) recordStringMarker(markerName string, details string) *swf.Decision {
	return &swf.Decision{
		DecisionType: aws.String(swf.DecisionTypeRecordMarker),
		RecordMarkerDecisionAttributes: &swf.RecordMarkerDecisionAttributes{
			MarkerName: aws.String(markerName),
			Details:    aws.String(details),
		},
	}
}

func (f *FSM) zeroStateData() interface{} {
	return reflect.New(reflect.TypeOf(f.DataType)).Interface()
}

// Stop causes the DecisionTask select loop to exit, and to stop the DecisionTaskPoller
func (f *FSM) Stop() {
	f.stop <- true
}

func (f *FSM) isStateMarker(e *swf.HistoryEvent) bool {
	return *e.EventType == swf.EventTypeMarkerRecorded && *e.MarkerRecordedEventAttributes.MarkerName == StateMarker
}

func (f *FSM) isCorrelatorMarker(e *swf.HistoryEvent) bool {
	return *e.EventType == swf.EventTypeMarkerRecorded && *e.MarkerRecordedEventAttributes.MarkerName == CorrelatorMarker
}

func (f *FSM) isErrorMarker(e *swf.HistoryEvent) bool {
	return *e.EventType == swf.EventTypeMarkerRecorded && *e.MarkerRecordedEventAttributes.MarkerName == ErrorMarker
}

// EmptyDecisions is a helper method to give you an empty decisions array for use in your Deciders.
func (f *FSM) EmptyDecisions() []*swf.Decision {
	return make([]*swf.Decision, 0)
}
