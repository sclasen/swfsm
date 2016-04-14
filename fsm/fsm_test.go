package fsm

import (
	"strconv"
	"testing"
	"time"

	"errors"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/swf"
	. "github.com/sclasen/swfsm/log"
	. "github.com/sclasen/swfsm/sugar"
	"github.com/sclasen/swfsm/testing/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

//Todo add tests of error handling mechanism
//assert that the decisions have the mark and the signal external...hmm need workflow id for signal external.

var testActivityInfo = ActivityInfo{ActivityId: "activityId", ActivityType: &swf.ActivityType{Name: S("activity"), Version: S("activityVersion")}}

var typedFuncs = Typed(new(TestData))

func TestFSM(t *testing.T) {

	fsm := testFSM()

	fsm.AddInitialState(&FSMState{
		Name: "start",
		Decider: func(f *FSMContext, lastEvent *swf.HistoryEvent, data interface{}) Outcome {
			testData := data.(*TestData)
			testData.States = append(testData.States, "start")
			serialized := f.Serialize(testData)
			decision := &swf.Decision{
				DecisionType: S(swf.DecisionTypeScheduleActivityTask),
				ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
					ActivityId:   S(testActivityInfo.ActivityId),
					ActivityType: testActivityInfo.ActivityType,
					TaskList:     &swf.TaskList{Name: S("taskList")},
					Input:        S(serialized),
				},
			}

			return f.Goto("working", testData, []*swf.Decision{decision})

		},
	})

	fsm.AddState(&FSMState{
		Name: "working",
		Decider: typedFuncs.Decider(func(f *FSMContext, lastEvent *swf.HistoryEvent, testData *TestData) Outcome {
			testData.States = append(testData.States, "working")
			serialized := f.Serialize(testData)
			var decisions = f.EmptyDecisions()
			if *lastEvent.EventType == swf.EventTypeActivityTaskCompleted {
				decision := &swf.Decision{
					DecisionType: aws.String(swf.DecisionTypeCompleteWorkflowExecution),
					CompleteWorkflowExecutionDecisionAttributes: &swf.CompleteWorkflowExecutionDecisionAttributes{
						Result: S(serialized),
					},
				}
				decisions = append(decisions, decision)
			} else if *lastEvent.EventType == swf.EventTypeActivityTaskFailed {
				decision := &swf.Decision{
					DecisionType: aws.String(swf.DecisionTypeScheduleActivityTask),
					ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
						ActivityId:   S(testActivityInfo.ActivityId),
						ActivityType: testActivityInfo.ActivityType,
						TaskList:     &swf.TaskList{Name: S("taskList")},
						Input:        S(serialized),
					},
				}
				decisions = append(decisions, decision)
			}

			return f.Stay(testData, decisions)
		}),
	})

	events := []*swf.HistoryEvent{
		&swf.HistoryEvent{EventType: S("DecisionTaskStarted"), EventId: I(3)},
		&swf.HistoryEvent{EventType: S("DecisionTaskScheduled"), EventId: I(2)},
		EventFromPayload(1, &swf.WorkflowExecutionStartedEventAttributes{
			Input: StartFSMWorkflowInput(fsm, new(TestData)),
		}),
	}

	first := testDecisionTask(0, events)

	_, decisions, _, err := fsm.Tick(first)

	if err != nil {
		t.Fatal(err)
	}

	if !Find(decisions, stateMarkerPredicate) {
		t.Fatal("No Record State Marker")
	}

	if !Find(decisions, correlationMarkerPredicate) {
		t.Fatal("No Record Correlator Marker")
	}

	if !Find(decisions, scheduleActivityPredicate) {
		t.Fatal("No ScheduleActivityTask")
	}

	if Find(decisions, errorMarkerPredicate) {
		t.Fatal("Found Error Marker")
	}

	secondEvents := DecisionsToEvents(decisions)
	secondEvents = append(secondEvents, events...)

	if state, _ := fsm.findSerializedState(secondEvents); state.StateName != "working" {
		t.Fatal("current state is not 'working'", secondEvents)
	}

	second := testDecisionTask(3, secondEvents)

	_, secondDecisions, _, err := fsm.Tick(second)

	if err != nil {
		t.Fatal(err)
	}

	if !Find(secondDecisions, stateMarkerPredicate) {
		t.Fatal("No Record State Marker")
	}

	if !Find(secondDecisions, completeWorkflowPredicate) {
		t.Fatal("No CompleteWorkflow")
	}

	if Find(decisions, errorMarkerPredicate) {
		t.Fatal("Found Error Marker")
	}
}

