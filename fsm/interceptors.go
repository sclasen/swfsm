package fsm

import (
	"strconv"

	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go/aws"
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

// DedupeWorkflowCompletes returns an interceptor that executes after a decision and removes
// any duplicate swf.DecisionTypeCompleteWorkflowExecution decisions from the outcome.
// Duplicates are removed from the beginning of the input list, so that
// the last complete decision is the one that remains in the list.
func DedupeWorkflowCompletes() DecisionInterceptor {
	return DedupeDecisions(swf.DecisionTypeCompleteWorkflowExecution)
}

// DedupeWorkflowCancellations returns an interceptor that executes after a decision and removes
// any duplicate swf.DecisionTypeCancelWorkflowExecution decisions from the outcome.
// Duplicates are removed from the beginning of the input list, so that
// the last cancel decision is the one that remains in the list.
func DedupeWorkflowCancellations() DecisionInterceptor {
	return DedupeDecisions(swf.DecisionTypeCancelWorkflowExecution)
}

// DedupeWorkflowFailures returns an interceptor that executes after a decision and removes
// any duplicate swf.DecisionTypeFailWorkflowExecution decisions from the outcome.
// Duplicates are removed from the beginning of the input list, so that
// the last failure decision is the one that remains in the list.
func DedupeWorkflowFailures() DecisionInterceptor {
	return DedupeDecisions(swf.DecisionTypeFailWorkflowExecution)
}

// DedupeWorkflowCloseDecisions returns an interceptor that executes after a decision and removes
// any duplicate workflow close decisions (cancel, complete, fail) from the outcome.
// Duplicates are removed from the beginning of the input list, so that
// the last failure decision is the one that remains in the list.
func DedupeWorkflowCloseDecisions() DecisionInterceptor {
	return NewComposedDecisionInterceptor(
		DedupeWorkflowCompletes(),
		DedupeWorkflowCancellations(),
		DedupeWorkflowFailures(),
	)
}

// DedupeDecisions returns an interceptor that executes after a decision and removes
// any duplicate decisions of the specified type from the outcome.
// Duplicates are removed from the beginning of the input list, so that
// the last decision of the specified type is the one that remains in the list.
//
// e.g.  An outcome with a list of decisions [a, a, b, a, c] where the type to dedupe was 'a' would
// result in an outcome with a list of decisions [b, a, c]
func DedupeDecisions(decisionType string) DecisionInterceptor {
	return &FuncInterceptor{
		AfterDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome) {
			in := outcome.Decisions
			out := []*swf.Decision{}
			specifiedTypeEncountered := false

			// iterate backwards so we can grab the last decision
			for i := len(in) - 1; i >= 0; i-- {
				currentDecision := in[i]
				// keep last instance of decisions of the specified type
				if *currentDecision.DecisionType == decisionType && !specifiedTypeEncountered {
					specifiedTypeEncountered = true
					// prepend
					out = append([]*swf.Decision{currentDecision}, out...)
				} else if *currentDecision.DecisionType != decisionType {
					// prepend
					out = append([]*swf.Decision{currentDecision}, out...)
				}
			}
			outcome.Decisions = out
		},
	}
}

// MoveWorkflowCloseDecisionsToEnd returns an interceptor that executes after a decision and moves
// any workflow close decisions (complete, fail, cancel) to the end of an outcome's decision list.
//
// Note: SWF responds with a 400 error if a workflow close decision is not the last decision
// in the list of decisions.
func MoveWorkflowCloseDecisionsToEnd() DecisionInterceptor {
	return NewComposedDecisionInterceptor(
		MoveDecisionsToEnd(swf.DecisionTypeFailWorkflowExecution),
		MoveDecisionsToEnd(swf.DecisionTypeCancelWorkflowExecution),
		MoveDecisionsToEnd(swf.DecisionTypeCompleteWorkflowExecution),
	)
}

