package fsm

import (
	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/sclasen/swfsm/enums/swf"
	. "github.com/sclasen/swfsm/sugar"
	"strconv"
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

//ManagedContinuations is an interceptr that will handle most of the mechanics of autmoatically continuing workflows.
func ManagedContinuations(historySize int, workflowAgeInSec int, timerRetrySeconds int) DecisionInterceptor {
	return &FuncInterceptor{
		AfterDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome) {
			for _, d := range outcome.Decisions {
				if *d.DecisionType == enums.DecisionTypeCompleteWorkflowExecution ||
				*d.DecisionType == enums.DecisionTypeCancelWorkflowExecution ||
				*d.DecisionType == enums.DecisionTypeFailWorkflowExecution {
					logf(ctx, "fn=managed-continuations at=terminating-decision")
					return //we have a terminating event, dont continue
				}
			}

			//if events contains eventID 1, add start timer decision.
			if *decision.Events[0].EventID == int64(1) {
				logf(ctx, "fn=managed-continuations at=workflow-start")
				outcome.Decisions = append(outcome.Decisions, &swf.Decision{
					DecisionType: S(enums.DecisionTypeStartTimer),
					StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
						TimerID: S(ContinueTimer),
						StartToFireTimeout: S(strconv.Itoa(workflowAgeInSec)),
					},
				})
			}

			//was the ContinueTimer fired?
			continueTimerFired := false
			for _, h := range decision.Events {
				if *h.EventType == enums.EventTypeTimerFired {
					if *h.TimerFiredEventAttributes.TimerID == ContinueTimer{
						continueTimerFired = true
					}
				}
			}

			historySizeExceeded := int64(historySize) > *decision.Events[len(decision.Events)-1].EventID


			//if we pass history sizex or if we see ContinuteTimer fired
			if continueTimerFired || historySizeExceeded {
				logf(ctx, "fn=managed-continuations at=attempt-continue continue-timer=%t history-size=%t", continueTimerFired, historySizeExceeded)
			//if we can safely continue
				if len(outcome.Decisions) == 0 &&
					len(ctx.Correlator().Activities) == 0 &&
					len(ctx.Correlator().SignalAttempts)  == 0 {
					logf(ctx, "fn=managed-continuations at=able-to-continue action=add-continue-decision")
					outcome.Decisions = append(outcome.Decisions, ctx.ContinueWorkflowDecision(ctx.State, ctx.stateData)) //stateData safe?
				} else {
					//re-start the timer for timerRetrySecs
					logf(ctx, "fn=managed-continuations at=unable-to-continue action=start-continue-timer-retry")
					outcome.Decisions = append(outcome.Decisions, &swf.Decision{
						DecisionType: S(enums.DecisionTypeStartTimer),
						StartTimerDecisionAttributes: &swf.StartTimerDecisionAttributes{
							TimerID: S(ContinueTimer),
							StartToFireTimeout: S(strconv.Itoa(timerRetrySeconds)),
						},
					})
				}
			}
		},
	}
}
