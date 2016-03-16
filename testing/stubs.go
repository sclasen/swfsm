package testing

import (
	"fmt"

	"sync"

	"strings"

	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/sclasen/swfsm/activity"
	"github.com/sclasen/swfsm/fsm"
	. "github.com/sclasen/swfsm/log"
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

type NoData struct{}

func StubFSM(domain string, client fsm.SWFOps) *fsm.FSM {
	f := &fsm.FSM{
		SWF:        client,
		DataType:   NoData{},
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
		Log.Printf("at=stub-event event=%+v", PrettyHistoryEvent(h))
		return ctx.Stay(data, ctx.EmptyDecisions())
	}
}

func ShortStubFSM(domain string, client fsm.SWFOps) *fsm.FSM {
	f := &fsm.FSM{
		SWF:        client,
		DataType:   NoData{},
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
		Log.Printf("at=short-stub-event event=%+v", PrettyHistoryEvent(h))
		return ctx.CompleteWorkflow(data)
	}
}

//intercept any attempts to start a workflow and launch the stub workflow instead.
func TestDecisionInterceptor(testId string, stubbedWorkflows, stubbedShortWorkflows []string) fsm.DecisionInterceptor {
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
					d.ScheduleActivityTaskDecisionAttributes.TaskList = &swf.TaskList{Name: S(*d.ScheduleActivityTaskDecisionAttributes.TaskList.Name + testId)}
				case swf.DecisionTypeContinueAsNewWorkflowExecution:
					d.ContinueAsNewWorkflowExecutionDecisionAttributes.TaskList = &swf.TaskList{Name: S(testId)}
				}
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

			if err != nil || tried[*t.ActivityId] {
				Log.Printf("interceptor.test.fail-once at=passthrough activity-id=%q", *t.ActivityId)
				return result, err
			}

			tried[*t.ActivityId] = true
			msg := fmt.Sprintf("interceptor.test.fail-once at=fail activity-id=%q", *t.ActivityId)
			Log.Println(msg)
			return nil, fmt.Errorf(msg)
		},
	}
}

/*
TestThrotteSignalsOnceInterceptor and all the TestThrottleInterceptor work as follows

On the initial decision task, the after decision interceptor rewrites signals/cancels/children/timers such that the subsequent decision task will have a
corresponding failure. By using a non-existent workflow-id or other similar mechanism.

On the subsequent decision task, the before decision task rewrites the error cause from the relevant "NotFound" to the relevant "Throttling" error.

The FSM handles these errors presumptiviely by rescheduling the signal/timer/cancel/child

On the after decision task the interceptor clears out the state set in the initial decision, used by the subsequent before interceptor, and the signal/timer/cancel/child succeeds/
*/

func TestThrotteSignalsOnceInterceptor() fsm.DecisionInterceptor {
	throttledSignals := make(map[string]*fsm.SignalInfo)
	return &fsm.FuncInterceptor{
		BeforeDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *fsm.FSMContext, outcome *fsm.Outcome) {
			for _, h := range decision.Events {
				if *h.EventId <= *decision.PreviousStartedEventId {
					break
				}
				if *h.EventType == swf.EventTypeSignalExternalWorkflowExecutionFailed {
					if info, found := throttledSignals[*h.SignalExternalWorkflowExecutionFailedEventAttributes.WorkflowId]; found {
						h.SignalExternalWorkflowExecutionFailedEventAttributes.WorkflowId = &info.WorkflowId
						//rewrite workflow not found to throttling.
						h.SignalExternalWorkflowExecutionFailedEventAttributes.Cause = S(swf.SignalExternalWorkflowExecutionFailedCauseSignalExternalWorkflowExecutionRateExceeded)
					}
				}
			}

		},
		AfterDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *fsm.FSMContext, outcome *fsm.Outcome) {
			for _, d := range outcome.Decisions {
				switch *d.DecisionType {
				case swf.DecisionTypeSignalExternalWorkflowExecution:
					signalWF := *d.SignalExternalWorkflowExecutionDecisionAttributes.WorkflowId
					signalName := *d.SignalExternalWorkflowExecutionDecisionAttributes.SignalName
					throttle := fmt.Sprintf("fail-on-purpose-%s-%s", signalWF, signalName)
					if _, found := throttledSignals[throttle]; !found {
						//if we havent failed the signal yet, add it and swap the workflow id so it will fail
						throttledSignals[throttle] = &fsm.SignalInfo{WorkflowId: signalWF, SignalName: signalName}
						d.SignalExternalWorkflowExecutionDecisionAttributes.WorkflowId = S(throttle)
					} else {
						//we are retrying after the forced failure, remove it
						delete(throttledSignals, throttle)
					}
				}

			}
		},
	}
}

