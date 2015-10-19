package fsm

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/sclasen/swfsm/testing/mocks"
)

func TestSegmentHistory(t *testing.T) {
	fsm := dummyFsm()

	var readyData interface{}
	readyData = TestData{States: []string{"ready data"}}

	var startData interface{}
	startData = TestData{States: []string{"start data"}}

	activityId := "activity-id"
	activityData := "activity data"

	errorState := &SerializedErrorState{
		EarliestUnprocessedEventId: -1,
		LatestUnprocessedEventId:   -2,
		ErrorEvent: &swf.HistoryEvent{
			EventId: aws.Int64(-3),
		},
	}

	history := &swf.GetWorkflowExecutionHistoryOutput{Events: []*swf.HistoryEvent{
		&swf.HistoryEvent{
			EventId:   aws.Int64(8),
			EventType: aws.String(swf.EventTypeActivityTaskScheduled),
			ActivityTaskScheduledEventAttributes: &swf.ActivityTaskScheduledEventAttributes{
				ActivityId: aws.String(activityId),
				Input:      aws.String(fsm.Serialize(activityData)),
				DecisionTaskCompletedEventId: aws.Int64(4),
			},
		},
		&swf.HistoryEvent{
			EventId:   aws.Int64(7),
			EventType: aws.String(swf.EventTypeMarkerRecorded),
			MarkerRecordedEventAttributes: &swf.MarkerRecordedEventAttributes{
				MarkerName: aws.String(ErrorMarker),
				Details:    aws.String(fsm.Serialize(errorState)),
			},
		},
		&swf.HistoryEvent{
			EventId:   aws.Int64(6),
			EventType: aws.String(swf.EventTypeMarkerRecorded),
			MarkerRecordedEventAttributes: &swf.MarkerRecordedEventAttributes{
				MarkerName: aws.String(CorrelatorMarker),
				Details:    aws.String(fsm.Serialize(EventCorrelator{})),
			},
		},
		&swf.HistoryEvent{
			EventId:   aws.Int64(5),
			EventType: aws.String(swf.EventTypeMarkerRecorded),
			MarkerRecordedEventAttributes: &swf.MarkerRecordedEventAttributes{
				MarkerName: aws.String(StateMarker),
				Details: aws.String(fsm.Serialize(SerializedState{
					StateVersion: 1,
					StateName:    "ready",
					StateData:    fsm.Serialize(readyData),
				})),
			},
		},
		&swf.HistoryEvent{
			EventId:   aws.Int64(4),
			EventType: aws.String(swf.EventTypeDecisionTaskCompleted),
		},
		&swf.HistoryEvent{
			EventId:   aws.Int64(3),
			EventType: aws.String(swf.EventTypeDecisionTaskStarted),
		},
		&swf.HistoryEvent{
			EventId:   aws.Int64(2),
			EventType: aws.String(swf.EventTypeDecisionTaskScheduled),
		},
		&swf.HistoryEvent{
			EventId:   aws.Int64(1),
			EventType: aws.String(swf.EventTypeWorkflowExecutionStarted),
			WorkflowExecutionStartedEventAttributes: &swf.WorkflowExecutionStartedEventAttributes{
				Input: StartFSMWorkflowInput(fsm, &startData),
			},
		},
	}}

	actual := []HistorySegment{}
	seg := NewFSMClient(fsm, &mocks.SWFAPI{}).NewHistorySegmentor()
	seg.OnSegment(func(segment HistorySegment) {
		actual = append(actual, segment)
	})
	seg.OnError(func(err error) {
		t.Error(err)
	})

	shouldContinue := seg.FromPage(history, true)
	if !shouldContinue {
		t.Error(shouldContinue)
	}

	expected := []HistorySegment{
		HistorySegment{
			State: &HistorySegmentState{
				ID:      aws.Int64(999999),
				Version: uInt64(999999),
				Name:    aws.String("<unrecorded>"),
				Data:    nil,
			},
			Correlator: nil,
			Error:      nil,
			Events: []*HistorySegmentEvent{
				&HistorySegmentEvent{
					ID:   aws.Int64(8),
					Type: aws.String(swf.EventTypeActivityTaskScheduled),
					Attributes: &map[string]interface{}{
						"ActivityId":                   activityId,
						"ActivityType":                 nil,
						"Control":                      nil,
						"DecisionTaskCompletedEventId": 4,
						"HeartbeatTimeout":             nil,
						"Input":                        fsm.Serialize(activityData), // TODO: un-double escape?
						"ScheduleToCloseTimeout":       nil,
						"ScheduleToStartTimeout":       nil,
						"StartToCloseTimeout":          nil,
						"TaskList":                     nil,
						"TaskPriority":                 nil,
					},
				},
			},
		},
		HistorySegment{
			State: &HistorySegmentState{
				ID:      aws.Int64(5),
				Version: uInt64(1),
				Name:    aws.String("ready"),
				Data:    &readyData,
			},
			Correlator: &EventCorrelator{},
			Error:      errorState,
			Events: []*HistorySegmentEvent{
				&HistorySegmentEvent{
					ID:         aws.Int64(4),
					Type:       aws.String(swf.EventTypeDecisionTaskCompleted),
					References: []*int64{aws.Int64(8)},
				},
				&HistorySegmentEvent{
					ID:   aws.Int64(3),
					Type: aws.String(swf.EventTypeDecisionTaskStarted),
				},
				&HistorySegmentEvent{
					ID:   aws.Int64(2),
					Type: aws.String(swf.EventTypeDecisionTaskScheduled),
				},
			},
		},
		HistorySegment{
			State: &HistorySegmentState{
				ID:      aws.Int64(1),
				Version: uInt64(0),
				Name:    aws.String("initial"),
				Data:    &startData,
			},
			Correlator: nil,
			Events:     []*HistorySegmentEvent{},
		},
	}

	for i, e := range expected {
		a := actual[i]
		if fsm.Serialize(e) != fsm.Serialize(a) {
			t.Logf("expected[%d]:\n%#v\n\n actual[%d]:\n%#v", i, fsm.Serialize(e), i, fsm.Serialize(a))
			t.FailNow()
		}
	}
}

func uInt64(v uint64) *uint64 {
	return &v
}
