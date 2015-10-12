package fsm

import (
	"os"
	"testing"

	"strings"

	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/pborman/uuid"
	. "github.com/sclasen/swfsm/log"
	"github.com/sclasen/swfsm/migrator"
	"github.com/sclasen/swfsm/testing/mocks"
)

func TestClient(t *testing.T) {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		Log.Printf("WARNING: NO AWS CREDS SPECIFIED, SKIPPING MIGRATIONS TEST")
		return
	}

	config := &aws.Config{
		Credentials: credentials.NewEnvCredentials(),
		Region:      aws.String("us-east-1"),
	}
	client := swf.New(config)

	req := swf.RegisterDomainInput{
		Name:                                   aws.String("client-test"),
		Description:                            aws.String("test domain"),
		WorkflowExecutionRetentionPeriodInDays: aws.String("30"),
	}

	d := migrator.DomainMigrator{
		RegisteredDomains: []swf.RegisterDomainInput{req},
		Client:            client,
	}

	d.Migrate()

	wreq := swf.RegisterWorkflowTypeInput{
		Name:        aws.String("client-test"),
		Description: aws.String("test workflow migration"),
		Version:     aws.String("1"),
		Domain:      aws.String("client-test"),
	}

	w := migrator.WorkflowTypeMigrator{
		RegisteredWorkflowTypes: []swf.RegisterWorkflowTypeInput{wreq},
		Client:                  client,
	}

	w.Migrate()

	fsm := &FSM{
		Domain:           "client-test",
		Name:             "client-test",
		DataType:         TestData{},
		Serializer:       JSONStateSerializer{},
		systemSerializer: JSONStateSerializer{},
		allowPanics:      false,
	}

	fsm.AddInitialState(&FSMState{Name: "initial",
		Decider: func(ctx *FSMContext, h *swf.HistoryEvent, data interface{}) Outcome {
			if *h.EventType == swf.EventTypeWorkflowExecutionSignaled {
				d := data.(*TestData)
				d.States = append(d.States, *h.WorkflowExecutionSignaledEventAttributes.SignalName)
			}
			return ctx.Stay(data, ctx.EmptyDecisions())
		},
	})

	fsmClient := NewFSMClient(fsm, client)

	workflow := uuid.New()
	testData := uuid.New()
	startTemplate := swf.StartWorkflowExecutionInput{
		WorkflowType:                 &swf.WorkflowType{Name: aws.String("client-test"), Version: aws.String("1")},
		ExecutionStartToCloseTimeout: aws.String("120"),
		TaskStartToCloseTimeout:      aws.String("120"),
		ChildPolicy:                  aws.String("ABANDON"),
		TaskList:                     &swf.TaskList{Name: aws.String("task-list")},
	}
	_, err := fsmClient.Start(startTemplate, workflow, &TestData{States: []string{testData}})

	if err != nil {
		t.Fatal(err)
	}

	state, data, err := fsmClient.GetState(workflow)
	if err != nil {
		t.Fatal(err)
	}

	if data.(*TestData).States[0] != testData {
		t.Fatal(data)
	}

	if state != "initial" {
		t.Fatal("not in initial")
	}

	found := false
	err = fsmClient.WalkOpenWorkflowInfos(&swf.ListOpenWorkflowExecutionsInput{}, func(infos *swf.WorkflowExecutionInfos) error {
		for _, info := range infos.ExecutionInfos {
			if *info.Execution.WorkflowId == workflow {
				found = true
				return StopWalking()
			}
		}
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	if !found {
		t.Fatalf("%s not found", workflow)
	}

	exec, err := fsmClient.FindLatestByWorkflowID(workflow)
	if err != nil {
		t.Fatal(err)
	}

	itr, err := fsmClient.GetHistoryEventIteratorFromWorkflowExecution(exec)
	if err != nil {
		t.Fatal(err)
	}

	segments, err := fsmClient.SegmentHistory(itr)
	if err != nil {
		t.Fatal(err)
	}

	if length := len(segments); length != 2 {
		t.Fatalf("segments length: %d \n%#v", length, segments)
	}

	if name := *segments[1].State.Name; name != "initial" {
		t.Fatalf("segments[1].State.Name: %s ", name)
	}

	if version := *segments[1].State.Version; version != 0 {
		t.Fatalf("segments[1].State.Version: %d ", version)
	}

	if id := *segments[1].State.ID; id != 1 {
		t.Fatalf("segments[1].State.ID: %d ", id)
	}
}

func TestStringDoesntSerialize(t *testing.T) {
	mockSwf := &mocks.SWFAPI{}
	mockSwf.MockOnAny_SignalWorkflowExecution().Return(func(req *swf.SignalWorkflowExecutionInput) *swf.SignalWorkflowExecutionOutput {
		if strings.Contains(*req.Input, "\"") {
			t.Fatal("simple string input has quotes")
		}
		if *req.Input != "simple" {
			t.Fatal("not simele")
		}
		return nil
	}, nil)

	NewFSMClient(dummyFsm(), mockSwf).Signal("wf", "signal", "simple")
	mockSwf.AssertExpectations(t)
}

func TestFindAll_Empty(t *testing.T) {
	input := &FindInput{}

	mockSwf := &mocks.SWFAPI{}

	expectedOpenInput := &swf.ListOpenWorkflowExecutionsInput{
		Domain: aws.String(dummyFsm().Domain),
	}
	mockSwf.MockOnTyped_ListOpenWorkflowExecutions(expectedOpenInput).Return(&swf.WorkflowExecutionInfos{}, nil)

	expectedClosedInput := &swf.ListClosedWorkflowExecutionsInput{
		Domain: aws.String(dummyFsm().Domain),
	}
	mockSwf.MockOnTyped_ListClosedWorkflowExecutions(expectedClosedInput).Return(&swf.WorkflowExecutionInfos{}, nil)

	output, err := NewFSMClient(dummyFsm(), mockSwf).FindAll(input)
	if err != nil {
		t.Fatal(err)
	}

	if len(output.ExecutionInfos) != 0 {
		t.Fatal(output.ExecutionInfos)
	}

	mockSwf.AssertExpectations(t)
}

func TestFindAll_MetadataFiltering(t *testing.T) {
	input := &FindInput{
		ExecutionFilter: &swf.WorkflowExecutionFilter{
			WorkflowId: aws.String("A"),
		},
		TagFilter: &swf.TagFilter{
			Tag: aws.String("T-2"),
		},
	}

	mockSwf := &mocks.SWFAPI{}

	expectedOpenInput := &swf.ListOpenWorkflowExecutionsInput{
		Domain:          aws.String(dummyFsm().Domain),
		ExecutionFilter: input.ExecutionFilter,
		/* does not include tag filter in server request, but filtered later locally */
	}
	mockSwf.MockOnTyped_ListOpenWorkflowExecutions(expectedOpenInput).Return(
		func(req *swf.ListOpenWorkflowExecutionsInput) *swf.WorkflowExecutionInfos {
			return &swf.WorkflowExecutionInfos{
				ExecutionInfos: []*swf.WorkflowExecutionInfo{
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("A"),
							RunId:      aws.String("A-1"),
						},
						TagList:        []*string{aws.String("T-1")},
						StartTimestamp: aws.Time(time.Now()),
					},
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("A"),
							RunId:      aws.String("A-2"),
						},
						TagList:        []*string{aws.String("T-2")},
						StartTimestamp: aws.Time(time.Now()),
					},
				},
			}
		}, nil)

	output, err := NewFSMClient(dummyFsm(), mockSwf).FindAll(input)
	if err != nil {
		t.Fatal(err)
	}

	if len(output.ExecutionInfos) != 1 {
		t.Fatal(output.ExecutionInfos)
	}

	if *output.ExecutionInfos[0].Execution.WorkflowId != "A" {
		t.Fatal(output.ExecutionInfos)
	}

	if *output.ExecutionInfos[0].Execution.RunId != "A-2" {
		t.Fatal(output.ExecutionInfos)
	}

	if *output.ExecutionInfos[0].TagList[0] != "T-2" {
		t.Fatal(output.ExecutionInfos)
	}

	mockSwf.AssertExpectations(t) // list closed not called
}

