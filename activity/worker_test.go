package activity

import (
	"errors"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/sclasen/swfsm/fsm"
	. "github.com/sclasen/swfsm/log"
	"github.com/sclasen/swfsm/migrator"
	. "github.com/sclasen/swfsm/sugar"
	"github.com/stretchr/testify/assert"
)

type MockSWF struct {
	Activity     *swf.PollForActivityTaskOutput
	Failed       bool
	FailedReason *string
	Completed    *string
	CompletedSet bool
	History      *swf.GetWorkflowExecutionHistoryOutput
	Canceled     bool
	SignalFail   bool
}

func (m *MockSWF) RecordActivityTaskHeartbeat(req *swf.RecordActivityTaskHeartbeatInput) (*swf.RecordActivityTaskHeartbeatOutput, error) {
	return &swf.RecordActivityTaskHeartbeatOutput{
		CancelRequested: &m.Canceled,
	}, nil
}
func (*MockSWF) RespondActivityTaskCanceled(req *swf.RespondActivityTaskCanceledInput) (*swf.RespondActivityTaskCanceledOutput, error) {
	return nil, nil
}
func (m *MockSWF) RespondActivityTaskCompleted(req *swf.RespondActivityTaskCompletedInput) (*swf.RespondActivityTaskCompletedOutput, error) {
	m.Completed = req.Result
	m.CompletedSet = true
	return nil, nil
}
func (m *MockSWF) RespondActivityTaskFailed(req *swf.RespondActivityTaskFailedInput) (*swf.RespondActivityTaskFailedOutput, error) {
	m.Failed = true
	m.FailedReason = req.Reason
	return nil, nil
}
func (m *MockSWF) PollForActivityTask(req *swf.PollForActivityTaskInput) (*swf.PollForActivityTaskOutput, error) {
	return m.Activity, nil
}

func (m *MockSWF) GetWorkflowExecutionHistory(req *swf.GetWorkflowExecutionHistoryInput) (*swf.GetWorkflowExecutionHistoryOutput, error) {
	return m.History, nil
}

func (m *MockSWF) SignalWorkflowExecution(req *swf.SignalWorkflowExecutionInput) (*swf.SignalWorkflowExecutionOutput, error) {
	if m.SignalFail {
		return nil, errors.New("signaling failed")
	}
	return nil, nil
}

func ExampleActivityWorker() {

	var swfOps SWFOps

	taskList := "aTaskListSharedBetweenTaskOneAndTwo"

	handleTask1 := func(task *swf.PollForActivityTaskOutput, input interface{}) (interface{}, error) {
		return input, nil
	}

	handleTask2 := func(task *swf.PollForActivityTaskOutput, input interface{}) (interface{}, error) {
		return input, nil
	}

	handler1 := &ActivityHandler{Activity: "one", HandlerFunc: handleTask1}

	handler2 := &ActivityHandler{Activity: "two", HandlerFunc: handleTask2}

	worker := &ActivityWorker{
		Domain:     "swf-domain",
		Serializer: fsm.JSONStateSerializer{},
		TaskList:   taskList,
		SWF:        swfOps,
		Identity:   "test-activity-worker",
	}

	worker.AddHandler(handler1)

	worker.AddHandler(handler2)

	go worker.Start()

}

func ExampleTypedActivityWorker() {

	var swfOps SWFOps

	taskList := "aTaskListSharedBetweenTaskOneAndTwo"

	worker := &ActivityWorker{
		Domain:     "swf-domain",
		Serializer: fsm.JSONStateSerializer{},
		TaskList:   taskList,
		SWF:        swfOps,
		Identity:   "test-activity-worker",
	}

	var activities Activities

	activities = MockActivities{}

	worker.AddHandler(NewActivityHandler("one", activities.Task1))

	worker.AddHandler(NewActivityHandler("two", activities.Task2))

	go worker.Start()

}

type Input1 struct {
	Data string
}

type Output1 struct {
	Data string
}

type Input2 struct {
	Data2 string
}

type Output2 struct {
	Data2 string
}

//We define the operations our activity worker will handle in an interface
//Then it is easy to provide a mocked impl
type Activities interface {
	Task1(*swf.PollForActivityTaskOutput, *Input1) (*Output1, error)
	Task2(*swf.PollForActivityTaskOutput, *Input2) (*Output2, error)
}

