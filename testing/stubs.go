package testing

import (
	"fmt"
	"log"

	"github.com/awslabs/aws-sdk-go/gen/swf"
	"github.com/sclasen/swfsm/fsm"
	. "github.com/sclasen/swfsm/sugar"
)

const (
	StubWorkflow      = "stub"
	ShortStubWorkflow = "stub"
	StubVersion       = "1"
)

var (
	StubTaskList      = &swf.TaskList{Name: S(fmt.Sprintf("%s->%s", StubWorkflow, StubVersion))}
	ShortStubTaskList = &swf.TaskList{Name: S(fmt.Sprintf("%s->%s", ShortStubWorkflow, StubVersion))}
)

type DecisionOutcome struct {
	DecisionTask *swf.DecisionTask
	State        string
	Decisions    []swf.Decision
}

func StubFSM(domain string, client fsm.SWFOps, outcomes chan DecisionOutcome) *fsm.FSM {
	f := &fsm.FSM{
		SWF:                client,
		DataType:           make(map[string]interface{}),
		Domain:             domain,
		Name:               StubWorkflow,
		Serializer:         fsm.JSONStateSerializer{},
		TaskList:           *StubTaskList.Name,
		ReplicationHandler: TestReplicator(outcomes),
	}

	f.AddInitialState(&fsm.FSMState{Name: "Initial", Decider: StubState()})
	return f
}

func StubState() fsm.Decider {
	return func(ctx *fsm.FSMContext, h swf.HistoryEvent, data interface{}) fsm.Outcome {
		log.Printf("at=stub-event event=%+v", PrettyHistoryEvent(h))
		return ctx.Stay(data, ctx.EmptyDecisions())
	}
}

func ShortStubFSM(domain string, client fsm.SWFOps, outcomes chan DecisionOutcome) *fsm.FSM {
	f := &fsm.FSM{
		SWF:                client,
		DataType:           make(map[string]interface{}),
		Domain:             domain,
		Name:               ShortStubWorkflow,
		Serializer:         fsm.JSONStateSerializer{},
		TaskList:           *StubTaskList.Name,
		ReplicationHandler: TestReplicator(outcomes),
	}

	f.AddInitialState(&fsm.FSMState{Name: "Initial", Decider: ShortStubState()})
	return f
}

func ShortStubState() fsm.Decider {
	return func(ctx *fsm.FSMContext, h swf.HistoryEvent, data interface{}) fsm.Outcome {
		log.Printf("at=short-stub-event event=%+v", PrettyHistoryEvent(h))
		return ctx.CompleteWorkflow(data)
	}
}

//intercept any attempts to start a workflow and launch the stub workflow instead.
func TestInterceptor(testID string, stubbedWorkflows, stubbedShortWorkflows []string) *fsm.FuncInterceptor {
	stubbed := make(map[string]struct{})
	stubbedShort := make(map[string]struct{})
	v := struct{}{}
	for _, s := range stubbedWorkflows {
		stubbed[s] = v
	}
	for _, s := range stubbedShortWorkflows {
		stubbedShort[s] = v
	}
	return &fsm.FuncInterceptor{
		AfterDecisionFn: func(decision *swf.DecisionTask, ctx *fsm.FSMContext, outcome *fsm.Outcome) {
			for _, d := range outcome.Decisions {
				switch *d.DecisionType {
				case swf.DecisionTypeStartChildWorkflowExecution:
					if _, ok := stubbed[*d.StartChildWorkflowExecutionDecisionAttributes.WorkflowType.Name]; ok {
						d.StartChildWorkflowExecutionDecisionAttributes.WorkflowType.Name = S(StubWorkflow)
						d.StartChildWorkflowExecutionDecisionAttributes.WorkflowType.Version = S(StubVersion)
						d.StartChildWorkflowExecutionDecisionAttributes.ExecutionStartToCloseTimeout = S("360")
						d.StartChildWorkflowExecutionDecisionAttributes.TaskList = StubTaskList
					}
					if _, ok := stubbedShort[*d.StartChildWorkflowExecutionDecisionAttributes.WorkflowType.Name]; ok {
						d.StartChildWorkflowExecutionDecisionAttributes.WorkflowType.Name = S(ShortStubWorkflow)
						d.StartChildWorkflowExecutionDecisionAttributes.WorkflowType.Version = S(StubVersion)
						d.StartChildWorkflowExecutionDecisionAttributes.ExecutionStartToCloseTimeout = S("360")
						d.StartChildWorkflowExecutionDecisionAttributes.TaskList = ShortStubTaskList
					}
				case swf.DecisionTypeScheduleActivityTask:
					d.ScheduleActivityTaskDecisionAttributes.TaskList.Name = S(*d.ScheduleActivityTaskDecisionAttributes.TaskList.Name + testID)
				}
			}
		},
	}
}

func TestReplicator(decisionOutcomes chan DecisionOutcome) fsm.ReplicationHandler {
	return func(ctx *fsm.FSMContext, task *swf.DecisionTask, outcome *swf.RespondDecisionTaskCompletedInput, state *fsm.SerializedState) error {
		decisionOutcomes <- DecisionOutcome{State: state.StateName, DecisionTask: task, Decisions: outcome.Decisions}
		return nil
	}
}

func TestSWF(client fsm.ClientSWFOps, stubbedWorkflow ...string) fsm.ClientSWFOps {
	stubbed := make(map[string]struct{})
	v := struct{}{}
	for _, s := range stubbedWorkflow {
		stubbed[s] = v
	}
	return &StubSWFClient{
		ClientSWFOps:     client,
		stubbedWorkflows: stubbed,
	}
}

//intercept any attempts to start a workflow and launch the stub workflow instead.
type StubSWFClient struct {
	fsm.ClientSWFOps
	stubbedWorkflows map[string]struct{}
}

func (s *StubSWFClient) StartWorkflowExecution(req *swf.StartWorkflowExecutionInput) (resp *swf.Run, err error) {
	if _, ok := s.stubbedWorkflows[*req.WorkflowType.Name]; ok {
		req.WorkflowType.Name = S(StubWorkflow)
		req.WorkflowType.Version = S(StubVersion)
		req.ExecutionStartToCloseTimeout = S("360")
		req.TaskList = StubTaskList
	}
	return s.ClientSWFOps.StartWorkflowExecution(req)
}
