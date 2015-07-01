package testing

import (
	"fmt"
	"log"

	"sync"

	"github.com/awslabs/aws-sdk-go/service/swf"
	"github.com/sclasen/swfsm/activity"
	"github.com/sclasen/swfsm/enums/swf"
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
	DecisionTask *swf.PollForDecisionTaskOutput
	State        *fsm.SerializedState
	Decisions    []*swf.Decision
}

type StateData struct {
	State string
	Data  interface{}
}

func StubFSM(domain string, client fsm.SWFOps) *fsm.FSM {
	f := &fsm.FSM{
		SWF:        client,
		DataType:   make(map[string]interface{}),
		Domain:     domain,
		Name:       StubWorkflow,
		Serializer: fsm.JSONStateSerializer{},
		TaskList:   *StubTaskList.Name,
	}

	f.AddInitialState(&fsm.FSMState{Name: "Initial", Decider: StubState()})
	return f
}

func StubState() fsm.Decider {
	return func(ctx *fsm.FSMContext, h *swf.HistoryEvent, data interface{}) fsm.Outcome {
		log.Printf("at=stub-event event=%+v", PrettyHistoryEvent(h))
		return ctx.Stay(data, ctx.EmptyDecisions())
	}
}

func ShortStubFSM(domain string, client fsm.SWFOps) *fsm.FSM {
	f := &fsm.FSM{
		SWF:        client,
		DataType:   make(map[string]interface{}),
		Domain:     domain,
		Name:       ShortStubWorkflow,
		Serializer: fsm.JSONStateSerializer{},
		TaskList:   *StubTaskList.Name,
	}

	f.AddInitialState(&fsm.FSMState{Name: "Initial", Decider: ShortStubState()})
	return f
}

func ShortStubState() fsm.Decider {
	return func(ctx *fsm.FSMContext, h *swf.HistoryEvent, data interface{}) fsm.Outcome {
		log.Printf("at=short-stub-event event=%+v", PrettyHistoryEvent(h))
		return ctx.CompleteWorkflow(data)
	}
}

//intercept any attempts to start a workflow and launch the stub workflow instead.
func TestDecisionInterceptor(testID string, stubbedWorkflows, stubbedShortWorkflows []string, interceptor fsm.DecisionInterceptor) fsm.DecisionInterceptor {
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
		AfterDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *fsm.FSMContext, outcome *fsm.Outcome) {
			for _, d := range outcome.Decisions {
				switch *d.DecisionType {
				case enums.DecisionTypeStartChildWorkflowExecution:
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
				case enums.DecisionTypeScheduleActivityTask:
					d.ScheduleActivityTaskDecisionAttributes.TaskList = &swf.TaskList{Name: S(*d.ScheduleActivityTaskDecisionAttributes.TaskList.Name + testID)}
				}
			}

			if interceptor != nil {
				interceptor.AfterDecision(decision, ctx, outcome)
			}
		},
	}
}

func NoOpActivityInterceptor() activity.ActivityInterceptor {
	return &activity.FuncInterceptor{}
}

// interceptor that fails the activity once per activity and returns to actual result subsequent time
// used to test error handling and retries of activities in fsms
func TestFailOnceActivityInterceptor() activity.ActivityInterceptor {
	mutex := sync.Mutex{}
	tried := map[string]bool{}
	return &activity.FuncInterceptor{
		AfterTaskFn: func(t *swf.PollForActivityTaskOutput, result interface{}, err error) (interface{}, error) {
			mutex.Lock()
			defer mutex.Unlock()

			if err != nil || tried[*t.ActivityID] {
				log.Printf("interceptor.test.fail-once at=passthrough activity-id=%q", *t.ActivityID)
				return result, err
			}

			tried[*t.ActivityID] = true
			msg := fmt.Sprintf("interceptor.test.fail-once at=fail activity-id=%q", *t.ActivityID)
			log.Println(msg)
			return nil, fmt.Errorf(msg)
		},
	}
}

func TestReplicator(decisionOutcomes chan DecisionOutcome) fsm.ReplicationHandler {
	return func(ctx *fsm.FSMContext, task *swf.PollForDecisionTaskOutput, outcome *swf.RespondDecisionTaskCompletedInput, state *fsm.SerializedState) error {
		decisionOutcomes <- DecisionOutcome{State: state, DecisionTask: task, Decisions: outcome.Decisions}
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

func (s *StubSWFClient) StartWorkflowExecution(req *swf.StartWorkflowExecutionInput) (resp *swf.StartWorkflowExecutionOutput, err error) {
	if _, ok := s.stubbedWorkflows[*req.WorkflowType.Name]; ok {
		req.WorkflowType.Name = S(StubWorkflow)
		req.WorkflowType.Version = S(StubVersion)
		req.ExecutionStartToCloseTimeout = S("360")
		req.TaskList = StubTaskList
	}
	return s.ClientSWFOps.StartWorkflowExecution(req)
}