type MockActivities struct{}

func (m MockActivities) Task1(task *swf.PollForActivityTaskOutput, in *Input1) (*Output1, error) {
	if rand.Intn(10) < 4 {
		Log.Printf("TASK 1 OK")
		return &Output1{Data: in.Data}, nil
	}
	Log.Printf("TASK 1 FAIL")
	return nil, errors.New("FAILED")
}

func (m MockActivities) Task2(task *swf.PollForActivityTaskOutput, in *Input2) (*Output2, error) {
	if rand.Intn(10) < 4 {
		Log.Printf("TASK 2 OK")
		return &Output2{Data2: in.Data2}, nil
	}
	Log.Printf("TASK 2 FAIL")
	return nil, errors.New("FAILED")
}

type WorkerFSM struct {
	*fsm.FSM
	done chan struct{}
}

func NewWorkerFSM(client *swf.SWF, done chan struct{}) *WorkerFSM {
	wfsm := &WorkerFSM{
		FSM: &fsm.FSM{
			DataType: Input1{},
			Domain:   "worker-test-domain",
			SWF:      client,
			TaskList: "worker-fsm",
		},
		done: done,
	}
	wfsm.AddInitialState(&fsm.FSMState{Name: "Initial", Decider: wfsm.Initial()})
	wfsm.AddState(&fsm.FSMState{Name: "One", Decider: wfsm.One()})
	wfsm.AddState(&fsm.FSMState{Name: "Two", Decider: wfsm.Two()})
	return wfsm
}

func (w *WorkerFSM) Initial() fsm.Decider {
	return fsm.OnStarted(
		fsm.AddDecision(w.actOne),
		fsm.Transition("One"),
	)
}

func (w *WorkerFSM) One() fsm.Decider {
	return fsm.NewComposedDecider(
		fsm.OnActivityFailedTimedOutCanceled("one", fsm.AddDecision(w.actOne)),
		fsm.OnActivityCompleted("one",
			fsm.UpdateState(w.afterOne),
			fsm.AddDecision(w.actTwo),
			fsm.Transition("Two"),
		),
	)
}

func (w *WorkerFSM) Two() fsm.Decider {
	return fsm.NewComposedDecider(
		fsm.OnActivityFailedTimedOutCanceled("two", fsm.AddDecision(w.actTwo)),
		fsm.OnActivityCompleted("two",
			fsm.UpdateState(w.afterTwo),
			fsm.CompleteWorkflow(),
		),
	)
}

func (w *WorkerFSM) actOne(ctx *fsm.FSMContext, h *swf.HistoryEvent, data interface{}) *swf.Decision {
	return w.activity(ctx, "one", &Input1{Data: "one"})
}

func (w *WorkerFSM) afterOne(ctx *fsm.FSMContext, h *swf.HistoryEvent, data interface{}) {
	result := new(Output1)
	ctx.EventData(h, result)
	data.(*Input1).Data = result.Data
}

func (w *WorkerFSM) afterTwo(ctx *fsm.FSMContext, h *swf.HistoryEvent, data interface{}) {
	result := new(Output2)
	ctx.EventData(h, result)
	data.(*Input1).Data = result.Data2
	w.done <- struct{}{}
}

func (w *WorkerFSM) actTwo(ctx *fsm.FSMContext, h *swf.HistoryEvent, data interface{}) *swf.Decision {
	return w.activity(ctx, "two", &Input1{Data: "two"})
}

func (w *WorkerFSM) activity(ctx *fsm.FSMContext, name string, input interface{}) *swf.Decision {
	var serialized *string
	if input != nil {
		serialized = S(ctx.Serialize(input))
	}
	return &swf.Decision{
		DecisionType: S(swf.DecisionTypeScheduleActivityTask),
		ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
			ActivityId:             S(name),
			ActivityType:           &swf.ActivityType{Name: S(name), Version: S(name)},
			Input:                  serialized,
			HeartbeatTimeout:       S("30"),
			ScheduleToStartTimeout: S("NONE"),
			ScheduleToCloseTimeout: S("NONE"),
			StartToCloseTimeout:    S("NONE"),
			TaskList:               &swf.TaskList{Name: S("aTaskListSharedBetweenTaskOneAndTwo")},
		},
	}
}