func TestFSMError(t *testing.T) {
	fsm := testFSM()

	fsm.AddInitialState(&FSMState{
		Name: "start",
		Decider: func(f *FSMContext, lastEvent *swf.HistoryEvent, data interface{}) Outcome {
			panic("BOOM")
		},
	})
	fsm.Init()

	events := []*swf.HistoryEvent{
		&swf.HistoryEvent{EventType: S("DecisionTaskStarted"), EventId: I(3)},
		&swf.HistoryEvent{EventType: S("DecisionTaskScheduled"), EventId: I(2)},
		EventFromPayload(1, &swf.WorkflowExecutionStartedEventAttributes{
			Input: StartFSMWorkflowInput(fsm, new(TestData)),
		}),
	}

	tasks := testDecisionTask(0, events)

	fsm.AllowPanics = false
	_, decisions, _, err := fsm.Tick(tasks)

	if err != nil {
		t.Fatal(err)
	}

	if !Find(decisions, stateMarkerPredicate) {
		t.Fatal("No Record State Marker")
	}

	if !Find(decisions, correlationMarkerPredicate) {
		t.Fatal("No Record Correlator Marker")
	}

	if !Find(decisions, errorMarkerPredicate) {
		t.Fatal("No Error Marker")
	}
}

func Find(decisions []*swf.Decision, predicate func(*swf.Decision) bool) bool {
	return FindDecision(decisions, predicate) != nil
}

func FindDecision(decisions []*swf.Decision, predicate func(*swf.Decision) bool) *swf.Decision {
	for _, d := range decisions {
		if predicate(d) {
			return d
		}
	}
	return nil
}

func stateMarkerPredicate(d *swf.Decision) bool {
	return *d.DecisionType == "RecordMarker" && *d.RecordMarkerDecisionAttributes.MarkerName == StateMarker
}

func correlationMarkerPredicate(d *swf.Decision) bool {
	return *d.DecisionType == "RecordMarker" && *d.RecordMarkerDecisionAttributes.MarkerName == CorrelatorMarker
}

func errorMarkerPredicate(d *swf.Decision) bool {
	return *d.DecisionType == "RecordMarker" && *d.RecordMarkerDecisionAttributes.MarkerName == ErrorMarker
}

func scheduleActivityPredicate(d *swf.Decision) bool {
	return *d.DecisionType == "ScheduleActivityTask"
}

func completeWorkflowPredicate(d *swf.Decision) bool {
	return *d.DecisionType == "CompleteWorkflowExecution"
}

func cancelWorkflowPredicate(d *swf.Decision) bool {
	return *d.DecisionType == "CancelWorkflowExecution"
}

func failWorkflowPredicate(d *swf.Decision) bool {
	return *d.DecisionType == "FailWorkflowExecution"
}

func startTimerPredicate(d *swf.Decision) bool {
	return *d.DecisionType == "StartTimer"
}