func TestFindAll_TimeFiltering(t *testing.T) {
	input := &FindInput{
		StartTimeFilter: &swf.ExecutionTimeFilter{
			OldestDate: aws.Time(time.Now().Add(-8 * time.Hour)),
			LatestDate: aws.Time(time.Now().Add(-6 * time.Hour)),
		},
		CloseTimeFilter: &swf.ExecutionTimeFilter{
			OldestDate: aws.Time(time.Now().Add(-4 * time.Hour)),
			LatestDate: aws.Time(time.Now().Add(-2 * time.Hour)),
		},
	}

	mockSwf := &mocks.SWFAPI{}

	expectedClosedInput := &swf.ListClosedWorkflowExecutionsInput{
		Domain:          aws.String(dummyFsm().Domain),
		StartTimeFilter: input.StartTimeFilter,
		/* does not include CloseTimeFilter in server request, but filtered later locally */
	}
	mockSwf.MockOnTyped_ListClosedWorkflowExecutions(expectedClosedInput).Return(
		func(req *swf.ListClosedWorkflowExecutionsInput) *swf.WorkflowExecutionInfos {
			return &swf.WorkflowExecutionInfos{
				ExecutionInfos: []*swf.WorkflowExecutionInfo{
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("A"),
						},
						StartTimestamp: aws.Time(time.Now().Add(-7 * time.Hour)),
						CloseTimestamp: aws.Time(time.Now().Add(-3 * time.Hour)),
					},
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("B-start timestamp too early"),
						},
						StartTimestamp: aws.Time(time.Now().Add(-9 * time.Hour)),
						CloseTimestamp: aws.Time(time.Now().Add(-3 * time.Hour)),
					},
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("C-start timestamp too late"),
						},
						StartTimestamp: aws.Time(time.Now().Add(-5 * time.Hour)),
						CloseTimestamp: aws.Time(time.Now().Add(-3 * time.Hour)),
					},
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("D-close timestamp too early"),
						},
						StartTimestamp: aws.Time(time.Now().Add(-7 * time.Hour)),
						CloseTimestamp: aws.Time(time.Now().Add(-5 * time.Hour)),
					},
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("E-close timestamp too late"),
						},
						StartTimestamp: aws.Time(time.Now().Add(-7 * time.Hour)),
						CloseTimestamp: aws.Time(time.Now().Add(-1 * time.Hour)),
					},
				},
			}
		}, nil)

	output, err := NewFSMClient(dummyFsm(), mockSwf).FindAll(input)
	if err != nil {
		t.Fatal(err)
	}

	if len(output.ExecutionInfos) != 1 {
		t.Fatal(output.ExecutionInfos)
	}

	if *output.ExecutionInfos[0].Execution.WorkflowId != "A" {
		t.Fatal(output.ExecutionInfos)
	}

	mockSwf.AssertExpectations(t) // list closed not called
}