func TestTypedActivityWorker(t *testing.T) {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		Log.Printf("WARNING: NO AWS CREDS SPECIFIED, SKIPPING MIGRATIONS TEST")
		return
	}

	config := &aws.Config{
		Credentials: credentials.NewEnvCredentials(),
		Region:      aws.String("us-east-1"),
	}

	client := swf.New(session.New(config))

	domain := "worker-test-domain"

	req := swf.RegisterDomainInput{
		Name:                                   aws.String(domain),
		Description:                            aws.String("test domain"),
		WorkflowExecutionRetentionPeriodInDays: aws.String("30"),
	}

	d := migrator.DomainMigrator{
		RegisteredDomains: []swf.RegisterDomainInput{req},
		Client:            client,
	}

	d.Migrate()

	workflow := "worker-test-workflow"
	version := "worker-test-workflow-version"

	wreq := swf.RegisterWorkflowTypeInput{
		Name:            &workflow,
		Description:     aws.String("test workflow migration"),
		Version:         &version,
		Domain:          aws.String(domain),
		DefaultTaskList: &swf.TaskList{Name: S("worker-fsm")},
	}

	w := migrator.WorkflowTypeMigrator{
		RegisteredWorkflowTypes: []swf.RegisterWorkflowTypeInput{wreq},
		Client:                  client,
	}

	w.Migrate()

	one := swf.RegisterActivityTypeInput{
		Name:        aws.String("one"),
		Description: aws.String("worker test activity 1"),
		Version:     aws.String("one"),
		Domain:      aws.String(domain),
	}

	two := swf.RegisterActivityTypeInput{
		Name:        aws.String("two"),
		Description: aws.String("worker test activity 2"),
		Version:     aws.String("two"),
		Domain:      aws.String(domain),
	}

	a := migrator.ActivityTypeMigrator{
		RegisteredActivityTypes: []swf.RegisterActivityTypeInput{one, two},
		Client:                  client,
	}

	a.Migrate()

	taskList := "aTaskListSharedBetweenTaskOneAndTwo"

	worker := &ActivityWorker{
		Domain:     domain,
		Serializer: fsm.JSONStateSerializer{},
		TaskList:   taskList,
		SWF:        client,
		Identity:   "test-activity-worker",
	}

	//This is where the actual worker config code is
	var activities Activities

	activities = MockActivities{}

	worker.AddHandler(NewActivityHandler("one", activities.Task1))

	worker.AddHandler(NewActivityHandler("two", activities.Task2))

	go worker.Start()
	//yep thats all the worker config code

	done := make(chan struct{})

	go NewWorkerFSM(client, done).Start()

	_, err := client.StartWorkflowExecution(&swf.StartWorkflowExecutionInput{
		Domain:       S(domain),
		WorkflowId:   S("worker-test"),
		WorkflowType: &swf.WorkflowType{Name: S(workflow), Version: S(version)},
		Input:        S("{}"),
		ExecutionStartToCloseTimeout: S("90"),
		TaskStartToCloseTimeout:      S("10"),
		ChildPolicy:                  S("ABANDON"),
	})

	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
	case <-time.After(1 * time.Minute):
		t.Fatal("One Minute Elapsed, not done with workflow")
	}

}

func TestStringHandler(t *testing.T) {
	ops := &MockSWF{}
	worker := &ActivityWorker{
		SWF: ops,
	}

	worker.Init()
	worker.AllowPanics = true

	handler := func(task *swf.PollForActivityTaskOutput, input string) (string, error) {
		return input + "Out", nil
	}

	nilHandler := func(task *swf.PollForActivityTaskOutput, input string) (*swf.PollForActivityTaskOutput, error) {
		return nil, nil
	}

	worker.AddHandler(NewActivityHandler("activity", handler))
	worker.AddHandler(NewActivityHandler("nilactivity", nilHandler))
	worker.HandleActivityTask(&swf.PollForActivityTaskOutput{
		WorkflowExecution: &swf.WorkflowExecution{},
		ActivityType:      &swf.ActivityType{Name: S("activity")},
		Input:             S("theInput"),
	})

	if !ops.CompletedSet || *ops.Completed != "theInputOut" {
		t.Fatal("Not Completed", ops.Completed)
	}

	ops.Completed = nil
	ops.CompletedSet = false

	worker.HandleActivityTask(&swf.PollForActivityTaskOutput{
		WorkflowExecution: &swf.WorkflowExecution{},
		ActivityType:      &swf.ActivityType{Name: S("nilactivity")},
		Input:             S("theInput"),
	})

	if !ops.CompletedSet || ops.Completed != nil {
		t.Fatal("Not Completed", ops.Completed)
	}

}