func DecisionsToEvents(decisions []*swf.Decision) []*swf.HistoryEvent {
	var events []*swf.HistoryEvent
	for _, d := range decisions {
		if scheduleActivityPredicate(d) {
			event := &swf.HistoryEvent{
				EventType: S("ActivityTaskCompleted"),
				EventId:   I(7),
				ActivityTaskCompletedEventAttributes: &swf.ActivityTaskCompletedEventAttributes{
					ScheduledEventId: I(6),
				},
			}
			events = append(events, event)
			event = &swf.HistoryEvent{
				EventType: S("ActivityTaskScheduled"),
				EventId:   I(6),
				ActivityTaskScheduledEventAttributes: &swf.ActivityTaskScheduledEventAttributes{
					ActivityId:   S(testActivityInfo.ActivityId),
					ActivityType: testActivityInfo.ActivityType,
				},
			}
			events = append(events, event)
		}
		if stateMarkerPredicate(d) {
			event := &swf.HistoryEvent{
				EventType: S("MarkerRecorded"),
				EventId:   I(5),
				MarkerRecordedEventAttributes: &swf.MarkerRecordedEventAttributes{
					MarkerName: S(StateMarker),
					Details:    d.RecordMarkerDecisionAttributes.Details,
				},
			}
			events = append(events, event)

		}
	}
	return events
}

type TestData struct {
	States []string
}

func (t *TestData) Tags() []*string {
	return []*string{S("tag1"), S("tag2")}
}

func TestMarshalledDecider(t *testing.T) {
	typedDecider := func(f *FSMContext, h *swf.HistoryEvent, d *TestData) Outcome {
		if d.States[0] != "marshalled" {
			t.Fail()
		}
		return f.Goto("ok", d, nil)
	}

	wrapped := typedFuncs.Decider(typedDecider)

	outcome := wrapped(&FSMContext{}, &swf.HistoryEvent{}, &TestData{States: []string{"marshalled"}})

	if outcome.State != "ok" {
		t.Fatal("NextState was not 'ok'")
	}
}

func TestPanicRecovery(t *testing.T) {
	s := &FSMState{
		Name: "panic",
		Decider: func(f *FSMContext, e *swf.HistoryEvent, data interface{}) Outcome {
			panic("can you handle it?")
		},
	}
	f := &FSM{}
	f.AddInitialState(s)
	_, err := f.panicSafeDecide(s, new(FSMContext), &swf.HistoryEvent{}, nil)
	if err == nil {
		t.Fatal("fatallz")
	} else {
		Log.Println(err)
	}
}

