package testing

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/aws/aws-sdk-go/service/swf/swfiface"
	"github.com/sclasen/swfsm/activity"
	"github.com/sclasen/swfsm/fsm"
	. "github.com/sclasen/swfsm/sugar"
)

//ugh thought testing.T was a TestAdapter, we usually use gocheck which is.
type TestingAdapter struct {
	*testing.T
	Name string
}

func (t TestingAdapter) TestName() string {
	return t.Name
}

//Data Type for the ExampleFSM
type ExampleData struct {
	Data string
}

//Interface describing the activites for the ExampleFSM
type ExampleActivities interface {
	ExampleActivity(*swf.PollForActivityTaskOutput, *ExampleData) *ExampleData
}

//Stubbed out activities, we are just testing the FSM
type ExampleStubActivities struct{}

func (e *ExampleStubActivities) ExampleActivity(t *swf.PollForActivityTaskOutput, d *ExampleData) *ExampleData {
	return d
}

type ExampleFSMTestSuite struct {
	fsm            *fsm.FSM
	client         fsm.FSMClient
	testListener   *TestListener
	activityWorker *activity.ActivityWorker
}

//This would actually wire up your FSM impl
func ExampleFSM() *fsm.FSM {
	return nil
}

//This would actually wire up your SWF client
func ExampleSWF() swfiface.SWFAPI {
	return &swf.SWF{}
}

//This would actually wire up your activity worker
func ExampleActivityWorker(a ExampleActivities) *activity.ActivityWorker {
	return nil
}

func (n *ExampleFSMTestSuite) ExampleSetUpTest(t *testing.T) {
	//The FSM under test
	n.fsm = ExampleFSM()
	//Interactions with other FSMs are stubbed out. StubFSMs are started so that the
	//fsm under test has a place to send signal to if necessary.
	stubFsm := StubFSM("swf-domain-name", ExampleSWF())
	//Client used to start test workflows. Note we are actually testing against real SWF here, so it needs real creds
	//and workflows and activities need to be registered already
	n.client = fsm.NewFSMClient(n.fsm, ExampleSWF())
	//activity worker with stubbed out activity handlers
	n.activityWorker = ExampleActivityWorker(&ExampleStubActivities{})

	testConfig := TestConfig{
		Testing: TestingAdapter{t, "test-name"},
		FSM:     n.fsm,
		StubFSM: stubFsm,
		Workers: []*activity.ActivityWorker{n.activityWorker},
		/*
			if the FSM under test starts a 'stubbed-workflow-type',
			 we replace it with a stub workflow that does nothing but recieve signals
		*/
		StubbedWorkflows:   []string{"stubbed-workflow-type"},
		DefaultWaitTimeout: 5,
		/*
			If true, we make all stubbed activities fail once before completing so that
			you can test failure recovery/retry
		*/
		FailActivitiesOnce: true,
	}
	n.testListener = NewTestListener(testConfig)

	go n.fsm.Start()
	go stubFsm.Start()
	go n.activityWorker.Start()

}

func (n *ExampleFSMTestSuite) ExampleTest(t *testing.T) {

	workflowID := "workflow-id"

	n.testListener.RegisterHistoryInterest(workflowID)
	n.testListener.RegisterStateInterest(workflowID)
	n.testListener.RegisterDecisionInterest(workflowID)
	n.testListener.RegisterDataInterest(workflowID)

	_, err := n.client.Start(swf.StartWorkflowExecutionInput{
		WorkflowId:                   S(workflowID),
		WorkflowType:                 &swf.WorkflowType{Name: S("workflow-type"), Version: S("1")},
		TaskStartToCloseTimeout:      S("5"),
		ExecutionStartToCloseTimeout: S("360"), //close in 6 minutes since this is a test.
		TaskList:                     &swf.TaskList{Name: S(n.testListener.TestId)},
	}, workflowID, &ExampleData{Data: "a-data"})

	//assert ok
	if err != nil {
		t.Fatal(err)
	}

	//if the FSM does not get to 'some-state-to-wait-for' in 5 seconds test panics/fails
	n.testListener.AwaitState(workflowID, "some-state-to-wait-for")
	//if the FSM does not start a child workflow (which will be stubbed here) in 5 seconds test panics/fail*
	n.testListener.AwaitDecision(workflowID, func(d *swf.Decision) bool {
		return *d.DecisionType == swf.DecisionTypeStartChildWorkflowExecution &&
			*d.StartChildWorkflowExecutionDecisionAttributes.WorkflowType.Name == "stubbed-workflow-type"
	})
	//if the FSM does not signal the started child workflow in 5 seconds, test fails/panics
	n.testListener.AwaitEvent(workflowID, func(e *swf.HistoryEvent) bool {
		return *e.EventType == swf.EventTypeExternalWorkflowExecutionSignaled &&
			*e.ExternalWorkflowExecutionSignaledEventAttributes.WorkflowExecution.WorkflowId == "stub-workflow-id"
	})

	//simulate a signal back from the stub workflow
	n.client.Signal(workflowID, "a-signal-from-the-stub", nil)

	n.testListener.AwaitEvent(workflowID, func(e *swf.HistoryEvent) bool {
		return *e.EventType == swf.EventTypeWorkflowExecutionSignaled &&
			*e.WorkflowExecutionSignaledEventAttributes.SignalName == "a-signal-from-the-stub"
	})

	n.client.RequestCancel(workflowID)

	n.testListener.AwaitEvent(workflowID, func(e *swf.HistoryEvent) bool {
		return *e.EventType == swf.EventTypeExternalWorkflowExecutionCancelRequested
	})

	n.testListener.AwaitDecision(workflowID, func(d *swf.Decision) bool {
		return *d.DecisionType == swf.DecisionTypeCancelWorkflowExecution
	})
}