func TestFindAll_CloseStatusFilterDefaultsStatusFilteredToClosed(t *testing.T) {
	input := &FindInput{
		CloseStatusFilter: &swf.CloseStatusFilter{
			Status: aws.String("ANYTHING"),
		},
	}

	mockSwf := &mocks.SWFAPI{}

	expectedClosedInput := &swf.ListClosedWorkflowExecutionsInput{
		Domain:            aws.String(dummyFsm().Domain),
		CloseStatusFilter: input.CloseStatusFilter,
	}
	mockSwf.MockOnTyped_ListClosedWorkflowExecutions(expectedClosedInput).Return(&swf.WorkflowExecutionInfos{}, nil)

	NewFSMClient(dummyFsm(), mockSwf).FindAll(input)
	mockSwf.AssertExpectations(t) // list open not called
}

func TestFindAll_CloseStatusFilterWithOverrideStatusFilteredToAll(t *testing.T) {
	input := &FindInput{
		CloseStatusFilter: &swf.CloseStatusFilter{
			Status: aws.String("ANYTHING"),
		},
		StatusFilter: FilterStatusAll,
	}

	mockSwf := &mocks.SWFAPI{}

	expectedOpenInput := &swf.ListOpenWorkflowExecutionsInput{
		Domain: aws.String(dummyFsm().Domain),
	}
	mockSwf.MockOnTyped_ListOpenWorkflowExecutions(expectedOpenInput).Return(&swf.WorkflowExecutionInfos{}, nil)

	expectedClosedInput := &swf.ListClosedWorkflowExecutionsInput{
		Domain:            aws.String(dummyFsm().Domain),
		CloseStatusFilter: input.CloseStatusFilter,
	}
	mockSwf.MockOnTyped_ListClosedWorkflowExecutions(expectedClosedInput).Return(&swf.WorkflowExecutionInfos{}, nil)

	NewFSMClient(dummyFsm(), mockSwf).FindAll(input)
	mockSwf.AssertExpectations(t) // list open and closed called
}