func ExampleFSM() {
	// create with swf.NewClient
	var client *swf.SWF
	// data type that will be managed by the FSM
	type StateData struct {
		Message string `json:"message,omitempty"`
		Count   int    `json:"count,omitempty"`
	}
	//event type that will be signalled to the FSM with signal name "hello"
	type Hello struct {
		Message string `json:"message,omitempty"`
	}

	//This is an example of building Deciders without using decider composition.
	//the FSM we will create will oscillate between 2 states,
	//waitForSignal -> will wait till the workflow is started or signalled, and update the StateData based on the Hello message received, set a timer, and transition to waitForTimer
	//waitForTimer -> will wait till the timer set by waitForSignal fires, and will signal the workflow with a Hello message, and transition to waitFotSignal
	waitForSignal := func(f *FSMContext, h *swf.HistoryEvent, d *StateData) Outcome {
		decisions := f.EmptyDecisions()
		switch *h.EventType {
		case swf.EventTypeWorkflowExecutionStarted, swf.EventTypeWorkflowExecutionSignaled:
			if *h.EventType == swf.EventTypeWorkflowExecutionSignaled && *h.WorkflowExecutionSignaledEventAttributes.SignalName == "hello" {
				hello := &Hello{}
				f.EventData(h, &Hello{})
				d.Count++
				d.Message = hello.Message
			}
			timeoutSeconds := "5" //swf uses stringy numbers in many places
			timerDecision := &swf.Decision{
				DecisionType: S(swf.DecisionTypeStartTimer),
				StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
					StartToFireTimeout: S(timeoutSeconds),
					TimerId:            S("timeToSignal"),
				},
			}
			decisions = append(decisions, timerDecision)
			return f.Goto("waitForTimer", d, decisions)
		}
		//if the event was unexpected just stay here
		return f.Stay(d, decisions)

	}

	waitForTimer := func(f *FSMContext, h *swf.HistoryEvent, d *StateData) Outcome {
		decisions := f.EmptyDecisions()
		switch *h.EventType {
		case swf.EventTypeTimerFired:
			//every time the timer fires, signal the workflow with a Hello
			message := strconv.FormatInt(time.Now().Unix(), 10)
			signalInput := &Hello{message}
			signalDecision := &swf.Decision{
				DecisionType: S(swf.DecisionTypeSignalExternalWorkflowExecution),
				SignalExternalWorkflowExecutionDecisionAttributes: &swf.SignalExternalWorkflowExecutionDecisionAttributes{
					SignalName: S("hello"),
					Input:      S(f.Serialize(signalInput)),
					RunId:      f.RunId,
					WorkflowId: f.WorkflowId,
				},
			}
			decisions = append(decisions, signalDecision)

			return f.Goto("waitForSignal", d, decisions)
		}
		//if the event was unexpected just stay here
		return f.Stay(d, decisions)
	}

	//These 2 deciders are the same as the ones above, but use composable decider bits.
	typed := Typed(new(StateData))

	updateState := typed.StateFunc(func(f *FSMContext, h *swf.HistoryEvent, d *StateData) {
		hello := &Hello{}
		f.EventData(h, &Hello{})
		d.Count++
		d.Message = hello.Message
	})

	setTimer := typed.DecisionFunc(func(f *FSMContext, h *swf.HistoryEvent, d *StateData) *swf.Decision {
		timeoutSeconds := "5" //swf uses stringy numbers in many places
		return &swf.Decision{
			DecisionType: S(swf.DecisionTypeStartTimer),
			StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
				StartToFireTimeout: S(timeoutSeconds),
				TimerId:            S("timeToSignal"),
			},
		}
	})

	sendSignal := typed.DecisionFunc(func(f *FSMContext, h *swf.HistoryEvent, d *StateData) *swf.Decision {
		message := strconv.FormatInt(time.Now().Unix(), 10)
		signalInput := &Hello{message}
		return &swf.Decision{
			DecisionType: S(swf.DecisionTypeSignalExternalWorkflowExecution),
			SignalExternalWorkflowExecutionDecisionAttributes: &swf.SignalExternalWorkflowExecutionDecisionAttributes{
				SignalName: S("hello"),
				Input:      S(f.Serialize(signalInput)),
				RunId:      f.RunId,
				WorkflowId: f.WorkflowId,
			},
		}
	})

	waitForSignalComposedDecider := NewComposedDecider(
		OnStarted(UpdateState(updateState), AddDecision(setTimer), Transition("waitForTimer")),
		OnSignalReceived("hello", UpdateState(updateState), AddDecision(setTimer), Transition("waitForTimer")),
		DefaultDecider(),
	)

	waitForTimerComposedDecider := NewComposedDecider(
		OnTimerFired("timeToSignal", AddDecision(sendSignal), Transition("waitForSignal")),
		DefaultDecider(),
	)

	//create the FSMState by passing the decider function through TypedDecider(),
	//which lets you use d *StateData rather than d interface{} in your decider.
	waitForSignalState := &FSMState{Name: "waitForSignal", Decider: typed.Decider(waitForSignal)}
	waitForTimerState := &FSMState{Name: "waitForTimer", Decider: typed.Decider(waitForTimer)}
	//or with the composed deciders
	waitForSignalState = &FSMState{Name: "waitForSignal", Decider: waitForSignalComposedDecider}
	waitForTimerState = &FSMState{Name: "waitForTimer", Decider: waitForTimerComposedDecider}
	//wire it up in an fsm
	fsm := &FSM{
		Name:       "example-fsm",
		SWF:        client,
		DataType:   StateData{},
		Domain:     "exaple-swf-domain",
		TaskList:   "example-decision-task-list-to-poll",
		Serializer: &JSONStateSerializer{},
	}
	//add states to FSM
	fsm.AddInitialState(waitForSignalState)
	fsm.AddState(waitForTimerState)

	//start it up!
	fsm.Start()

	//To start workflows using this fsm
	client.StartWorkflowExecution(&swf.StartWorkflowExecutionInput{
		Domain:     S("exaple-swf-domain"),
		WorkflowId: S("your-id"),
		//you will have previously regiestered a WorkflowType that this FSM will work.
		WorkflowType: &swf.WorkflowType{Name: S("the-name"), Version: S("the-version")},
		Input:        StartFSMWorkflowInput(fsm, &StateData{Count: 0, Message: "starting message"}),
	})
}

