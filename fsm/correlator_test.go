package fsm

import (
	"fmt"
	"log"
	"math/rand"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/sclasen/swfsm/enums/swf"
	. "github.com/sclasen/swfsm/sugar"
)

func TestTrackPendingActivities(t *testing.T) {
	fsm := testFSM()

	fsm.AddInitialState(&FSMState{
		Name: "start",
		Decider: func(f *FSMContext, lastEvent *swf.HistoryEvent, data interface{}) Outcome {
			testData := data.(*TestData)
			testData.States = append(testData.States, "start")
			serialized := f.Serialize(testData)
			decision := &swf.Decision{
				DecisionType: S(enums.DecisionTypeScheduleActivityTask),
				ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
					ActivityID:   S(testActivityInfo.ActivityID),
					ActivityType: testActivityInfo.ActivityType,
					TaskList:     &swf.TaskList{Name: S("taskList")},
					Input:        S(serialized),
				},
			}
			return f.Goto("working", testData, []*swf.Decision{decision})
		},
	})

	// Deciders should be able to retrieve info about the pending activity
	fsm.AddState(&FSMState{
		Name: "working",
		Decider: typedFuncs.Decider(func(f *FSMContext, lastEvent *swf.HistoryEvent, testData *TestData) Outcome {
			testData.States = append(testData.States, "working")
			serialized := f.Serialize(testData)
			var decisions = f.EmptyDecisions()
			if *lastEvent.EventType == enums.EventTypeActivityTaskCompleted {
				trackedActivityInfo := f.ActivityInfo(lastEvent)
				log.Printf("----->>>>> %+v %+v", trackedActivityInfo, testActivityInfo)
				if !reflect.DeepEqual(*trackedActivityInfo, testActivityInfo) {
					t.Fatalf("pending activity not being tracked\nExpected:\n%+v\nGot:\n%+v",
						testActivityInfo, trackedActivityInfo,
					)
				}
				timeoutSeconds := "5" //swf uses stringy numbers in many places
				decision := &swf.Decision{
					DecisionType: S(enums.DecisionTypeStartTimer),
					StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
						StartToFireTimeout: S(timeoutSeconds),
						TimerID:            S("timeToComplete"),
					},
				}
				return f.Goto("done", testData, []*swf.Decision{decision})
			} else if *lastEvent.EventType == enums.EventTypeActivityTaskFailed {
				trackedActivityInfo := f.ActivityInfo(lastEvent)
				log.Printf("----->>>>> %+v %+v %+v", trackedActivityInfo, testActivityInfo, f.ActivitiesInfo())
				if !reflect.DeepEqual(*trackedActivityInfo, testActivityInfo) {
					t.Fatalf("pending activity not being tracked\nExpected:\n%+v\nGot:\n%+v",
						testActivityInfo, trackedActivityInfo,
					)
				}
				decision := &swf.Decision{
					DecisionType: S(enums.DecisionTypeScheduleActivityTask),
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
		Decider: typedFuncs.Decider(func(f *FSMContext, lastEvent *swf.HistoryEvent, testData *TestData) Outcome {
			decisions := f.EmptyDecisions()
			if *lastEvent.EventType == enums.EventTypeTimerFired {
				testData.States = append(testData.States, "done")
				serialized := f.Serialize(testData)
				trackedActivityInfo := f.ActivityInfo(lastEvent)
				if trackedActivityInfo != nil {
					t.Fatalf("pending activity not being cleared\nGot:\n%+v", trackedActivityInfo)
				}
				decision := &swf.Decision{
					DecisionType: S(enums.DecisionTypeCompleteWorkflowExecution),
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
	events := []*swf.HistoryEvent{
		&swf.HistoryEvent{EventType: S("DecisionTaskStarted"), EventID: I(3)},
		&swf.HistoryEvent{EventType: S("DecisionTaskScheduled"), EventID: I(2)},
		&swf.HistoryEvent{
			EventID:   I(1),
			EventType: S("WorkflowExecutionStarted"),
			WorkflowExecutionStartedEventAttributes: &swf.WorkflowExecutionStartedEventAttributes{
				Input: S(fsm.Serialize(new(TestData))),
			},
		},
	}
	first := testDecisionTask(0, events)

	_, decisions, _, _ := fsm.Tick(first)
	recordMarker := FindDecision(decisions, stateMarkerPredicate)
	if recordMarker == nil {
		t.Fatal("No Record State Marker")
	}
	if !Find(decisions, scheduleActivityPredicate) {
		t.Fatal("No ScheduleActivityTask")
	}

	// Fail the task
	secondEvents := []*swf.HistoryEvent{
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
	if state, _ := fsm.findSerializedState(secondEvents); state.StateName != "working" {
		t.Fatal("current state is not 'working'", secondEvents)
	}
	second := testDecisionTask(3, secondEvents)

	_, secondDecisions, _, _ := fsm.Tick(second)
	recordMarker = FindDecision(secondDecisions, stateMarkerPredicate)
	if recordMarker == nil {
		t.Fatal("No Record State Marker")
	}
	if !Find(secondDecisions, scheduleActivityPredicate) {
		t.Fatal("No ScheduleActivityTask (retry)")
	}

	// Complete the task
	thirdEvents := []*swf.HistoryEvent{
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
	if state, _ := fsm.findSerializedState(thirdEvents); state.StateName != "working" {
		t.Fatal("current state is not 'working'", thirdEvents)
	}
	third := testDecisionTask(7, thirdEvents)
	_, thirdDecisions, _, _ := fsm.Tick(third)
	recordMarker = FindDecision(thirdDecisions, stateMarkerPredicate)
	if recordMarker == nil {
		t.Fatal("No Record State Marker")
	}
	if !Find(thirdDecisions, startTimerPredicate) {
		t.Fatal("No StartTimer")
	}

	// Finish the workflow, check if pending activities were cleared
	fourthEvents := []*swf.HistoryEvent{
		{
			EventType: S("TimerFired"),
			EventID:   I(14),
			TimerFiredEventAttributes: &swf.TimerFiredEventAttributes{
				StartedEventID: I(12),
				TimerID:        S("FOO"),
			},
		},
		{
			EventType: S("MarkerRecorded"),
			EventID:   I(13),
			MarkerRecordedEventAttributes: &swf.MarkerRecordedEventAttributes{
				MarkerName: S(StateMarker),
				Details:    recordMarker.RecordMarkerDecisionAttributes.Details,
			},
		},
		{
			EventType: S(enums.EventTypeTimerStarted),
			EventID:   I(12),
			TimerStartedEventAttributes: &swf.TimerStartedEventAttributes{
				TimerID: S("foo"),
			},
		},
	}
	fourthEvents = append(fourthEvents, thirdEvents...)
	if state, _ := fsm.findSerializedState(fourthEvents); state.StateName != "done" {
		t.Fatal("current state is not 'done'", fourthEvents)
	}
	fourth := testDecisionTask(11, fourthEvents)
	_, fourthDecisions, _, _ := fsm.Tick(fourth)
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
	taskScheduled := &swf.HistoryEvent{
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
	ctx.Decide(taskScheduled, &TestData{}, func(ctx *FSMContext, h *swf.HistoryEvent, data interface{}) Outcome {
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
	taskCompleted := &swf.HistoryEvent{
		EventType: S("ActivityTaskCompleted"),
		EventID:   I(rand.Int()),
		ActivityTaskCompletedEventAttributes: &swf.ActivityTaskCompletedEventAttributes{
			ScheduledEventID: I(scheduledEventID),
		},
	}
	taskFailed := &swf.HistoryEvent{
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
	ctx.Decide(taskCompleted, &TestData{}, func(ctx *FSMContext, h *swf.HistoryEvent, data interface{}) Outcome {
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

func TestCountActivityAttemtps(t *testing.T) {
	c := new(EventCorrelator)
	c.Serializer = JSONStateSerializer{}

	start := func(eventId int) *swf.HistoryEvent {
		return EventFromPayload(eventId, &swf.ActivityTaskScheduledEventAttributes{
			ActivityID: S("the-id"),
		})
	}

	fail := EventFromPayload(2, &swf.ActivityTaskFailedEventAttributes{
		ScheduledEventID: I(1),
	})

	timeout := EventFromPayload(4, &swf.ActivityTaskTimedOutEventAttributes{
		ScheduledEventID: I(3),
	})

	c.Track(start(1))
	c.Track(fail)
	c.Track(start(3))
	info := c.ActivityInfo(timeout)
	c.Track(timeout)
	log.Println("=====")
	log.Printf("%+v", c.ActivityAttempts)
	if c.AttemptsForActivity(info) != 2 {
		t.Fatal(PrettyHistoryEvent(start(1)), PrettyHistoryEvent(fail), PrettyHistoryEvent(timeout), c)
	}
	cancel := EventFromPayload(6, &swf.ActivityTaskCanceledEventAttributes{
		ScheduledEventID: I(5),
	})

	c.Track(start(5))
	c.Track(cancel)

	if c.AttemptsForActivity(info) != 0 {
		t.Fatal(c)
	}

	c.Track(start(7))

	if c.AttemptsForActivity(info) != 0 {
		t.Fatal(c)
	}

}

func TestSignalTracking(t *testing.T) {
	//track signal'->'workflowID => attempts
	event := func(eventId int, payload interface{}) *swf.HistoryEvent {
		return EventFromPayload(eventId, payload)
	}

	start := event(1, &swf.SignalExternalWorkflowExecutionInitiatedEventAttributes{
		SignalName: S("the-signal"),
		WorkflowID: S("the-workflow"),
		RunID:      S("the-runid"),
	})

	fail := EventFromPayload(2, &swf.SignalExternalWorkflowExecutionFailedEventAttributes{
		InitiatedEventID: I(1),
	})

	ok := EventFromPayload(4, &swf.ExternalWorkflowExecutionSignaledEventAttributes{
		InitiatedEventID: I(3),
	})

	c := new(EventCorrelator)
	c.Serializer = JSONStateSerializer{}

	c.Track(start)
	//track happens in FSM after Decider
	info := c.SignalInfo(fail)
	c.Track(fail)
	start.EventID = I(3)
	c.Track(start)
	if c.AttemptsForSignal(info) != 1 {
		t.Fatal(c.SignalAttempts)
	}
	info = c.SignalInfo(ok)
	c.Track(ok)

	if c.AttemptsForSignal(info) != 0 {
		t.Fatal("expected zero attempts", c)
	}
}

func TestTimerTracking(t *testing.T) {
	//track signal'->'workflowID => attempts
	event := func(eventId int, payload interface{}) *swf.HistoryEvent {
		return EventFromPayload(eventId, payload)
	}

	start := event(1, &swf.SignalExternalWorkflowExecutionInitiatedEventAttributes{
		SignalName: S("the-signal"),
		WorkflowID: S("the-workflow"),
		RunID:      S("the-runid"),
	})

	timerStart := EventFromPayload(2, &swf.TimerStartedEventAttributes{
		TimerID: S("the-timer"),
		Control: S("the-control"),
	})

	timerFired := EventFromPayload(3, &swf.TimerFiredEventAttributes{
		StartedEventID: I(2),
	})

	timerStart2 := EventFromPayload(4, &swf.TimerStartedEventAttributes{
		TimerID: S("the-timer"),
		Control: S("the-control"),
	})

	timerCanceled := EventFromPayload(5, &swf.TimerCanceledEventAttributes{
		StartedEventID: I(4),
	})

	c := new(EventCorrelator)
	c.Serializer = JSONStateSerializer{}

	c.Track(start)
	c.Track(timerStart)
	//track happens in FSM after Decider
	info := c.TimerInfo(timerFired)
	if info == nil || info.Control != "the-control" || info.TimerID != "the-timer" {
		t.Fatal(info)
	}

	c.Track(timerFired)
	info = c.TimerInfo(timerFired)

	if info != nil {
		t.Fatal("non nil info %v", info)
	}

	c.Track(timerStart2)
	info = c.TimerInfo(timerCanceled)

	if info == nil || info.Control != "the-control" || info.TimerID != "the-timer" {
		t.Fatal(info)
	}

	c.Track(timerCanceled)
	info = c.TimerInfo(timerCanceled)

	if info != nil {
		t.Fatal("non nil info2 %v", info)
	}
}

func TestCancelTracking(t *testing.T) {
	//track signal'->'workflowID => attempts
	event := func(eventId int, payload interface{}) *swf.HistoryEvent {
		return EventFromPayload(eventId, payload)
	}

	start := event(1, &swf.RequestCancelExternalWorkflowExecutionInitiatedEventAttributes{
		WorkflowID: S("the-workflow"),
	})

	fail := EventFromPayload(2, &swf.RequestCancelExternalWorkflowExecutionFailedEventAttributes{
		InitiatedEventID: I(1),
	})

	ok := EventFromPayload(4, &swf.WorkflowExecutionCancelRequestedEventAttributes{
		ExternalInitiatedEventID: I(3),
	})

	c := new(EventCorrelator)
	c.Serializer = JSONStateSerializer{}

	c.Track(start)

	//track happens in FSM after Decider
	info := c.CancellationInfo(fail)
	t.Log(info, c.Cancellations, c.CancelationAttempts)

	c.Track(fail)
	start.EventID = I(3)
	c.Track(start)
	if c.AttemptsForCancellation(info) != 1 {
		t.Fatal("attempts not", info, c.AttemptsForCancellation(info), c.Cancellations, c.CancelationAttempts)
	}
	info = c.CancellationInfo(ok)
	c.Track(ok)

	if c.AttemptsForCancellation(info) != 0 {
		t.Fatal("expected zero attempts", c)
	}
}