func TestBackoff(t *testing.T) {
	serializer := fsm.JSONStateSerializer{}

	correlator := new(fsm.EventCorrelator)
	correlator.ActivityAttempts = make(map[string]int)
	correlator.ActivityAttempts["the-id"] = 3
	s, _ := serializer.Serialize(correlator)

	history := &swf.GetWorkflowExecutionHistoryOutput{
		Events: []*swf.HistoryEvent{
			{
				EventType: S(swf.EventTypeMarkerRecorded),
				MarkerRecordedEventAttributes: &swf.MarkerRecordedEventAttributes{
					MarkerName: S(fsm.CorrelatorMarker),
					Details:    S(s),
				},
			},
		},
	}
	ops := &MockSWF{
		History: history,
	}
	worker := &ActivityWorker{
		SWF:               ops,
		BackoffOnFailure:  true,
		MaxBackoffSeconds: 5,
		Serializer:        fsm.JSONStateSerializer{},
	}

	failed := make(chan struct{})
	go func() {
		worker.fail(&swf.PollForActivityTaskOutput{
			WorkflowExecution: &swf.WorkflowExecution{},
			ActivityType:      &swf.ActivityType{Name: S("activity")},
			ActivityId:        S("the-id"),
			Input:             S("theInput"),
		}, errors.New("the error"))
		failed <- struct{}{}
	}()

	select {
	case <-time.After(2 * time.Second):
	case <-failed:
		t.Fatal("fail finished before 2 seconds")
	}
}

func TestFailWhenErrorMoreThanMaxCharactersExpectsErrorTruncated(t *testing.T) {
	// arrange
	ops := &MockSWF{}
	worker := &ActivityWorker{
		SWF:        ops,
		Serializer: fsm.JSONStateSerializer{},
	}
	longErrorMessage := `Lorem ipsum dolor sitamet, consectetur adipisicing elit, sed do eiusmod
tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam,
quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo
consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum
dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident,
sunt in culpa qui officia deserunt mollit anim id est laborum.`

	// act
	worker.fail(&swf.PollForActivityTaskOutput{
		WorkflowExecution: &swf.WorkflowExecution{},
		ActivityType:      &swf.ActivityType{Name: S("activity")},
		ActivityId:        S("the-id"),
		Input:             S("theInput"),
	}, errors.New(longErrorMessage))

	// assert
	assert.EqualValues(t, FailureReasonMaxChars, len(*ops.FailedReason),
		"Expected long error message to be truncated to the max characters allowed.")
	assert.Equal(t, longErrorMessage[:FailureReasonMaxChars], *ops.FailedReason,
		"Expected failure reason to match the first "+strconv.Itoa(FailureReasonMaxChars)+
			" characters of the long error message")
}

func TestFailWhenErrorLessThanMaxCharactersExpectsErrorNotTruncated(t *testing.T) {
	// arrange
	ops := &MockSWF{}
	worker := &ActivityWorker{
		SWF:        ops,
		Serializer: fsm.JSONStateSerializer{},
	}
	shortErrorMessage := `Lorem ipsum dolor sitamet`

	// act
	worker.fail(&swf.PollForActivityTaskOutput{
		WorkflowExecution: &swf.WorkflowExecution{},
		ActivityType:      &swf.ActivityType{Name: S("activity")},
		ActivityId:        S("the-id"),
		Input:             S("theInput"),
	}, errors.New(shortErrorMessage))

	// assert
	assert.EqualValues(t, len(shortErrorMessage), len(*ops.FailedReason),
		"Expected short error message to not be truncated because it is below the max character limit.")
	assert.Equal(t, shortErrorMessage, *ops.FailedReason,
		"Expected failure reason to match the short error message")
}