func TestContinuedWorkflows(t *testing.T) {
	fsm := testFSM()

	fsm.AddInitialState(&FSMState{
		Name: "ok",
		Decider: func(f *FSMContext, h *swf.HistoryEvent, d interface{}) Outcome {
			if *h.EventType == swf.EventTypeWorkflowExecutionStarted && d.(*TestData).States[0] == "continuing" {
				return f.Stay(d, nil)
			}
			panic("broken")
		},
	})

	stateData := fsm.Serialize(TestData{States: []string{"continuing"}})
	state := SerializedState{
		StateVersion: 23,
		StateName:    "ok",
		StateData:    stateData,
	}

	serializedState := fsm.Serialize(state)
	resp := testDecisionTask(4, []*swf.HistoryEvent{&swf.HistoryEvent{
		EventType: S(swf.EventTypeWorkflowExecutionStarted),
		WorkflowExecutionStartedEventAttributes: &swf.WorkflowExecutionStartedEventAttributes{
			Input: S(serializedState),
			ContinuedExecutionRunId: S("someRunId"),
		},
	}})

	Log.Printf("%+v", resp)
	_, decisions, updatedState, _ := fsm.Tick(resp)

	Log.Println(updatedState)

	if updatedState.StateVersion != 24 {
		t.Fatal("StateVersion !=24 ", updatedState.StateVersion)
	}

	if len(decisions) != 1 && *decisions[0].RecordMarkerDecisionAttributes.MarkerName != StateMarker {
		t.Fatal("unexpected decisions")
	}
}

func TestContinueWorkflowDecision(t *testing.T) {

	fsm := testFSM()
	ctx := testContext(fsm)
	ctx.stateVersion = uint64(7)
	ctx.stateData = &TestData{States: []string{"continuing"}}

	fsm.AddInitialState(&FSMState{
		Name: "InitialState",
		Decider: func(ctx *FSMContext, h *swf.HistoryEvent, data interface{}) Outcome {
			return ctx.Pass()
		},
	},
	)

	cont := ctx.ContinueWorkflowDecision("InitialState", ctx.stateData)
	testData := new(TestData)
	serState := new(SerializedState)
	ctx.Deserialize(*cont.ContinueAsNewWorkflowExecutionDecisionAttributes.Input, serState)
	ctx.Deserialize(serState.StateData, testData)
	if len(testData.States) != 1 || testData.States[0] != "continuing" || serState.StateVersion != 7 || serState.StateName != "InitialState" {
		t.Fatal(testData, cont)
	}

	tags := cont.ContinueAsNewWorkflowExecutionDecisionAttributes.TagList
	if len(tags) != 2 || *tags[0] != "tag1" || *tags[1] != "tag2" {
		t.Fatal(testData, cont)
	}

}