func TestThrotteCancelsOnceInterceptor() fsm.DecisionInterceptor {
	throttledCancels := make(map[string]*fsm.CancellationInfo)
	return &fsm.FuncInterceptor{
		BeforeDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *fsm.FSMContext, outcome *fsm.Outcome) {
			for _, h := range decision.Events {
				if *h.EventId <= *decision.PreviousStartedEventId {
					break
				}
				if *h.EventType == swf.EventTypeRequestCancelExternalWorkflowExecutionFailed {
					if info, found := throttledCancels[*h.RequestCancelExternalWorkflowExecutionFailedEventAttributes.WorkflowId]; found {
						h.RequestCancelExternalWorkflowExecutionFailedEventAttributes.WorkflowId = &info.WorkflowId
						//rewrite workflow not found to throttling.
						h.RequestCancelExternalWorkflowExecutionFailedEventAttributes.Cause = S(swf.RequestCancelExternalWorkflowExecutionFailedCauseRequestCancelExternalWorkflowExecutionRateExceeded)
					}
				}
			}

		},
		AfterDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *fsm.FSMContext, outcome *fsm.Outcome) {
			for _, d := range outcome.Decisions {
				switch *d.DecisionType {
				case swf.DecisionTypeRequestCancelExternalWorkflowExecution:
					cancelWF := *d.RequestCancelExternalWorkflowExecutionDecisionAttributes.WorkflowId
					throttle := fmt.Sprintf("fail-on-purpose-%s", cancelWF)
					if _, found := throttledCancels[throttle]; !found {
						//if we havent failed the signal yet, add it and swap the workflow id so it will fail
						throttledCancels[throttle] = &fsm.CancellationInfo{WorkflowId: cancelWF}
						d.RequestCancelExternalWorkflowExecutionDecisionAttributes.WorkflowId = S(throttle)
					} else {
						//we are retrying after the forced failure, remove it
						delete(throttledCancels, throttle)
					}
				}

			}
		},
	}
}

func TestThrotteChildrenOnceInterceptor() fsm.DecisionInterceptor {
	throttledChildren := make(map[string]*fsm.ChildInfo)
	return &fsm.FuncInterceptor{
		BeforeDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *fsm.FSMContext, outcome *fsm.Outcome) {
			for _, h := range decision.Events {
				if *h.EventId <= *decision.PreviousStartedEventId {
					break
				}
				if *h.EventType == swf.EventTypeStartChildWorkflowExecutionFailed {
					if info, found := throttledChildren[*h.StartChildWorkflowExecutionFailedEventAttributes.WorkflowId]; found {
						h.StartChildWorkflowExecutionFailedEventAttributes.WorkflowId = &info.WorkflowId
						//rewrite workflow not found to throttling.
						h.StartChildWorkflowExecutionFailedEventAttributes.Cause = S(swf.StartChildWorkflowExecutionFailedCauseChildCreationRateExceeded)
					}
				}
			}

		},
		AfterDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *fsm.FSMContext, outcome *fsm.Outcome) {
			for _, d := range outcome.Decisions {
				switch *d.DecisionType {
				case swf.DecisionTypeStartChildWorkflowExecution:
					childWF := *d.StartChildWorkflowExecutionDecisionAttributes.WorkflowId
					throttle := fmt.Sprintf("fail-on-purpose-%s", childWF)
					if _, found := throttledChildren[throttle]; !found {
						//if we havent failed the child yet, add it and swap the workflow id so it will fail
						throttledChildren[throttle] = &fsm.ChildInfo{WorkflowId: childWF}
						d.StartChildWorkflowExecutionDecisionAttributes.WorkflowId = S(throttle)
						d.StartChildWorkflowExecutionDecisionAttributes.WorkflowType.Name = S(throttle)
					} else {
						//we are retrying after the forced failure, remove it
						delete(throttledChildren, throttle)
					}
				}

			}
		},
	}
}

//This one is a bit weird. OnStart, we need to schedule a bunch of timers that we can use to
//cause timer id in use failures, that we rewrite to throttles
func TestThrotteTimersOnceInterceptor(numTimers int) fsm.DecisionInterceptor {
	timerNames := []string{}
	origToReplace := make(map[string]string)
	replaceToOrig := make(map[string]string)
	return &fsm.FuncInterceptor{
		BeforeDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *fsm.FSMContext, outcome *fsm.Outcome) {
			for _, h := range decision.Events {
				if *h.EventId <= *decision.PreviousStartedEventId {
					break
				}
				if *h.EventType == swf.EventTypeStartTimerFailed {
					if info, found := replaceToOrig[*h.StartTimerFailedEventAttributes.TimerId]; found {
						h.StartTimerFailedEventAttributes.TimerId = S(info)
						//rewrite timer in use to throttling.
						h.StartTimerFailedEventAttributes.Cause = S(swf.StartTimerFailedCauseTimerCreationRateExceeded)
					}
				}
			}

		},
		AfterDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *fsm.FSMContext, outcome *fsm.Outcome) {
			for _, h := range decision.Events {
				if *h.EventId <= *decision.PreviousStartedEventId {
					break
				}
				//when the workflow starts schedule numTimers timers to use for causing fake throttles
				if *h.EventType == swf.EventTypeWorkflowExecutionStarted {
					for i := 0; i < numTimers; i++ {
						timer := fmt.Sprintf("throttle-test-%d", i)
						timerNames = append(timerNames, timer)
						decision := &swf.Decision{
							DecisionType: S(swf.DecisionTypeStartTimer),
							StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
								TimerId:            S(timer),
								StartToFireTimeout: S("3600"),
							},
						}
						outcome.Decisions = append(outcome.Decisions, decision)
					}
				}
			}
			for _, d := range outcome.Decisions {
				switch *d.DecisionType {
				case swf.DecisionTypeStartTimer:
					timerId := *d.StartTimerDecisionAttributes.TimerId
					if timerId == fsm.ContinueTimer || strings.HasPrefix(timerId, "throttle-test") {
						continue
					}

					throttle := timerNames[0]
					timerNames = timerNames[1:]
					if _, found := origToReplace[timerId]; !found {
						//if we havent failed the signal yet, add it and swap the workflow id so it wont fail
						origToReplace[timerId] = throttle
						replaceToOrig[throttle] = timerId
						d.StartTimerDecisionAttributes.TimerId = S(throttle)
					} else {
						//we are retrying after the forced failure, remove it
						throttle = origToReplace[timerId]
						delete(origToReplace, timerId)
						delete(replaceToOrig, throttle)
						timerNames = append(timerNames, throttle)
					}
				}

			}
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
