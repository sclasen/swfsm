package activity

import (
	"errors"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/swf"
	"github.com/sclasen/swfsm/fsm"
	"github.com/sclasen/swfsm/migrator"
	. "github.com/sclasen/swfsm/sugar"
)

type MockSWF struct {
	Activity *swf.ActivityTask
}

func (MockSWF) RecordActivityTaskHeartbeat(req *swf.RecordActivityTaskHeartbeatInput) (resp *swf.ActivityTaskStatus, err error) {
	return nil, nil
}
func (MockSWF) RespondActivityTaskCanceled(req *swf.RespondActivityTaskCanceledInput) (err error) {
	return nil
}
func (MockSWF) RespondActivityTaskCompleted(req *swf.RespondActivityTaskCompletedInput) (err error) {
	return nil
}
func (MockSWF) RespondActivityTaskFailed(req *swf.RespondActivityTaskFailedInput) (err error) {
	return nil
}
func (m MockSWF) PollForActivityTask(req *swf.PollForActivityTaskInput) (resp *swf.ActivityTask, err error) {
	return m.Activity, nil
}

func ExampleActivityWorker() {

	var swfOps SWFOps

	taskList := "aTaskListSharedBetweenTaskOneAndTwo"

	handleTask1 := func(task *swf.ActivityTask, input interface{}) (interface{}, error) {
		return input, nil
	}

	handleTask2 := func(task *swf.ActivityTask, input interface{}) (interface{}, error) {
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
	Task1(*swf.ActivityTask, *Input1) (*Output1, error)
	Task2(*swf.ActivityTask, *Input2) (*Output2, error)
}

type MockActivities struct{}

func (m MockActivities) Task1(task *swf.ActivityTask, in *Input1) (*Output1, error) {
	if rand.Intn(10) < 4 {
		log.Printf("TASK 1 OK")
		return &Output1{Data: in.Data}, nil
	}
	log.Printf("TASK 1 FAIL")
	return nil, errors.New("FAILED")
}

func (m MockActivities) Task2(task *swf.ActivityTask, in *Input2) (*Output2, error) {
	if rand.Intn(10) < 4 {
		log.Printf("TASK 2 OK")
		return &Output2{Data2: in.Data2}, nil
	}
	log.Printf("TASK 2 FAIL")
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

func (w *WorkerFSM) actOne(ctx *fsm.FSMContext, h swf.HistoryEvent, data interface{}) swf.Decision {
	return w.activity(ctx, "one", &Input1{Data: "one"})
}

func (w *WorkerFSM) afterOne(ctx *fsm.FSMContext, h swf.HistoryEvent, data interface{}) {
	result := new(Output1)
	ctx.EventData(h, result)
	data.(*Input1).Data = result.Data
}

func (w *WorkerFSM) afterTwo(ctx *fsm.FSMContext, h swf.HistoryEvent, data interface{}) {
	result := new(Output2)
	ctx.EventData(h, result)
	data.(*Input1).Data = result.Data2
	w.done <- struct{}{}
}

func (w *WorkerFSM) actTwo(ctx *fsm.FSMContext, h swf.HistoryEvent, data interface{}) swf.Decision {
	return w.activity(ctx, "two", &Input1{Data: "two"})
}

func (w *WorkerFSM) activity(ctx *fsm.FSMContext, name string, input interface{}) swf.Decision {
	var serialized *string
	if input != nil {
		serialized = S(ctx.Serialize(input))
	}
	return swf.Decision{
		DecisionType: S(swf.DecisionTypeScheduleActivityTask),
		ScheduleActivityTaskDecisionAttributes: &swf.ScheduleActivityTaskDecisionAttributes{
			ActivityID:             S(name),
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
		log.Printf("WARNING: NO AWS CREDS SPECIFIED, SKIPPING MIGRATIONS TEST")
		return
	}

	creds, _ := aws.EnvCreds()
	client := swf.New(creds, "us-east-1", nil)

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
		WorkflowID:   S("worker-test"),
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