func TestCompleteState(t *testing.T) {
	fsm := testFSM()

	ctx := testContext(fsm)

	event := &swf.HistoryEvent{
		EventId:   I(1),
		EventType: S("WorkflowExecutionStarted"),
		WorkflowExecutionStartedEventAttributes: &swf.WorkflowExecutionStartedEventAttributes{
			Input: StartFSMWorkflowInput(fsm, new(TestData)),
		},
	}

	fsm.AddInitialState(fsm.DefaultCompleteState())
	fsm.Init()
	outcome := fsm.completeState.Decider(ctx, event, new(TestData))

	if len(outcome.Decisions) != 1 {
		t.Fatal(outcome)
	}

	if *outcome.Decisions[0].DecisionType != swf.DecisionTypeCompleteWorkflowExecution {
		t.Fatal(outcome)
	}
}

func TestFailState(t *testing.T) {
	fsm := testFSM()

	ctx := testContext(fsm)

	event := &swf.HistoryEvent{
		EventId:   I(1),
		EventType: S("WorkflowExecutionStarted"),
		WorkflowExecutionStartedEventAttributes: &swf.WorkflowExecutionStartedEventAttributes{
			Input: StartFSMWorkflowInput(fsm, new(TestData)),
		},
	}

	fsm.AddInitialState(fsm.DefaultFailedState())
	fsm.Init()
	outcome := fsm.failedState.Decider(ctx, event, new(TestData))

	if len(outcome.Decisions) != 1 {
		t.Fatal(outcome)
	}

	if *outcome.Decisions[0].DecisionType != swf.DecisionTypeFailWorkflowExecution {
		t.Fatal(outcome)
	}
}

func TestSerializationInterface(t *testing.T) {
	f := func(s Serialization) {

	}

	f(&FSM{})
	f(&FSMContext{})
}

func TestTaskReady(t *testing.T) {
	f := testFSM()
	prevStarted := testHistoryEvent(1, swf.EventTypeDecisionTaskStarted)
	missed := testHistoryEvent(2, swf.EventTypeWorkflowExecutionSignaled)
	missed.WorkflowExecutionSignaledEventAttributes = &swf.WorkflowExecutionSignaledEventAttributes{SignalName: S("Important")}
	state := testHistoryEvent(3, swf.EventTypeMarkerRecorded)
	state.MarkerRecordedEventAttributes = &swf.MarkerRecordedEventAttributes{MarkerName: S(StateMarker)}
	correlator := testHistoryEvent(4, swf.EventTypeMarkerRecorded)
	correlator.MarkerRecordedEventAttributes = &swf.MarkerRecordedEventAttributes{MarkerName: S(CorrelatorMarker)}
	task := testDecisionTask(1, []*swf.HistoryEvent{correlator, state})
	if f.taskReady(task) {
		t.Fatal("task signaled ready, and events were missed")
	}
	task.Events = append(task.Events, missed, prevStarted)
	if !f.taskReady(task) {
		t.Fatal("task not signaled ready, but state correlator and prevStarted were present")
	}
}

func TestStasher(t *testing.T) {

	mapIn := make(map[string]interface{})
	stasher := NewStasher(mapIn)
	buf := stasher.Stash(mapIn)
	stasher.Unstash(buf, &mapIn)

	in := &TestData{
		States: []string{"test123"},
	}

	stasher = NewStasher(&TestData{})
	//make a second to verify gob.Register doesnt panic on dupes.
	stasher = NewStasher(&TestData{})

	buf = stasher.Stash(in)

	out := &TestData{}
	stasher.Unstash(buf, out)

	if out.States[0] != "test123" {
		t.Fatal("bad stasher")
	}

}

func TestInitWhenTaskErrorHandlerNotSetExpectsDefaultUsed(t *testing.T) {
	// arrange
	f := testFSM()
	f.AddInitialState(f.DefaultCompleteState())

	// act
	f.Init()

	// assert
	assert.Equal(t, reflect.ValueOf(f.DefaultTaskErrorHandler).Pointer(), reflect.ValueOf(f.TaskErrorHandler).Pointer(),
		"Expected TaskErrorHandler to default to the DefaultTaskErrorHandler upon Init() if none is set")
}