// MoveDecisionsToEnd returns an interceptor that executes after a decision and moves
// any decisions of the specified type to the end of an outcome's decision list.
//
// e.g.  An outcome with a list of decisions [a, a, b, a, c] where the type to move was 'a' would
// result in an outcome with a list of decisions [b, c, a, a, a]
func MoveDecisionsToEnd(decisionType string) DecisionInterceptor {
	return &FuncInterceptor{
		AfterDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome) {
			in := outcome.Decisions
			out := []*swf.Decision{}
			decisionsToMove := []*swf.Decision{}

			for i, currentDecision := range in {
				if *currentDecision.DecisionType == decisionType {
					// don't append currentDecision because it's value changes on each iteration
					decisionsToMove = append(decisionsToMove, in[i])
				} else {
					out = append(out, in[i])
				}
			}
			out = append(out, decisionsToMove...)
			outcome.Decisions = out
		},
	}
}

// RemoveLowerPriorityDecisions returns an interceptor that executes after a decision and removes
// any lower priority decisions from an outcome if a higher priority decision exists.
// The decisionTypes passed to this function should be listed in highest to
// lowest priority order.
//
// e.g.  An outcome with a list of decisions [a, a, b, a, c] where the priority
// was a > b > c would return [a, a, a]
func RemoveLowerPriorityDecisions(prioritizedDecisionTypes ...string) DecisionInterceptor {
	return &FuncInterceptor{
		AfterDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome) {
			in := outcome.Decisions
			out := []*swf.Decision{}
			var indexOfHighestPriorityDecision *int

			// Find highest priority item that is in list
			for _, currentDecision := range in {
				if index := indexOfString(prioritizedDecisionTypes, *currentDecision.DecisionType); index != -1 {
					if indexOfHighestPriorityDecision == nil || index < *indexOfHighestPriorityDecision {
						indexOfHighestPriorityDecision = aws.Int(index)
					}
				}
			}
			// Leave lower priority items off the final decision list
			for _, currentDecision := range in {
				index := indexOfString(prioritizedDecisionTypes, *currentDecision.DecisionType)
				if index != -1 && indexOfHighestPriorityDecision != nil && index > *indexOfHighestPriorityDecision {
					continue
				}
				out = append(out, currentDecision)
			}
			outcome.Decisions = out
		},
	}
}

func indexOfString(stringSlice []string, testString string) int {
	index := -1
	for i, currentString := range stringSlice {
		if currentString == testString {
			index = i
			break
		}
	}
	return index
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

			eventCount := *decision.Events[0].EventId
			historySizeExceeded := int64(historySize+rng.Intn(maxSizeJitter)) < eventCount

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
					logf(ctx, "fn=managed-continuations at=able-to-continue action=add-continue-decision events=%d", eventCount)
					outcome.Decisions = append(outcome.Decisions, ctx.ContinueWorkflowDecision(ctx.State, ctx.stateData)) //stateData safe?
				} else {
					//re-start the timer for timerRetrySecs
					logf(ctx, "fn=managed-continuations at=unable-to-continue decisions=%d activities=%d signals=%d children=%d cancels=%d  events=%d action=start-continue-timer-retry", decisions, activities, signals, children, cancels, eventCount)
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

func StartCancelInterceptor() DecisionInterceptor {
	return &FuncInterceptor{
		AfterDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome) {
			outcome.Decisions = handleStartCancelTypes(outcome.Decisions, ctx)
		},
	}
}

type StartCancelPair struct {
	idField        string
	startDecision  string
	cancelDecision string
	startId        func(d *swf.Decision) *string
	cancelId       func(d *swf.Decision) *string
}

var startCancelPairs = []*StartCancelPair{
	&StartCancelPair{
		idField:        "workflow",
		startDecision:  swf.DecisionTypeStartChildWorkflowExecution,
		cancelDecision: swf.DecisionTypeRequestCancelExternalWorkflowExecution,
		startId:        func(d *swf.Decision) *string { return d.StartChildWorkflowExecutionDecisionAttributes.WorkflowId },
		cancelId: func(d *swf.Decision) *string {
			return d.RequestCancelExternalWorkflowExecutionDecisionAttributes.WorkflowId
		},
	},
	&StartCancelPair{
		idField:        "activity",
		startDecision:  swf.DecisionTypeScheduleActivityTask,
		cancelDecision: swf.DecisionTypeRequestCancelActivityTask,
		startId:        func(d *swf.Decision) *string { return d.ScheduleActivityTaskDecisionAttributes.ActivityId },
		cancelId: func(d *swf.Decision) *string {
			return d.RequestCancelActivityTaskDecisionAttributes.ActivityId
		},
	},
	&StartCancelPair{
		idField:        "timer",
		startDecision:  swf.DecisionTypeStartTimer,
		cancelDecision: swf.DecisionTypeCancelTimer,
		startId:        func(d *swf.Decision) *string { return d.StartTimerDecisionAttributes.TimerId },
		cancelId: func(d *swf.Decision) *string {
			return d.CancelTimerDecisionAttributes.TimerId
		},
	},
}

