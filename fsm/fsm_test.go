package fsm

import (
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"strconv"
	"testing"
	"time"

	"code.google.com/p/goprotobuf/proto"
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/kinesis"
	"github.com/awslabs/aws-sdk-go/gen/swf"
	. "github.com/sclasen/swf-go/sugar"
)

//Todo add tests of error handling mechanism
//assert that the decisions have the mark and the signal external...hmm need workflow id for signal external.

var testActivityInfo = ActivityInfo{ActivityID: "activityId", ActivityType: &swf.ActivityType{Name: S("activity"), Version: S("activityVersion")}}

var typedFuncs = Typed(new(TestData))

func TestFSM(t *testing.T) {

	fsm := testFSM()

	fsm.AddInitialState(&FSMState{
		Name: "start",
		Decider: func(f *FSMContext, lastEvent swf.HistoryEvent, data interface{}) Outcome {
			testData := data.(*TestData)
			testData.States = append(testData.States, "start")
			serialized := f.Serialize(testData)
			decision := swf.Decision{
				DecisionType: S(swf.DecisionTypeScheduleActivityTask),
				ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
					ActivityID:   S(testActivityInfo.ActivityID),
					ActivityType: testActivityInfo.ActivityType,
					TaskList:     &swf.TaskList{Name: S("taskList")},
					Input:        S(serialized),
				},
			}

			return f.Goto("working", testData, []swf.Decision{decision})

		},
	})

	fsm.AddState(&FSMState{
		Name: "working",
		Decider: typedFuncs.Decider(func(f *FSMContext, lastEvent swf.HistoryEvent, testData *TestData) Outcome {
			testData.States = append(testData.States, "working")
			serialized := f.Serialize(testData)
			var decisions = f.EmptyDecisions()
			if *lastEvent.EventType == swf.EventTypeActivityTaskCompleted {
				decision := swf.Decision{
					DecisionType: aws.String(swf.DecisionTypeCompleteWorkflowExecution),
					CompleteWorkflowExecutionDecisionAttributes: &swf.CompleteWorkflowExecutionDecisionAttributes{
						Result: S(serialized),
					},
				}
				decisions = append(decisions, decision)
			} else if *lastEvent.EventType == swf.EventTypeActivityTaskFailed {
				decision := swf.Decision{
					DecisionType: aws.String(swf.DecisionTypeScheduleActivityTask),
					ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
						ActivityID:   S(testActivityInfo.ActivityID),
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

	events := []swf.HistoryEvent{
		swf.HistoryEvent{EventType: S("DecisionTaskStarted"), EventID: I(3)},
		swf.HistoryEvent{EventType: S("DecisionTaskScheduled"), EventID: I(2)},
		swf.HistoryEvent{
			EventID:   I(1),
			EventType: S("WorkflowExecutionStarted"),
			WorkflowExecutionStartedEventAttributes: &swf.WorkflowExecutionStartedEventAttributes{
				Input: S(StartFSMWorkflowInput(fsm.Serializer, new(TestData))),
			},
		},
	}

	first := testDecisionTask(0, events)

	decisions, _ := fsm.Tick(first)

	if !Find(decisions, stateMarkerPredicate) {
		t.Fatal("No Record State Marker")
	}

	if !Find(decisions, scheduleActivityPredicate) {
		t.Fatal("No ScheduleActivityTask")
	}

	secondEvents := DecisionsToEvents(decisions)
	secondEvents = append(secondEvents, events...)

	if state, _ := fsm.findSerializedState(secondEvents); state.ReplicationData.StateName != "working" {
		t.Fatal("current state is not 'working'", secondEvents)
	}

	second := testDecisionTask(3, secondEvents)

	secondDecisions, _ := fsm.Tick(second)

	if !Find(secondDecisions, stateMarkerPredicate) {
		t.Fatal("No Record State Marker")
	}

	if !Find(secondDecisions, completeWorkflowPredicate) {
		t.Fatal("No CompleteWorkflow")
	}

}

func Find(decisions []swf.Decision, predicate func(swf.Decision) bool) bool {
	return FindDecision(decisions, predicate) != nil
}

func FindDecision(decisions []swf.Decision, predicate func(swf.Decision) bool) *swf.Decision {
	for _, d := range decisions {
		if predicate(d) {
			return &d
		}
	}
	return nil
}

func stateMarkerPredicate(d swf.Decision) bool {
	return *d.DecisionType == "RecordMarker" && *d.RecordMarkerDecisionAttributes.MarkerName == StateMarker
}

func scheduleActivityPredicate(d swf.Decision) bool {
	return *d.DecisionType == "ScheduleActivityTask"
}

func completeWorkflowPredicate(d swf.Decision) bool {
	return *d.DecisionType == "CompleteWorkflowExecution"
}

func startTimerPredicate(d swf.Decision) bool {
	return *d.DecisionType == "StartTimer"
}

func DecisionsToEvents(decisions []swf.Decision) []swf.HistoryEvent {
	var events []swf.HistoryEvent
	for _, d := range decisions {
		if scheduleActivityPredicate(d) {
			event := swf.HistoryEvent{
				EventType: S("ActivityTaskCompleted"),
				EventID:   I(7),
				ActivityTaskCompletedEventAttributes: &swf.ActivityTaskCompletedEventAttributes{
					ScheduledEventID: I(6),
				},
			}
			events = append(events, event)
			event = swf.HistoryEvent{
				EventType: S("ActivityTaskScheduled"),
				EventID:   I(6),
				ActivityTaskScheduledEventAttributes: &swf.ActivityTaskScheduledEventAttributes{
					ActivityID:   S(testActivityInfo.ActivityID),
					ActivityType: testActivityInfo.ActivityType,
				},
			}
			events = append(events, event)
		}
		if stateMarkerPredicate(d) {
			event := swf.HistoryEvent{
				EventType: S("MarkerRecorded"),
				EventID:   I(5),
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

func TestMarshalledDecider(t *testing.T) {
	typedDecider := func(f *FSMContext, h swf.HistoryEvent, d *TestData) Outcome {
		if d.States[0] != "marshalled" {
			t.Fail()
		}
		return f.Goto("ok", d, nil)
	}

	wrapped := typedFuncs.Decider(typedDecider)

	outcome := wrapped(&FSMContext{}, swf.HistoryEvent{}, &TestData{States: []string{"marshalled"}})

	if outcome.(TransitionOutcome).state != "ok" {
		t.Fatal("NextState was not 'ok'")
	}
}

func TestPanicRecovery(t *testing.T) {
	s := &FSMState{
		Name: "panic",
		Decider: func(f *FSMContext, e swf.HistoryEvent, data interface{}) Outcome {
			panic("can you handle it?")
		},
	}
	f := &FSM{}
	f.AddInitialState(s)
	_, err := f.panicSafeDecide(s, new(FSMContext), swf.HistoryEvent{}, nil)
	if err == nil {
		t.Fatal("fatallz")
	} else {
		log.Println(err)
	}
}

func TestProtobufSerialization(t *testing.T) {
	ser := &ProtobufStateSerializer{}
	key := "FOO"
	val := "BAR"
	init := &ConfigVar{Key: &key, Str: &val}
	serialized, err := ser.Serialize(init)
	if err != nil {
		t.Fatal(err)
	}

	log.Println(serialized)

	deserialized := new(ConfigVar)
	err = ser.Deserialize(serialized, deserialized)
	if err != nil {
		t.Fatal(err)
	}

	if init.GetKey() != deserialized.GetKey() {
		t.Fatal(init, deserialized)
	}

	if init.GetStr() != deserialized.GetStr() {
		t.Fatal(init, deserialized)
	}
}

//This is c&p from som generated code

type ConfigVar struct {
	Key             *string `protobuf:"bytes,1,req,name=key" json:"key,omitempty"`
	Str             *string `protobuf:"bytes,2,opt,name=str" json:"str,omitempty"`
	XXXunrecognized []byte  `json:"-"`
}

func (m *ConfigVar) Reset()         { *m = ConfigVar{} }
func (m *ConfigVar) String() string { return proto.CompactTextString(m) }
func (*ConfigVar) ProtoMessage()    {}

func (m *ConfigVar) GetKey() string {
	if m != nil && m.Key != nil {
		return *m.Key
	}
	return ""
}

func (m *ConfigVar) GetStr() string {
	if m != nil && m.Str != nil {
		return *m.Str
	}
	return ""
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
	//the FSM we will create will oscillate between 2 states,
	//waitForSignal -> will wait till the workflow is started or signalled, and update the StateData based on the Hello message received, set a timer, and transition to waitForTimer
	//waitForTimer -> will wait till the timer set by waitForSignal fires, and will signal the workflow with a Hello message, and transition to waitFotSignal
	waitForSignal := func(f *FSMContext, h swf.HistoryEvent, d *StateData) Outcome {
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
			timerDecision := swf.Decision{
				DecisionType: S(swf.DecisionTypeStartTimer),
				StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
					StartToFireTimeout: S(timeoutSeconds),
					TimerID:            S("timeToSignal"),
				},
			}
			decisions = append(decisions, timerDecision)
			return f.Goto("waitForTimer", d, decisions)
		}
		//if the event was unexpected just stay here
		return f.Stay(d, decisions)

	}

	waitForTimer := func(f *FSMContext, h swf.HistoryEvent, d *StateData) Outcome {
		decisions := f.EmptyDecisions()
		switch *h.EventType {
		case swf.EventTypeTimerFired:
			//every time the timer fires, signal the workflow with a Hello
			message := strconv.FormatInt(time.Now().Unix(), 10)
			signalInput := &Hello{message}
			signalDecision := swf.Decision{
				DecisionType: S(swf.DecisionTypeSignalExternalWorkflowExecution),
				SignalExternalWorkflowExecutionDecisionAttributes: &swf.SignalExternalWorkflowExecutionDecisionAttributes{
					SignalName: S("hello"),
					Input:      S(f.Serialize(signalInput)),
					RunID:      f.RunID,
					WorkflowID: f.WorkflowID,
				},
			}
			decisions = append(decisions, signalDecision)

			return f.Goto("waitForSignal", d, decisions)
		}
		//if the event was unexpected just stay here
		return f.Stay(d, decisions)
	}

	//create the FSMState by passing the decider function through TypedDecider(),
	//which lets you use d *StateData rather than d interface{} in your decider.
	waitForSignalState := &FSMState{Name: "waitForSignal", Decider: typedFuncs.Decider(waitForSignal)}
	waitForTimerState := &FSMState{Name: "waitForTimer", Decider: typedFuncs.Decider(waitForTimer)}
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
		WorkflowID: S("your-id"),
		//you will have previously regiestered a WorkflowType that this FSM will work.
		WorkflowType: &swf.WorkflowType{Name: S("the-name"), Version: S("the-version")},
		// It is *very* important to use StartFSMWorkflowInput so the state management works properly
		Input: S(StartFSMWorkflowInput(fsm.Serializer, &StateData{Count: 0, Message: "starting message"})),
	})
}

func TestContinuedWorkflows(t *testing.T) {
	fsm := testFSM()

	fsm.AddInitialState(&FSMState{
		Name: "ok",
		Decider: func(f *FSMContext, h swf.HistoryEvent, d interface{}) Outcome {
			if *h.EventType == swf.EventTypeWorkflowExecutionStarted && d.(*TestData).States[0] == "continuing" {
				return f.Stay(d, nil)
			}
			panic("broken")
		},
	})

	stateData := fsm.Serialize(TestData{States: []string{"continuing"}})
	state := SerializedState{
		ReplicationData: ReplicationData{
			StateVersion: 23,
			StateName:    "ok",
			StateData:    stateData,
		},
		EventCorrelator: EventCorrelator{},
	}
	serializedState := fsm.Serialize(state)
	resp := testDecisionTask(4, []swf.HistoryEvent{swf.HistoryEvent{
		EventType: S(swf.EventTypeWorkflowExecutionStarted),
		WorkflowExecutionStartedEventAttributes: &swf.WorkflowExecutionStartedEventAttributes{
			Input: S(serializedState),
		},
	}})

	log.Printf("%+v", resp)
	decisions, updatedState := fsm.Tick(resp)

	log.Println(updatedState)

	if updatedState.ReplicationData.StateVersion != 24 {
		t.Fatal("StateVersion !=24 ", updatedState.ReplicationData.StateVersion)
	}

	if len(decisions) != 1 && *decisions[0].RecordMarkerDecisionAttributes.MarkerName != StateMarker {
		t.Fatal("unexpected decisions")
	}
}

func TestKinesisReplication(t *testing.T) {
	client := &MockClient{}
	fsm := testFSM()
	fsm.SWF = client
	fsm.Kinesis = client
	fsm.AddInitialState(&FSMState{
		Name: "initial",
		Decider: func(f *FSMContext, h swf.HistoryEvent, d interface{}) Outcome {
			if *h.EventType == swf.EventTypeWorkflowExecutionStarted {
				return f.Goto("done", d, f.EmptyDecisions())
			}
			t.Fatal("unexpected")
			return nil // unreachable
		},
	})
	fsm.AddState(&FSMState{
		Name: "done",
		Decider: func(f *FSMContext, h swf.HistoryEvent, d interface{}) Outcome {
			go fsm.PollerShutdownManager.StopPollers()
			return f.Stay(d, f.EmptyDecisions())
		},
	})
	events := []swf.HistoryEvent{
		swf.HistoryEvent{EventType: S("DecisionTaskStarted"), EventID: I(3)},
		swf.HistoryEvent{EventType: S("DecisionTaskScheduled"), EventID: I(2)},
		swf.HistoryEvent{
			EventID:   I(1),
			EventType: S("WorkflowExecutionStarted"),
			WorkflowExecutionStartedEventAttributes: &swf.WorkflowExecutionStartedEventAttributes{
				Input: S(StartFSMWorkflowInput(fsm.Serializer, new(TestData))),
			},
		},
	}
	decisionTask := testDecisionTask(0, events)

	fsm.handleDecisionTask(decisionTask)

	if client.putRecords == nil || len(client.putRecords) != 1 {
		t.Fatalf("expected only one state to be replicated, got: %v", client.putRecords)
	}
	replication := client.putRecords[0]
	if *replication.StreamName != fsm.KinesisStream {
		t.Fatalf("expected Kinesis stream: %q, got %q", fsm.KinesisStream, replication.StreamName)
	}
	var replicatedState ReplicationData
	if err := fsm.Serializer.Deserialize(string(replication.Data), &replicatedState); err != nil {
		t.Fatal(err)
	}
	if replicatedState.StateVersion != 1 {
		t.Fatalf("state.StateVersion != 1, got: %d", replicatedState.StateVersion)
	}
	if replicatedState.StateName != "done" {
		t.Fatalf("current state being replicated is not 'done', got %q", replicatedState.StateName)
	}
}

func TestTrackPendingActivities(t *testing.T) {
	fsm := testFSM()

	fsm.AddInitialState(&FSMState{
		Name: "start",
		Decider: func(f *FSMContext, lastEvent swf.HistoryEvent, data interface{}) Outcome {
			testData := data.(*TestData)
			testData.States = append(testData.States, "start")
			serialized := f.Serialize(testData)
			decision := swf.Decision{
				DecisionType: S(swf.DecisionTypeScheduleActivityTask),
				ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
					ActivityID:   S(testActivityInfo.ActivityID),
					ActivityType: testActivityInfo.ActivityType,
					TaskList:     &swf.TaskList{Name: S("taskList")},
					Input:        S(serialized),
				},
			}
			return f.Goto("working", testData, []swf.Decision{decision})
		},
	})

	// Deciders should be able to retrieve info about the pending activity
	fsm.AddState(&FSMState{
		Name: "working",
		Decider: typedFuncs.Decider(func(f *FSMContext, lastEvent swf.HistoryEvent, testData *TestData) Outcome {
			testData.States = append(testData.States, "working")
			serialized := f.Serialize(testData)
			var decisions = f.EmptyDecisions()
			if *lastEvent.EventType == swf.EventTypeActivityTaskCompleted {
				trackedActivityInfo := f.ActivityInfo(lastEvent)
				log.Printf("----->>>>> %+v %+v", trackedActivityInfo, testActivityInfo)
				if !reflect.DeepEqual(*trackedActivityInfo, testActivityInfo) {
					t.Fatalf("pending activity not being tracked\nExpected:\n%+v\nGot:\n%+v",
						testActivityInfo, trackedActivityInfo,
					)
				}
				timeoutSeconds := "5" //swf uses stringy numbers in many places
				decision := swf.Decision{
					DecisionType: S(swf.DecisionTypeStartTimer),
					StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
						StartToFireTimeout: S(timeoutSeconds),
						TimerID:            S("timeToComplete"),
					},
				}
				return f.Goto("done", testData, []swf.Decision{decision})
			} else if *lastEvent.EventType == swf.EventTypeActivityTaskFailed {
				trackedActivityInfo := f.ActivityInfo(lastEvent)
				log.Printf("----->>>>> %+v %+v %+v", trackedActivityInfo, testActivityInfo, f.ActivitiesInfo())
				if !reflect.DeepEqual(*trackedActivityInfo, testActivityInfo) {
					t.Fatalf("pending activity not being tracked\nExpected:\n%+v\nGot:\n%+v",
						testActivityInfo, trackedActivityInfo,
					)
				}
				decision := swf.Decision{
					DecisionType: S(swf.DecisionTypeScheduleActivityTask),
					ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
						ActivityID:   S(testActivityInfo.ActivityID),
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

	// Pending activities are cleared after finished
	fsm.AddState(&FSMState{
		Name: "done",
		Decider: typedFuncs.Decider(func(f *FSMContext, lastEvent swf.HistoryEvent, testData *TestData) Outcome {
			decisions := f.EmptyDecisions()
			if *lastEvent.EventType == swf.EventTypeTimerFired {
				testData.States = append(testData.States, "done")
				serialized := f.Serialize(testData)
				trackedActivityInfo := f.ActivityInfo(lastEvent)
				if trackedActivityInfo != nil {
					t.Fatalf("pending activity not being cleared\nGot:\n%+v", trackedActivityInfo)
				}
				decision := swf.Decision{
					DecisionType: S(swf.DecisionTypeCompleteWorkflowExecution),
					CompleteWorkflowExecutionDecisionAttributes: &swf.CompleteWorkflowExecutionDecisionAttributes{
						Result: S(serialized),
					},
				}
				decisions = append(decisions, decision)
			}
			return f.Stay(testData, decisions)
		}),
	})

	// Schedule a task
	events := []swf.HistoryEvent{
		swf.HistoryEvent{EventType: S("DecisionTaskStarted"), EventID: I(3)},
		swf.HistoryEvent{EventType: S("DecisionTaskScheduled"), EventID: I(2)},
		swf.HistoryEvent{
			EventID:   I(1),
			EventType: S("WorkflowExecutionStarted"),
			WorkflowExecutionStartedEventAttributes: &swf.WorkflowExecutionStartedEventAttributes{
				Input: S(StartFSMWorkflowInput(fsm.Serializer, new(TestData))),
			},
		},
	}
	first := testDecisionTask(0, events)

	decisions, _ := fsm.Tick(first)
	recordMarker := FindDecision(decisions, stateMarkerPredicate)
	if recordMarker == nil {
		t.Fatal("No Record State Marker")
	}
	if !Find(decisions, scheduleActivityPredicate) {
		t.Fatal("No ScheduleActivityTask")
	}

	// Fail the task
	secondEvents := []swf.HistoryEvent{
		{
			EventType: S("ActivityTaskFailed"),
			EventID:   I(7),
			ActivityTaskFailedEventAttributes: &swf.ActivityTaskFailedEventAttributes{
				ScheduledEventID: I(6),
			},
		},
		{
			EventType: S("ActivityTaskScheduled"),
			EventID:   I(6),
			ActivityTaskScheduledEventAttributes: &swf.ActivityTaskScheduledEventAttributes{
				ActivityID:   S(testActivityInfo.ActivityID),
				ActivityType: testActivityInfo.ActivityType,
			},
		},
		{
			EventType: S("MarkerRecorded"),
			EventID:   I(5),
			MarkerRecordedEventAttributes: &swf.MarkerRecordedEventAttributes{
				MarkerName: S(StateMarker),
				Details:    recordMarker.RecordMarkerDecisionAttributes.Details,
			},
		},
	}
	secondEvents = append(secondEvents, events...)
	if state, _ := fsm.findSerializedState(secondEvents); state.ReplicationData.StateName != "working" {
		t.Fatal("current state is not 'working'", secondEvents)
	}
	second := testDecisionTask(3, secondEvents)

	secondDecisions, _ := fsm.Tick(second)
	recordMarker = FindDecision(secondDecisions, stateMarkerPredicate)
	if recordMarker == nil {
		t.Fatal("No Record State Marker")
	}
	if !Find(secondDecisions, scheduleActivityPredicate) {
		t.Fatal("No ScheduleActivityTask (retry)")
	}

	// Complete the task
	thirdEvents := []swf.HistoryEvent{
		{
			EventType: S("ActivityTaskCompleted"),
			EventID:   I(11),
			ActivityTaskCompletedEventAttributes: &swf.ActivityTaskCompletedEventAttributes{
				ScheduledEventID: I(10),
			},
		},
		{
			EventType: S("ActivityTaskScheduled"),
			EventID:   I(10),
			ActivityTaskScheduledEventAttributes: &swf.ActivityTaskScheduledEventAttributes{
				ActivityID:   S(testActivityInfo.ActivityID),
				ActivityType: testActivityInfo.ActivityType,
			},
		},
		{
			EventType: S("MarkerRecorded"),
			EventID:   I(9),
			MarkerRecordedEventAttributes: &swf.MarkerRecordedEventAttributes{
				MarkerName: S(StateMarker),
				Details:    recordMarker.RecordMarkerDecisionAttributes.Details,
			},
		},
	}
	thirdEvents = append(thirdEvents, secondEvents...)
	if state, _ := fsm.findSerializedState(thirdEvents); state.ReplicationData.StateName != "working" {
		t.Fatal("current state is not 'working'", thirdEvents)
	}
	third := testDecisionTask(7, thirdEvents)
	thirdDecisions, _ := fsm.Tick(third)
	recordMarker = FindDecision(thirdDecisions, stateMarkerPredicate)
	if recordMarker == nil {
		t.Fatal("No Record State Marker")
	}
	if !Find(thirdDecisions, startTimerPredicate) {
		t.Fatal("No StartTimer")
	}

	// Finish the workflow, check if pending activities were cleared
	fourthEvents := []swf.HistoryEvent{
		{
			EventType: S("TimerFired"),
			EventID:   I(14),
		},
		{
			EventType: S("MarkerRecorded"),
			EventID:   I(13),
			MarkerRecordedEventAttributes: &swf.MarkerRecordedEventAttributes{
				MarkerName: S(StateMarker),
				Details:    recordMarker.RecordMarkerDecisionAttributes.Details,
			},
		},
	}
	fourthEvents = append(fourthEvents, thirdEvents...)
	if state, _ := fsm.findSerializedState(fourthEvents); state.ReplicationData.StateName != "done" {
		t.Fatal("current state is not 'done'", fourthEvents)
	}
	fourth := testDecisionTask(11, fourthEvents)
	fourthDecisions, _ := fsm.Tick(fourth)
	recordMarker = FindDecision(fourthDecisions, stateMarkerPredicate)
	if recordMarker == nil {
		t.Fatal("No Record State Marker")
	}
	if !Find(fourthDecisions, completeWorkflowPredicate) {
		t.Fatal("No CompleteWorkflow")
	}
}

func TestFSMContextActivityTracking(t *testing.T) {
	ctx := testContext(testFSM())
	scheduledEventID := rand.Int()
	activityID := fmt.Sprintf("test-activity-%d", scheduledEventID)
	taskScheduled := swf.HistoryEvent{
		EventType: S("ActivityTaskScheduled"),
		EventID:   I(scheduledEventID),
		ActivityTaskScheduledEventAttributes: &swf.ActivityTaskScheduledEventAttributes{
			ActivityID: S(activityID),
			ActivityType: &swf.ActivityType{
				Name:    S("test-activity"),
				Version: S("1"),
			},
		},
	}
	ctx.Decide(taskScheduled, &TestData{}, func(ctx *FSMContext, h swf.HistoryEvent, data interface{}) Outcome {
		if len(ctx.ActivitiesInfo()) != 0 {
			t.Fatal("There should be no activies being tracked yet")
		}
		if !reflect.DeepEqual(h, taskScheduled) {
			t.Fatal("Got an unexpected event")
		}
		return ctx.Stay(data, ctx.EmptyDecisions())
	})
	if len(ctx.ActivitiesInfo()) < 1 {
		t.Fatal("Pending activity task is not being tracked")
	}

	// the pending activity can now be retrieved
	taskCompleted := swf.HistoryEvent{
		EventType: S("ActivityTaskCompleted"),
		EventID:   I(rand.Int()),
		ActivityTaskCompletedEventAttributes: &swf.ActivityTaskCompletedEventAttributes{
			ScheduledEventID: I(scheduledEventID),
		},
	}
	taskFailed := swf.HistoryEvent{
		EventType: S("ActivityTaskFailed"),
		EventID:   I(rand.Int()),
		ActivityTaskFailedEventAttributes: &swf.ActivityTaskFailedEventAttributes{
			ScheduledEventID: I(scheduledEventID),
		},
	}
	infoOnCompleted := ctx.ActivityInfo(taskCompleted)
	infoOnFailed := ctx.ActivityInfo(taskFailed)
	if infoOnCompleted.ActivityID != activityID ||
		*infoOnCompleted.Name != "test-activity" ||
		*infoOnCompleted.Version != "1" {
		t.Fatal("Pending activity can not be retrieved when completed")
	}
	if infoOnFailed.ActivityID != activityID ||
		*infoOnFailed.Name != "test-activity" ||
		*infoOnFailed.Version != "1" {
		t.Fatal("Pending activity can not be retrieved when failed")
	}

	// pending activities are also cleared after terminated
	ctx.Decide(taskCompleted, &TestData{}, func(ctx *FSMContext, h swf.HistoryEvent, data interface{}) Outcome {
		if len(ctx.ActivitiesInfo()) != 1 {
			t.Fatal("There should be one activity being tracked")
		}
		if !reflect.DeepEqual(h, taskCompleted) {
			t.Fatal("Got an unexpected event")
		}
		infoOnCompleted := ctx.ActivityInfo(taskCompleted)
		if infoOnCompleted.ActivityID != activityID ||
			*infoOnCompleted.Name != "test-activity" ||
			*infoOnCompleted.Version != "1" {
			t.Fatal("Pending activity can not be retrieved when completed")
		}
		return ctx.Stay(data, ctx.EmptyDecisions())
	})
	if len(ctx.ActivitiesInfo()) > 0 {
		t.Fatal("Pending activity task is not being cleared after completed")
	}
}

func TestContinueWorkflowDecision(t *testing.T) {

	fsm := testFSM()
	ctx := testContext(fsm)
	ctx.stateVersion = uint64(7)
	ctx.stateData = &TestData{States: []string{"continuing"}}

	fsm.AddInitialState(&FSMState{
		Name: "InitialState",
		Decider: func(ctx *FSMContext, h swf.HistoryEvent, data interface{}) Outcome {
			return nil
		},
	},
	)

	cont := ctx.ContinueWorkflowDecision("InitialState")
	testData := new(TestData)
	serState := new(SerializedState)
	ctx.Deserialize(*cont.ContinueAsNewWorkflowExecutionDecisionAttributes.Input, serState)
	ctx.Deserialize(serState.ReplicationData.StateData, testData)
	if len(testData.States) != 1 || testData.States[0] != "continuing" || serState.ReplicationData.StateVersion != 7 || serState.ReplicationData.StateName != "InitialState" {
		t.Fatal(testData, cont)
	}

}

func TestCompleteState(t *testing.T) {
	fsm := testFSM()

	ctx := testContext(fsm)

	event := swf.HistoryEvent{
		EventID:   I(1),
		EventType: S("WorkflowExecutionStarted"),
		WorkflowExecutionStartedEventAttributes: &swf.WorkflowExecutionStartedEventAttributes{
			Input: S(StartFSMWorkflowInput(ctx.Serializer(), new(TestData))),
		},
	}

	fsm.AddInitialState(fsm.DefaultCompleteState())
	fsm.Init()
	outcome := fsm.completeState.Decider(ctx, event, new(TestData))

	if len(outcome.Decisions()) != 1 {
		t.Fatal(outcome)
	}

	if *outcome.Decisions()[0].DecisionType != swf.DecisionTypeCompleteWorkflowExecution {
		t.Fatal(outcome)
	}
}

func testFSM() *FSM {
	fsm := &FSM{
		Name:              "test-fsm",
		DataType:          TestData{},
		KinesisStream:     "test-stream",
		Serializer:        JSONStateSerializer{},
		systemSerializer:  JSONStateSerializer{},
		KinesisReplicator: defaultKinesisReplicator(),
		allowPanics:       true,
	}
	return fsm
}

func testContext(fsm *FSM) *FSMContext {
	return NewFSMContext(
		fsm,
		swf.WorkflowType{Name: S("test-workflow"), Version: S("1")},
		swf.WorkflowExecution{WorkflowID: S("test-workflow-1"), RunID: S("123123")},
		&EventCorrelator{},
		"InitialState", &TestData{}, 0,
	)
}

func testDecisionTask(prevStarted int, events []swf.HistoryEvent) *swf.DecisionTask {

	d := &swf.DecisionTask{
		Events:                 events,
		PreviousStartedEventID: I(prevStarted),
		StartedEventID:         I(prevStarted + len(events)),
		WorkflowExecution:      testWorkflowExecution,
		WorkflowType:           testWorkflowType,
	}
	for i, e := range d.Events {
		if e.EventID == nil {
			e.EventID = L(*d.StartedEventID - int64(i))
		}
		e.EventTimestamp = &aws.UnixTimestamp{time.Unix(0, 0)}
		d.Events[i] = e
	}
	return d
}

var testWorkflowExecution = &swf.WorkflowExecution{WorkflowID: S("workflow-id"), RunID: S("run-id")}
var testWorkflowType = &swf.WorkflowType{Name: S("workflow-name"), Version: S("workflow-version")}

type MockClient struct {
	*kinesis.Kinesis
	*swf.SWF
	putRecords []kinesis.PutRecordInput
	seqNumber  int
}

func (c *MockClient) PutRecord(req *kinesis.PutRecordInput) (*kinesis.PutRecordOutput, error) {
	if c.putRecords == nil {
		c.seqNumber = rand.Int()
		c.putRecords = make([]kinesis.PutRecordInput, 0)
	}
	c.putRecords = append(c.putRecords, *req)
	c.seqNumber++
	return &kinesis.PutRecordOutput{
		SequenceNumber: S(strconv.Itoa(c.seqNumber)),
		ShardID:        req.PartitionKey,
	}, nil
}

func (c *MockClient) RespondDecisionTaskCompleted(req *swf.RespondDecisionTaskCompletedInput) (err error) {
	return nil
}