func TestInitWhenTaskErrorHandlerSetExpectsSetFuncUsed(t *testing.T) {
	// arrange
	f := testFSM()
	f.AddInitialState(f.DefaultCompleteState())
	expectedHandler := func(decisionTask *swf.PollForDecisionTaskOutput, err error) {}
	f.TaskErrorHandler = expectedHandler

	// act
	f.Init()

	// assert
	assert.Equal(t, reflect.ValueOf(expectedHandler).Pointer(), reflect.ValueOf(f.TaskErrorHandler).Pointer(),
		"Expected FSM to use the set handler after Init()")
}

func TestHandleDecisionTaskWhenTickErrorsExpectsTaskErrorHandlerCalled(t *testing.T) {
	// arrange
	f := testFSM()
	f.AddInitialState(f.DefaultCompleteState())
	handlerCalled := false
	expectedHandler := func(decisionTask *swf.PollForDecisionTaskOutput, err error) {
		handlerCalled = true
	}
	f.TaskErrorHandler = expectedHandler
	events := []*swf.HistoryEvent{
		&swf.HistoryEvent{EventType: S("DecisionTaskStarted"), EventId: I(3)},
		&swf.HistoryEvent{EventType: S("DecisionTaskScheduled"), EventId: I(2)},
	}
	decisionTask := testDecisionTask(1, events)
	f.Init()
	f.AllowPanics = false

	// act
	f.handleDecisionTask(decisionTask)

	// assert
	assert.True(t, handlerCalled, "Expected handler called because Tick errored")
}

func TestHandleDecisionTaskWhenRespondingToSWFErrorsExpectsTaskErrorHandlerCalled(t *testing.T) {
	// arrange
	f := testFSM()
	f.AddInitialState(f.DefaultCompleteState())

	handlerCalled := false
	expectedHandler := func(decisionTask *swf.PollForDecisionTaskOutput, err error) {
		handlerCalled = true
	}
	f.TaskErrorHandler = expectedHandler

	events := []*swf.HistoryEvent{
		&swf.HistoryEvent{EventType: S("DecisionTaskStarted"), EventId: I(3)},
		&swf.HistoryEvent{EventType: S("DecisionTaskScheduled"), EventId: I(2)},
		EventFromPayload(1, &swf.WorkflowExecutionStartedEventAttributes{
			Input: StartFSMWorkflowInput(f, new(TestData)),
		}),
	}
	decisionTask := testDecisionTask(0, events)

	f.AllowPanics = false
	mockSWFAPI := &mocks.SWFAPI{}
	expectedError := errors.New("Some SWF error")
	mockSWFAPI.MockOn_RespondDecisionTaskCompleted(mock.Anything).Return(nil, expectedError)
	f.SWF = mockSWFAPI

	// act
	f.Init()
	f.handleDecisionTask(decisionTask)

	// assert
	assert.True(t, handlerCalled, "Expected handler called because RespondDecisionTaskCompleted errored")
}

func TestHandleDecisionTaskReplicationErrorsExpectsTaskErrorHandlerCalled(t *testing.T) {
	// arrange
	f := testFSM()
	f.AddInitialState(f.DefaultCompleteState())

	handlerCalled := false
	expectedHandler := func(decisionTask *swf.PollForDecisionTaskOutput, err error) {
		handlerCalled = true
	}
	f.TaskErrorHandler = expectedHandler

	events := []*swf.HistoryEvent{
		&swf.HistoryEvent{EventType: S("DecisionTaskStarted"), EventId: I(3)},
		&swf.HistoryEvent{EventType: S("DecisionTaskScheduled"), EventId: I(2)},
		EventFromPayload(1, &swf.WorkflowExecutionStartedEventAttributes{
			Input: StartFSMWorkflowInput(f, new(TestData)),
		}),
	}
	decisionTask := testDecisionTask(0, events)

	f.AllowPanics = false
	mockSWFAPI := &mocks.SWFAPI{}
	mockSWFAPI.MockOn_RespondDecisionTaskCompleted(mock.Anything).Return(nil, nil)
	f.SWF = mockSWFAPI
	f.ReplicationHandler = func(*FSMContext, *swf.PollForDecisionTaskOutput, *swf.RespondDecisionTaskCompletedInput, *SerializedState) error {
		return errors.New("Some replication error")
	}

	// act
	f.Init()
	f.handleDecisionTask(decisionTask)

	// assert
	assert.True(t, handlerCalled, "Expected handler called because there was a replication error")
}