func handleStartCancelTypes(in []*swf.Decision, ctx *FSMContext) []*swf.Decision {
	for _, scp := range startCancelPairs {
		in = scp.removeStartBeforeCancel(in, ctx)
	}
	return in
}

func (s *StartCancelPair) removeStartBeforeCancel(in []*swf.Decision, ctx *FSMContext) []*swf.Decision {
	var out []*swf.Decision

	for _, decision := range in {
		switch *decision.DecisionType {
		case s.cancelDecision:
			cancelId := aws.StringValue(s.cancelId(decision))
			if cancelId == "" {
				continue
			}
			i := s.decisionsContainStartCancel(out, &cancelId)
			if i >= 0 {

				startDecision := out[i]
				startId := aws.StringValue(s.startId(startDecision))
				if startId == "" {
					continue
				}
				out = append(out[:i], out[i+1:]...)
				logf(ctx, "fn=remove-start-before-cancel at=start-cancel-detected status=removing-start-cancel workflow=%s run=%s start-%s=%s cancel-%s=%s", aws.StringValue(ctx.WorkflowId), aws.StringValue(ctx.RunId), s.idField, startId, s.idField, cancelId)
			} else {
				out = append(out, decision)
			}
		default:
			out = append(out, decision)
		}
	}

	return out
}

// This ensures the pairs are of the same workflow/activity/timer ID
func (s *StartCancelPair) decisionsContainStartCancel(in []*swf.Decision, cancelId *string) int {
	for i, d := range in {
		if *d.DecisionType == s.startDecision && *s.startId(d) == *cancelId {
			return i
		}
	}
	return -1
}

func CloseDecisionTypes() []string {
	return []string{
		swf.DecisionTypeCompleteWorkflowExecution,
		swf.DecisionTypeCancelWorkflowExecution,
		swf.DecisionTypeFailWorkflowExecution,

		// swf.DecisionTypeContinueAsNewWorkflowExecution is technically a close type,
		// but swfsm currently doesn't really treat it as one and there's
		// a lot of special logic, so excluding it from here for now
	}
}

func CloseDecisionIncompatableDecisionTypes() []string {
	return []string{
		// TODO: flush out this list with other incompatible types
		// TODO: see https://console.aws.amazon.com/support/home?#/case/?caseId=5223303261&displayId=5223303261&language=en
		swf.DecisionTypeScheduleActivityTask,
		swf.DecisionTypeStartTimer,
	}
}

// CloseWorkflowRemoveIncompatibleDecisionInterceptor checks for
// incompatible decisions with a Complete workflow decision, and if
// found removes it from the outcome.
func CloseWorkflowRemoveIncompatibleDecisionInterceptor() DecisionInterceptor {
	return &FuncInterceptor{
		AfterDecisionFn: func(decision *swf.PollForDecisionTaskOutput, ctx *FSMContext, outcome *Outcome) {
			closingDecisionFound := false
			// Search for close decisions
			for _, d := range outcome.Decisions {
				if stringsContain(CloseDecisionTypes(), *d.DecisionType) {
					closingDecisionFound = true
				}
			}

			// If we have a complete decision, search for banned decisions and drop them.
			if closingDecisionFound {
				var decisions []*swf.Decision
				for _, d := range outcome.Decisions {
					if stringsContain(CloseDecisionIncompatableDecisionTypes(), *d.DecisionType) {
						logf(ctx, "fn=CloseWorkflowRemoveIncompatibleDecisionInterceptor at=remove decision-type=%s", *d.DecisionType)
						continue
					}
					decisions = append(decisions, d)
				}
				outcome.Decisions = decisions
			}
		},
	}
}