func TestFindAll_Max(t *testing.T) {
	input := &FindInput{
		MaximumPageSize: aws.Int64(int64(1)),
		StatusFilter:    FilterStatusAll,
	}

	mockSwf := &mocks.SWFAPI{}

	expectedOpenInput := &swf.ListOpenWorkflowExecutionsInput{
		Domain:          aws.String(dummyFsm().Domain),
		MaximumPageSize: input.MaximumPageSize,
	}
	mockSwf.MockOnTyped_ListOpenWorkflowExecutions(expectedOpenInput).Return(
		func(req *swf.ListOpenWorkflowExecutionsInput) *swf.WorkflowExecutionInfos {
			return &swf.WorkflowExecutionInfos{
				ExecutionInfos: []*swf.WorkflowExecutionInfo{
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("open-1"),
						},
						StartTimestamp: aws.Time(time.Now()),
					},
				},
			}
		}, nil)

	expectedClosedInput := &swf.ListClosedWorkflowExecutionsInput{
		Domain:          aws.String(dummyFsm().Domain),
		MaximumPageSize: input.MaximumPageSize,
	}
	mockSwf.MockOnTyped_ListClosedWorkflowExecutions(expectedClosedInput).Return(
		func(req *swf.ListClosedWorkflowExecutionsInput) *swf.WorkflowExecutionInfos {
			return &swf.WorkflowExecutionInfos{
				ExecutionInfos: []*swf.WorkflowExecutionInfo{
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("closed-1"),
						},
						StartTimestamp: aws.Time(time.Now()),
					},
				},
			}
		}, nil)

	output, err := NewFSMClient(dummyFsm(), mockSwf).FindAll(input)
	if err != nil {
		t.Fatal(err)
	}

	if len(output.ExecutionInfos) != 1 {
		t.Fatal(output.ExecutionInfos)
	}

	if *output.ExecutionInfos[0].Execution.WorkflowId != "closed-1" {
		t.Fatal(output.ExecutionInfos)
	}

	mockSwf.AssertExpectations(t) // both open and closed are called and return 1, but client filtered to only 1
}

func TestFindAll_ReverseOrder_Interleaving(t *testing.T) {
	input := &FindInput{
		ReverseOrder: aws.Bool(true),
		StatusFilter: FilterStatusAll,
	}

	mockSwf := &mocks.SWFAPI{}

	expectedOpenInput := &swf.ListOpenWorkflowExecutionsInput{
		Domain:       aws.String(dummyFsm().Domain),
		ReverseOrder: input.ReverseOrder,
	}
	mockSwf.MockOnTyped_ListOpenWorkflowExecutions(expectedOpenInput).Return(
		func(req *swf.ListOpenWorkflowExecutionsInput) *swf.WorkflowExecutionInfos {
			return &swf.WorkflowExecutionInfos{
				ExecutionInfos: []*swf.WorkflowExecutionInfo{
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("C-open"),
						},
						StartTimestamp: aws.Time(time.Now().Add(-1 * time.Hour)),
					},
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("A-open"),
						},
						StartTimestamp: aws.Time(time.Now().Add(-3 * time.Hour)),
					},
				},
			}
		}, nil)

	expectedClosedInput := &swf.ListClosedWorkflowExecutionsInput{
		Domain:       aws.String(dummyFsm().Domain),
		ReverseOrder: input.ReverseOrder,
	}
	mockSwf.MockOnTyped_ListClosedWorkflowExecutions(expectedClosedInput).Return(
		func(req *swf.ListClosedWorkflowExecutionsInput) *swf.WorkflowExecutionInfos {
			return &swf.WorkflowExecutionInfos{
				ExecutionInfos: []*swf.WorkflowExecutionInfo{
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("B-closed"),
						},
						StartTimestamp: aws.Time(time.Now().Add(-2 * time.Hour)),
					},
				},
			}
		}, nil)

	output, err := NewFSMClient(dummyFsm(), mockSwf).FindAll(input)
	if err != nil {
		t.Fatal(err)
	}

	if len(output.ExecutionInfos) != 3 {
		t.Fatal(output.ExecutionInfos)
	}

	if *output.ExecutionInfos[0].Execution.WorkflowId != "A-open" {
		t.Fatal(output.ExecutionInfos)
	}
	if *output.ExecutionInfos[1].Execution.WorkflowId != "B-closed" {
		t.Fatal(output.ExecutionInfos)
	}
	if *output.ExecutionInfos[2].Execution.WorkflowId != "C-open" {
		t.Fatal(output.ExecutionInfos)
	}

	mockSwf.AssertExpectations(t)
}