func TestHandleDecisionTaskWhenNoErrorsExpectsTaskErrorHandlerNotCalled(t *testing.T) {
	// arrange
	f := testFSM()
	f.AddInitialState(f.DefaultCompleteState())

	handlerCalled := false
	expectedHandler := func(decisionTask *swf.PollForDecisionTaskOutput, err error) {
		handlerCalled = true
	}
	f.TaskErrorHandler = expectedHandler

	events := []*swf.HistoryEvent{
		&swf.HistoryEvent{EventType: S("DecisionTaskStarted"), EventId: I(3)},
		&swf.HistoryEvent{EventType: S("DecisionTaskScheduled"), EventId: I(2)},
		EventFromPayload(1, &swf.WorkflowExecutionStartedEventAttributes{
			Input: StartFSMWorkflowInput(f, new(TestData)),
		}),
	}
	decisionTask := testDecisionTask(0, events)

	mockSWFAPI := &mocks.SWFAPI{}
	mockSWFAPI.MockOn_RespondDecisionTaskCompleted(mock.Anything).Return(nil, nil)
	f.SWF = mockSWFAPI
	f.ReplicationHandler = func(*FSMContext, *swf.PollForDecisionTaskOutput, *swf.RespondDecisionTaskCompletedInput, *SerializedState) error {
		return nil
	}

	// act
	f.Init()
	f.handleDecisionTask(decisionTask)

	// assert
	assert.False(t, handlerCalled, "Expected handler not called because nothing errored")
}

func testFSM() *FSM {
	fsm := &FSM{
		Name:             "test-fsm",
		DataType:         TestData{},
		Serializer:       JSONStateSerializer{},
		SystemSerializer: JSONStateSerializer{},
		AllowPanics:      true,
	}
	return fsm
}

func testContext(fsm *FSM) *FSMContext {
	return NewFSMContext(
		fsm,
		swf.WorkflowType{Name: S("test-workflow"), Version: S("1")},
		swf.WorkflowExecution{WorkflowId: S("test-workflow-1"), RunId: S("123123")},
		&EventCorrelator{Serializer: JSONStateSerializer{}},
		"InitialState", &TestData{}, 0,
	)
}

func testDecisionTask(prevStarted int, events []*swf.HistoryEvent) *swf.PollForDecisionTaskOutput {

	d := &swf.PollForDecisionTaskOutput{
		Events:                 events,
		PreviousStartedEventId: I(prevStarted),
		StartedEventId:         I(prevStarted + len(events)),
		WorkflowExecution:      testWorkflowExecution,
		WorkflowType:           testWorkflowType,
	}
	for i, e := range d.Events {
		if e.EventId == nil {
			e.EventId = L(*d.StartedEventId - int64(i))
		}
		e.EventTimestamp = aws.Time(time.Unix(0, 0))
		d.Events[i] = e
	}
	return d
}

func testHistoryEvent(eventId int, eventType string) *swf.HistoryEvent {
	return &swf.HistoryEvent{
		EventId:   I(eventId),
		EventType: S(eventType),
	}
}

var testWorkflowExecution = &swf.WorkflowExecution{WorkflowId: S("workflow-id"), RunId: S("run-id")}
var testWorkflowType = &swf.WorkflowType{Name: S("workflow-name"), Version: S("workflow-version")}
