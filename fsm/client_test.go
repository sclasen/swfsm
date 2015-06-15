package fsm

import (
	"log"
	"os"
	"testing"

	"strings"

	"code.google.com/p/go-uuid/uuid"
	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/aws/credentials"
	"github.com/awslabs/aws-sdk-go/service/swf"
	"github.com/sclasen/swfsm/enums/swf"
	"github.com/sclasen/swfsm/migrator"
)

func TestClient(t *testing.T) {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		log.Printf("WARNING: NO AWS CREDS SPECIFIED, SKIPPING MIGRATIONS TEST")
		return
	}

	config := &aws.Config{
		Credentials: credentials.NewEnvCredentials(),
		Region:      "us-east-1",
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
			if *h.EventType == enums.EventTypeWorkflowExecutionSignaled {
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
			if *info.Execution.WorkflowID == workflow {
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
}

func TestStringDoesntSerialize(t *testing.T) {

	fsm := &FSM{
		Domain:           "client-test",
		Name:             "test-fsm",
		DataType:         TestData{},
		Serializer:       JSONStateSerializer{},
		systemSerializer: JSONStateSerializer{},
		allowPanics:      false,
	}

	swf := &swf.SWF{}
	mock := &MockSWF{
		t:   t,
		SWF: swf,
	}

	fsmClient := NewFSMClient(fsm, mock)

	fsmClient.Signal("wf", "signal", "simple")

}

type MockSWF struct {
	t *testing.T
	*swf.SWF
}

func (m *MockSWF) SignalWorkflowExecution(req *swf.SignalWorkflowExecutionInput) (*swf.SignalWorkflowExecutionOutput, error) {
	if strings.Contains(*req.Input, "\"") {
		m.t.Fatal("simple string input has quotes")
	}
	if *req.Input != "simple" {
		m.t.Fatal("not simele")
	}
	return nil, nil
}
