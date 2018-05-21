package fsm

import (
	"os"
	"testing"

	"strings"

	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
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
	client := swf.New(session.New(config))

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
		SystemSerializer: JSONStateSerializer{},
		AllowPanics:      false,
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

	segments := []HistorySegment{}
	seg := fsmClient.NewHistorySegmentor()
	seg.OnSegment(func(segment HistorySegment) {
		segments = append(segments, segment)
	})
	seg.OnError(func(err error) {
		t.Fatal(err)
	})

	err = fsmClient.GetWorkflowExecutionHistoryPages(exec, seg.FromPage)
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

func TestFindAll_OpenPriorityWorkflow_ByTagIncludingContinuations(t *testing.T) {
	input := &FindInput{
		StatusFilter: FilterStatusOpenPriorityWorkflow,
		TagFilter: &swf.TagFilter{
			Tag: aws.String("T"),
		},
		CloseStatusFilter: &swf.CloseStatusFilter{
			Status: aws.String(swf.CloseStatusContinuedAsNew),
		},
		CloseTimeFilter: &swf.ExecutionTimeFilter{
			OldestDate: aws.Time(time.Now().Add(-1 * time.Minute)),
		},
	}

	mockSwf := &mocks.SWFAPI{}

	expectedOpenInput := &swf.ListOpenWorkflowExecutionsInput{
		Domain:    aws.String(dummyFsm().Domain),
		TagFilter: input.TagFilter,
	}
	expectedClosedInput := &swf.ListClosedWorkflowExecutionsInput{
		Domain:          aws.String(dummyFsm().Domain),
		TagFilter:       input.TagFilter,
		CloseTimeFilter: input.CloseTimeFilter,
		/* does not include close status filter in server request, but filtered later locally */
	}

	mockSwf.MockOnTyped_ListOpenWorkflowExecutions(expectedOpenInput).Return(
		func(req *swf.ListOpenWorkflowExecutionsInput) *swf.WorkflowExecutionInfos {
			return &swf.WorkflowExecutionInfos{
				ExecutionInfos: []*swf.WorkflowExecutionInfo{
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("A"),
							RunId:      aws.String("A-open"),
						},
						TagList:        []*string{aws.String("T")},
						StartTimestamp: aws.Time(time.Now().Add(-6 * time.Minute)),
					},
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("B"),
							RunId:      aws.String("B-open"),
						},
						TagList:        []*string{aws.String("T")},
						StartTimestamp: aws.Time(time.Now().Add(-5 * time.Minute)),
					},
				},
			}
		}, nil)

	mockSwf.MockOnTyped_ListClosedWorkflowExecutions(expectedClosedInput).Return(
		func(req *swf.ListClosedWorkflowExecutionsInput) *swf.WorkflowExecutionInfos {
			return &swf.WorkflowExecutionInfos{
				ExecutionInfos: []*swf.WorkflowExecutionInfo{
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("B"),
							RunId:      aws.String("B-closed-recently"),
						},
						TagList:        []*string{aws.String("T")},
						StartTimestamp: aws.Time(time.Now().Add(-4 * time.Minute)),
						CloseTimestamp: aws.Time(time.Now()),
						CloseStatus:    aws.String(swf.CloseStatusContinuedAsNew),
					},
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("C"),
							RunId:      aws.String("C-closed-recently"),
						},
						TagList:        []*string{aws.String("T")},
						StartTimestamp: aws.Time(time.Now().Add(-3 * time.Minute)),
						CloseTimestamp: aws.Time(time.Now()),
						CloseStatus:    aws.String(swf.CloseStatusContinuedAsNew),
					},
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("D"),
							RunId:      aws.String("D-closed-a-while-ago"),
						},
						TagList:        []*string{aws.String("T")},
						StartTimestamp: aws.Time(time.Now().Add(-3 * time.Minute)),
						CloseTimestamp: aws.Time(time.Now().Add(-2 * time.Minute)),
						CloseStatus:    aws.String(swf.CloseStatusContinuedAsNew),
					},
					&swf.WorkflowExecutionInfo{
						Execution: &swf.WorkflowExecution{
							WorkflowId: aws.String("E"),
							RunId:      aws.String("E-closed-recently-completed"),
						},
						TagList:        []*string{aws.String("T")},
						StartTimestamp: aws.Time(time.Now().Add(-3 * time.Minute)),
						CloseTimestamp: aws.Time(time.Now()),
						CloseStatus:    aws.String(swf.CloseStatusCompleted),
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

	if *output.ExecutionInfos[0].Execution.WorkflowId != "C" {
		t.Fatal(output.ExecutionInfos)
	}
	if *output.ExecutionInfos[0].Execution.RunId != "C-closed-recently" {
		t.Fatal(output.ExecutionInfos)
	}

	if *output.ExecutionInfos[1].Execution.WorkflowId != "B" {
		t.Fatal(output.ExecutionInfos)
	}
	if *output.ExecutionInfos[1].Execution.RunId != "B-open" {
		t.Fatal(output.ExecutionInfos)
	}

	if *output.ExecutionInfos[2].Execution.WorkflowId != "A" {
		t.Fatal(output.ExecutionInfos)
	}
	if *output.ExecutionInfos[2].Execution.RunId != "A-open" {
		t.Fatal(output.ExecutionInfos)
	}

	mockSwf.AssertExpectations(t)
}

func TestEmptyFindLatestByWorkflowID(t *testing.T) {
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
	expectedClosedInput := &swf.ListClosedWorkflowExecutionsInput{
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
				ExecutionInfos: []*swf.WorkflowExecutionInfo{},
			}
		}, nil,
	)
	mockSwf.MockOnTyped_ListClosedWorkflowExecutions(expectedClosedInput).Return(
		func(req *swf.ListClosedWorkflowExecutionsInput) *swf.WorkflowExecutionInfos {
			return &swf.WorkflowExecutionInfos{
				ExecutionInfos: []*swf.WorkflowExecutionInfo{},
			}
		}, nil,
	)

	if _, err := NewFSMClient(dummyFsm(), mockSwf).FindLatestByWorkflowID("workflow-A"); err == nil {
		t.Fatal("expected an error when no results")
	}
	mockSwf.AssertExpectations(t)
}

func dummyFsm() *FSM {
	fsm := &FSM{
		Domain:           "client-test",
		Name:             "test-fsm",
		DataType:         TestData{},
		Serializer:       JSONStateSerializer{},
		SystemSerializer: JSONStateSerializer{},
		AllowPanics:      false,
	}

	fsm.AddInitialState(&FSMState{Name: "initial",
		Decider: func(ctx *FSMContext, h *swf.HistoryEvent, data interface{}) Outcome {
			return ctx.Stay(data, ctx.EmptyDecisions())
		},
	})

	return fsm
}
