package fsm

import (
	"strconv"

	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go/service/swf"
	. "github.com/sclasen/swfsm/sugar"
)

//DecisionInterceptor allows manipulation of the decision task and the outcome at key points in the task lifecycle.
type DecisionInterceptor interface {
	BeforeTask(decision *swf.PollForDecisionTaskOutput)
	BeforeDecision(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome)
	AfterDecision(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome)
}

//FuncInterceptor is a DecisionInterceptor that you can set handler funcs on. if any are unset, they are no-ops.
type FuncInterceptor struct {
	BeforeTaskFn     func(decision *swf.PollForDecisionTaskOutput)
	BeforeDecisionFn func(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome)
	AfterDecisionFn  func(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome)
}

//BeforeTask runs the BeforeTaskFn if not nil
func (i *FuncInterceptor) BeforeTask(decision *swf.PollForDecisionTaskOutput) {
	if i.BeforeTaskFn != nil {
		i.BeforeTaskFn(decision)
	}
}

//BeforeDecision runs the BeforeDecisionFn if not nil
func (i *FuncInterceptor) BeforeDecision(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome) {
	if i.BeforeDecisionFn != nil {
		i.BeforeDecisionFn(decision, ctx, outcome)
	}
}

//AfterDecision runs the AfterDecisionFn if not nil
func (i *FuncInterceptor) AfterDecision(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome) {
	if i.AfterDecisionFn != nil {
		i.AfterDecisionFn(decision, ctx, outcome)
	}
}

type ComposedDecisionInterceptor struct {
	interceptors []DecisionInterceptor
}

func NewComposedDecisionInterceptor(interceptors ...DecisionInterceptor) DecisionInterceptor {
	c := &ComposedDecisionInterceptor{}
	for _, i := range interceptors {
		if i != nil {
			c.interceptors = append(c.interceptors, i)
		}
	}
	return c
}

func (c *ComposedDecisionInterceptor) BeforeTask(decision *swf.PollForDecisionTaskOutput) {
	for _, i := range c.interceptors {
		i.BeforeTask(decision)
	}
}

func (c *ComposedDecisionInterceptor) BeforeDecision(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome) {
	for _, i := range c.interceptors {
		i.BeforeDecision(decision, ctx, outcome)
	}
}

func (c *ComposedDecisionInterceptor) AfterDecision(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome) {
	for _, i := range c.interceptors {
		i.AfterDecision(decision, ctx, outcome)
	}
}

//ManagedContinuations is an interceptor that will handle most of the mechanics of automatically continuing workflows.
//
//For workflows without persistent, heartbeating activities, it should do everything.
//
//ContinueSignal: How to continue fsms with persistent activities.
//How it works
//FSM in steady+activity, listens for ContinueTimer.
//OnTimer, cancel activity, transition to continuing.
//In continuing, OnActivityCanceled send ContinueSignal.
//Interceptor handles ContinueSignal, if 0,0,0,0 Continue, else start ContinueTimer.
//In continuing, OnStarted re-starts the activity, transition back to steady.
func ManagedContinuations(historySize int, workflowAgeInSec int, timerRetrySeconds int) DecisionInterceptor {
	return ManagedContinuationsWithJitter(historySize, 0, workflowAgeInSec, 0, timerRetrySeconds)
}

//To avoid stampedes of workflows that are started at the same time being continued at the same time
//ManagedContinuationsWithJitter will schedule the initial continue randomly between
//workflowAgeInSec and workflowAgeInSec + maxAgeJitterInSec
//and will attempt to continue workflows with more than between
//historySize and historySize + maxSizeJitter events
func ManagedContinuationsWithJitter(historySize int, maxSizeJitter int, workflowAgeInSec int, maxAgeJitterInSec int, timerRetrySeconds int) DecisionInterceptor {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	//dont blow up on bad values
	if maxSizeJitter <= 0 {
		maxSizeJitter = 1
	}
	if maxAgeJitterInSec <= 0 {
		maxAgeJitterInSec = 1
	}
	return &FuncInterceptor{
		AfterDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome) {
			for _, d := range outcome.Decisions {
				if *d.DecisionType == swf.DecisionTypeCompleteWorkflowExecution ||
					*d.DecisionType == swf.DecisionTypeCancelWorkflowExecution ||
					*d.DecisionType == swf.DecisionTypeFailWorkflowExecution {
					logf(ctx, "fn=managed-continuations at=terminating-decision")
					return //we have a terminating event, dont continue
				}
			}

			//if prevStarted = 0 this is the first decision of the workflow, so start the continue timer.
			if *decision.PreviousStartedEventId == int64(0) {
				logf(ctx, "fn=managed-continuations at=workflow-start %d", *decision.PreviousStartedEventId)
				outcome.Decisions = append(outcome.Decisions, &swf.Decision{
					DecisionType: S(swf.DecisionTypeStartTimer),
					StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
						TimerId:            S(ContinueTimer),
						StartToFireTimeout: S(strconv.Itoa(workflowAgeInSec + rng.Intn(maxAgeJitterInSec))),
					},
				})
			}

			//was the ContinueTimer fired?
			continueTimerFired := false
			for _, h := range decision.Events {
				if *h.EventType == swf.EventTypeTimerFired {
					if *h.TimerFiredEventAttributes.TimerId == ContinueTimer {
						continueTimerFired = true
					}
				}
			}

			//was the ContinueSignal fired?
			continueSignalFired := false
			for _, h := range decision.Events {
				if *h.EventType == swf.EventTypeWorkflowExecutionSignaled {
					if *h.WorkflowExecutionSignaledEventAttributes.SignalName == ContinueSignal {
						continueSignalFired = true
					}
				}
			}

			historySizeExceeded := int64(historySize+rng.Intn(maxSizeJitter)) < *decision.Events[0].EventId

			//if we pass history size or if we see ContinuteTimer or ContinueSignal fired
			if continueTimerFired || continueSignalFired || historySizeExceeded {
				logf(ctx, "fn=managed-continuations at=attempt-continue continue-timer=%t continue-signal=%t history-size=%t", continueTimerFired, continueSignalFired, historySizeExceeded)
				//if we can safely continue
				decisions := len(outcome.Decisions)
				activities := len(ctx.Correlator().Activities)
				signals := len(ctx.Correlator().Signals)
				children := len(ctx.Correlator().Children)
				cancels := len(ctx.Correlator().Cancellations)
				if decisions == 0 && activities == 0 && signals == 0 && children == 0 && cancels == 0 {
					logf(ctx, "fn=managed-continuations at=able-to-continue action=add-continue-decision")
					outcome.Decisions = append(outcome.Decisions, ctx.ContinueWorkflowDecision(ctx.State, ctx.stateData)) //stateData safe?
				} else {
					//re-start the timer for timerRetrySecs
					logf(ctx, "fn=managed-continuations at=unable-to-continue decisions=%d activities=%d signals=%d children=%d cancels=%d action=start-continue-timer-retry", decisions, activities, signals, children, cancels)
					if continueTimerFired || !ctx.Correlator().TimerScheduled(ContinueTimer) {
						outcome.Decisions = append(outcome.Decisions, &swf.Decision{
							DecisionType: S(swf.DecisionTypeStartTimer),
							StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
								TimerId:            S(ContinueTimer),
								StartToFireTimeout: S(strconv.Itoa(timerRetrySeconds)),
							},
						})
					}
				}
			}
		},
	}
}