func TestFindAll_FindLatestByWorkflowID(t *testing.T) {
	mockSwf := &mocks.SWFAPI{}

	expectedOpenInput := &swf.ListOpenWorkflowExecutionsInput{
		Domain: aws.String(dummyFsm().Domain),
		ExecutionFilter: &swf.WorkflowExecutionFilter{
			WorkflowId: aws.String("workflow-A"),
		},
		MaximumPageSize: aws.Int64(int64(1)),
		ReverseOrder:    aws.Bool(false),
		StartTimeFilter: &swf.ExecutionTimeFilter{OldestDate: aws.Time(time.Unix(0, 0))},
	}
	mockSwf.MockOnTyped_ListOpenWorkflowExecutions(expectedOpenInput).Return(
		func(req *swf.ListOpenWorkflowExecutionsInput) *swf.WorkflowExecutionInfos {
			return &swf.WorkflowExecutionInfos{
				ExecutionInfos: []*swf.WorkflowExecutionInfo{
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("workflow-A"),
						},
						StartTimestamp: aws.Time(time.Now().Add(-1 * time.Hour)),
					},
				},
			}
		}, nil)

	exec, err := NewFSMClient(dummyFsm(), mockSwf).FindLatestByWorkflowID("workflow-A")
	if err != nil {
		t.Fatal(err)
	}

	if *exec.WorkflowId != "workflow-A" {
		t.Fatal(exec)
	}

	mockSwf.AssertExpectations(t)
}

func TestSegmentHistory(t *testing.T) {
	fsm := dummyFsm()

	var readyData interface{}
	readyData = TestData{States: []string{"ready data"}}

	var startData interface{}
	startData = TestData{States: []string{"start data"}}

	activityId := "activity-id"
	activityData := "activity data"

	itr := sliceHistoryIterator(t, []*swf.HistoryEvent{
		&swf.HistoryEvent{
			EventId:   aws.Int64(7),
			EventType: aws.String(swf.EventTypeActivityTaskScheduled),
			ActivityTaskScheduledEventAttributes: &swf.ActivityTaskScheduledEventAttributes{
				ActivityId: aws.String(activityId),
				Input:      aws.String(fsm.Serialize(activityData)),
				DecisionTaskCompletedEventId: aws.Int64(4),
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
	})

	actual, err := NewFSMClient(fsm, &mocks.SWFAPI{}).SegmentHistory(itr)
	if err != nil {
		t.Fatal(err)
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
			Events: []*HistorySegmentEvent{
				&HistorySegmentEvent{
					ID:   aws.Int64(7),
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
			Events: []*HistorySegmentEvent{
				&HistorySegmentEvent{
					ID:         aws.Int64(4),
					Type:       aws.String(swf.EventTypeDecisionTaskCompleted),
					References: []*int64{aws.Int64(7)},
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

func sliceHistoryIterator(t *testing.T, events []*swf.HistoryEvent) HistoryEventIterator {
	cursor := 0
	return func() (*swf.HistoryEvent, error) {
		if cursor >= len(events) {
			return nil, nil
		}
		event := events[cursor]
		cursor++
		t.Log(event)
		return event, nil
	}
}

func dummyFsm() *FSM {
	fsm := &FSM{
		Domain:           "client-test",
		Name:             "test-fsm",
		DataType:         TestData{},
		Serializer:       JSONStateSerializer{},
		systemSerializer: JSONStateSerializer{},
		allowPanics:      false,
	}

	fsm.AddInitialState(&FSMState{Name: "initial",
		Decider: func(ctx *FSMContext, h *swf.HistoryEvent, data interface{}) Outcome {
			return ctx.Stay(data, ctx.EmptyDecisions())
		},
	})

	return fsm
}
