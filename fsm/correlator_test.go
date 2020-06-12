package fsm

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/service/swf"
	. "github.com/sclasen/swfsm/log"
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

	// Deciders should be able to retrieve info about the pending activity
	fsm.AddState(&FSMState{
		Name: "working",
		Decider: typedFuncs.Decider(func(f *FSMContext, lastEvent *swf.HistoryEvent, testData *TestData) Outcome {
			testData.States = append(testData.States, "working")
			serialized := f.Serialize(testData)
			var decisions = f.EmptyDecisions()
			if *lastEvent.EventType == swf.EventTypeActivityTaskCompleted {
				trackedActivityInfo := f.ActivityInfo(lastEvent)
				Log.Printf("----->>>>> %+v %+v", trackedActivityInfo, testActivityInfo)
				if !reflect.DeepEqual(*trackedActivityInfo, testActivityInfo) {
					t.Fatalf("pending activity not being tracked\nExpected:\n%+v\nGot:\n%+v",
						testActivityInfo, trackedActivityInfo,
					)
				}
				timeoutSeconds := "5" //swf uses stringy numbers in many places
				decision := &swf.Decision{
					DecisionType: S(swf.DecisionTypeStartTimer),
					StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
						StartToFireTimeout: S(timeoutSeconds),
						TimerId:            S("timeToComplete"),
					},
				}
				return f.Goto("done", testData, []*swf.Decision{decision})
			} else if *lastEvent.EventType == swf.EventTypeActivityTaskFailed {
				trackedActivityInfo := f.ActivityInfo(lastEvent)
				Log.Printf("----->>>>> %+v %+v %+v", trackedActivityInfo, testActivityInfo, f.ActivitiesInfo())
				if !reflect.DeepEqual(*trackedActivityInfo, testActivityInfo) {
					t.Fatalf("pending activity not being tracked\nExpected:\n%+v\nGot:\n%+v",
						testActivityInfo, trackedActivityInfo,
					)
				}
				decision := &swf.Decision{
					DecisionType: S(swf.DecisionTypeScheduleActivityTask),
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

	// Pending activities are cleared after finished
	fsm.AddState(&FSMState{
		Name: "done",
		Decider: typedFuncs.Decider(func(f *FSMContext, lastEvent *swf.HistoryEvent, testData *TestData) Outcome {
			decisions := f.EmptyDecisions()
			if *lastEvent.EventType == swf.EventTypeTimerFired {
				testData.States = append(testData.States, "done")
				serialized := f.Serialize(testData)
				trackedActivityInfo := f.ActivityInfo(lastEvent)
				if trackedActivityInfo != nil {
					t.Fatalf("pending activity not being cleared\nGot:\n%+v", trackedActivityInfo)
				}
				decision := &swf.Decision{
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
	events := []*swf.HistoryEvent{
		&swf.HistoryEvent{EventType: S("DecisionTaskStarted"), EventId: I(3)},
		&swf.HistoryEvent{EventType: S("DecisionTaskScheduled"), EventId: I(2)},
		&swf.HistoryEvent{
			EventId:   I(1),
			EventType: S("WorkflowExecutionStarted"),
			WorkflowExecutionStartedEventAttributes: &swf.WorkflowExecutionStartedEventAttributes{
				Input: StartFSMWorkflowInput(fsm, new(TestData)),
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
			EventId:   I(7),
			ActivityTaskFailedEventAttributes: &swf.ActivityTaskFailedEventAttributes{
				ScheduledEventId: I(6),
			},
		},
		{
			EventType: S("ActivityTaskScheduled"),
			EventId:   I(6),
			ActivityTaskScheduledEventAttributes: &swf.ActivityTaskScheduledEventAttributes{
				ActivityId:   S(testActivityInfo.ActivityId),
				ActivityType: testActivityInfo.ActivityType,
			},
		},
		{
			EventType: S("MarkerRecorded"),
			EventId:   I(5),
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
			EventId:   I(11),
			ActivityTaskCompletedEventAttributes: &swf.ActivityTaskCompletedEventAttributes{
				ScheduledEventId: I(10),
			},
		},
		{
			EventType: S("ActivityTaskScheduled"),
			EventId:   I(10),
			ActivityTaskScheduledEventAttributes: &swf.ActivityTaskScheduledEventAttributes{
				ActivityId:   S(testActivityInfo.ActivityId),
				ActivityType: testActivityInfo.ActivityType,
			},
		},
		{
			EventType: S("MarkerRecorded"),
			EventId:   I(9),
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
			EventId:   I(14),
			TimerFiredEventAttributes: &swf.TimerFiredEventAttributes{
				StartedEventId: I(12),
				TimerId:        S("FOO"),
			},
		},
		{
			EventType: S("MarkerRecorded"),
			EventId:   I(13),
			MarkerRecordedEventAttributes: &swf.MarkerRecordedEventAttributes{
				MarkerName: S(StateMarker),
				Details:    recordMarker.RecordMarkerDecisionAttributes.Details,
			},
		},
		{
			EventType: S(swf.EventTypeTimerStarted),
			EventId:   I(12),
			TimerStartedEventAttributes: &swf.TimerStartedEventAttributes{
				TimerId:            S("foo"),
				StartToFireTimeout: S("20"),
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
	scheduledEventId := rand.Int()
	activityId := fmt.Sprintf("test-activity-%d", scheduledEventId)
	taskScheduled := &swf.HistoryEvent{
		EventType: S("ActivityTaskScheduled"),
		EventId:   I(scheduledEventId),
		ActivityTaskScheduledEventAttributes: &swf.ActivityTaskScheduledEventAttributes{
			ActivityId: S(activityId),
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
		EventId:   I(rand.Int()),
		ActivityTaskCompletedEventAttributes: &swf.ActivityTaskCompletedEventAttributes{
			ScheduledEventId: I(scheduledEventId),
		},
	}
	taskFailed := &swf.HistoryEvent{
		EventType: S("ActivityTaskFailed"),
		EventId:   I(rand.Int()),
		ActivityTaskFailedEventAttributes: &swf.ActivityTaskFailedEventAttributes{
			ScheduledEventId: I(scheduledEventId),
		},
	}
	infoOnCompleted := ctx.ActivityInfo(taskCompleted)
	infoOnFailed := ctx.ActivityInfo(taskFailed)
	if infoOnCompleted.ActivityId != activityId ||
		*infoOnCompleted.Name != "test-activity" ||
		*infoOnCompleted.Version != "1" {
		t.Fatal("Pending activity can not be retrieved when completed")
	}
	if infoOnFailed.ActivityId != activityId ||
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
		if infoOnCompleted.ActivityId != activityId ||
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
			ActivityId: S("the-id"),
		})
	}

	fail := EventFromPayload(2, &swf.ActivityTaskFailedEventAttributes{
		ScheduledEventId: I(1),
	})

	timeout := EventFromPayload(4, &swf.ActivityTaskTimedOutEventAttributes{
		ScheduledEventId: I(3),
	})

	c.Track(start(1))
	c.Track(fail)
	c.Track(start(3))
	info := c.ActivityInfo(timeout)
	if c.Attempts(timeout) != 1 {
		t.Fatal(PrettyHistoryEvent(start(1)), PrettyHistoryEvent(fail), PrettyHistoryEvent(timeout), info, c.ActivityAttempts, c.Activities)
	}
	c.Track(timeout)
	Log.Println("=====")
	Log.Printf("%+v", c.ActivityAttempts)
	if c.AttemptsForActivity(info) != 2 {
		t.Fatal(PrettyHistoryEvent(start(1)), PrettyHistoryEvent(fail), PrettyHistoryEvent(timeout), c)
	}

	cancel := EventFromPayload(6, &swf.ActivityTaskCanceledEventAttributes{
		ScheduledEventId: I(5),
	})

	c.Track(start(5))
	c.Track(cancel)

	if c.AttemptsForActivity(info) != 0 {
		t.Fatal(c)
	}
	if c.Attempts(cancel) != 0 {
		t.Fatal(c)
	}

	c.Track(start(7))

	if c.AttemptsForActivity(info) != 0 {
		t.Fatal(c)
	}
	if c.Attempts(start(7)) != 0 {
		t.Fatal(c)
	}

}

func TestSignalTracking(t *testing.T) {
	//track signal'->'workflowId => attempts
	event := func(eventId int, payload interface{}) *swf.HistoryEvent {
		return EventFromPayload(eventId, payload)
	}

	start := event(1, &swf.SignalExternalWorkflowExecutionInitiatedEventAttributes{
		SignalName: S("the-signal"),
		WorkflowId: S("the-workflow"),
		RunId:      S("the-runid"),
	})

	fail := EventFromPayload(2, &swf.SignalExternalWorkflowExecutionFailedEventAttributes{
		InitiatedEventId: I(1),
	})

	ok := EventFromPayload(4, &swf.ExternalWorkflowExecutionSignaledEventAttributes{
		InitiatedEventId: I(3),
	})

	c := new(EventCorrelator)
	c.Serializer = JSONStateSerializer{}

	c.Track(start)
	//track happens in FSM after Decider
	info := c.SignalInfo(fail)
	c.Track(fail)
	start.EventId = I(3)
	c.Track(start)
	if c.AttemptsForSignal(info) != 1 {
		t.Fatal(c.SignalAttempts)
	}
	if c.Attempts(start) != 1 {
		t.Fatal(c.SignalAttempts)
	}
	info = c.SignalInfo(ok)
	c.Track(ok)

	if c.AttemptsForSignal(info) != 0 {
		t.Fatal("expected zero attempts", c)
	}
	if c.Attempts(ok) != 0 {
		t.Fatal("expected zero attempts", c)
	}
}

func TestTimerTracking(t *testing.T) {
	//track signal'->'workflowId => attempts
	event := func(eventId int, payload interface{}) *swf.HistoryEvent {
		return EventFromPayload(eventId, payload)
	}

	start := event(1, &swf.SignalExternalWorkflowExecutionInitiatedEventAttributes{
		SignalName: S("the-signal"),
		WorkflowId: S("the-workflow"),
		RunId:      S("the-runid"),
	})

	timerStart := EventFromPayload(2, &swf.TimerStartedEventAttributes{
		TimerId:            S("the-timer"),
		Control:            S("the-control"),
		StartToFireTimeout: S("20"),
	})

	timerFired := EventFromPayload(3, &swf.TimerFiredEventAttributes{
		StartedEventId: I(2),
	})

	timerStart2 := EventFromPayload(4, &swf.TimerStartedEventAttributes{
		TimerId:            S("the-timer"),
		Control:            S("the-control"),
		StartToFireTimeout: S("20"),
	})

	timerCanceled := EventFromPayload(5, &swf.TimerCanceledEventAttributes{
		StartedEventId: I(4),
	})

	c := new(EventCorrelator)
	c.Serializer = JSONStateSerializer{}

	c.Track(start)
	c.Track(timerStart)
	//track happens in FSM after Decider
	info := c.TimerInfo(timerFired)
	if info == nil || *info.Control != "the-control" || info.TimerId != "the-timer" {
		t.Fatal(info)
	}

	c.Track(timerFired)
	info = c.TimerInfo(timerFired)

	if info != nil {
		t.Fatalf("non nil info %v", info)
	}

	c.Track(timerStart2)
	info = c.TimerInfo(timerCanceled)

	if info == nil || *info.Control != "the-control" || info.TimerId != "the-timer" {
		t.Fatal(info)
	}

	c.Track(timerCanceled)
	info = c.TimerInfo(timerCanceled)

	if info != nil {
		t.Fatalf("non nil info2 %v", info)
	}
}

func TestCancelTracking(t *testing.T) {
	//track signal'->'workflowId => attempts
	event := func(eventId int, payload interface{}) *swf.HistoryEvent {
		return EventFromPayload(eventId, payload)
	}

	start := event(1, &swf.RequestCancelExternalWorkflowExecutionInitiatedEventAttributes{
		WorkflowId: S("the-workflow"),
	})

	fail := EventFromPayload(2, &swf.RequestCancelExternalWorkflowExecutionFailedEventAttributes{
		InitiatedEventId: I(1),
	})

	ok := EventFromPayload(4, &swf.WorkflowExecutionCancelRequestedEventAttributes{
		ExternalInitiatedEventId: I(3),
	})

	c := new(EventCorrelator)
	c.Serializer = JSONStateSerializer{}

	c.Track(start)

	//track happens in FSM after Decider
	info := c.CancellationInfo(fail)
	t.Log(info, c.Cancellations, c.CancelationAttempts)

	c.Track(fail)
	start.EventId = I(3)
	c.Track(start)
	if c.AttemptsForCancellation(info) != 1 {
		t.Fatal("attempts not", info, c.AttemptsForCancellation(info), c.Cancellations, c.CancelationAttempts)
	}
	if c.Attempts(start) != 1 {
		t.Fatal("attempts not", start, c.Attempts(start), c.Cancellations, c.CancelationAttempts)
	}

	info = c.CancellationInfo(ok)
	c.Track(ok)

	if c.AttemptsForCancellation(info) != 0 {
		t.Fatal("expected zero attempts", c)
	}
	if c.Attempts(ok) != 0 {
		t.Fatal("expected zero attempts", c)
	}
}

func TestChildTracking(t *testing.T) {
	//track signal'->'workflowId => attempts
	event := func(eventId int, payload interface{}) *swf.HistoryEvent {
		return EventFromPayload(eventId, payload)
	}

	start := event(1, &swf.StartChildWorkflowExecutionInitiatedEventAttributes{
		WorkflowId: S("the-workflow"),
		WorkflowType: &swf.WorkflowType{
			Name:    S("the-name"),
			Version: S("the-version"),
		},
	})

	fail := EventFromPayload(2, &swf.StartChildWorkflowExecutionFailedEventAttributes{
		InitiatedEventId: I(1),
	})

	ok := EventFromPayload(4, &swf.ChildWorkflowExecutionStartedEventAttributes{
		InitiatedEventId: I(3),
	})

	c := new(EventCorrelator)
	c.Serializer = JSONStateSerializer{}

	c.Track(start)

	//track happens in FSM after Decider
	info := c.ChildInfo(fail)
	t.Log(info, c.Children, c.ChildrenAttempts)

	c.Track(fail)
	start.EventId = I(3)
	c.Track(start)
	if c.AttemptsForChild(info) != 1 {
		t.Fatal("attempts not", info, c.AttemptsForChild(info), c.Children, c.ChildrenAttempts)
	}
	if c.Attempts(start) != 1 {
		t.Fatal("attempts not", info, c.Attempts(start), c.Children, c.ChildrenAttempts)
	}
	info = c.ChildInfo(ok)
	c.Track(ok)

	if c.AttemptsForChild(info) != 0 {
		t.Fatal("expected zero attempts", c)
	}
	if c.Attempts(ok) != 0 {
		t.Fatal("expected zero attempts", c)
	}
}

func TestActivityInfoFromSignalEvent(t *testing.T) {
	event := func(eventId int, payload interface{}) *swf.HistoryEvent {
		return EventFromPayload(eventId, payload)
	}

	start := event(1, &swf.StartChildWorkflowExecutionInitiatedEventAttributes{
		WorkflowId: S("the-workflow"),
		WorkflowType: &swf.WorkflowType{
			Name:    S("the-name"),
			Version: S("the-version"),
		},
	})

	sched := EventFromPayload(2, &swf.ActivityTaskScheduledEventAttributes{
		ActivityId: S("the-activity"),
		Input:      S("the-input"),
		ActivityType: &swf.ActivityType{
			Name:    S("activity-name"),
			Version: S("activity-verions"),
		},
	})

	c := new(EventCorrelator)
	c.Serializer = JSONStateSerializer{}

	state := &SerializedActivityState{
		ActivityId: "the-activity",
		Input:      S("the-update"),
	}

	ser, _ := c.Serializer.Serialize(state)

	update := EventFromPayload(4, &swf.WorkflowExecutionSignaledEventAttributes{
		SignalName: S(ActivityUpdatedSignal),
		Input:      S(ser),
	})

	c.Track(start)
	c.Track(sched)

	info := c.ActivityInfo(update)
	if info == nil {
		s, _ := c.Serializer.Serialize(c.Activities)
		t.Fatalf("didnt find the activity! %s", s)
	}

}
