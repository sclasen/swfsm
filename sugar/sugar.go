package sugar

import (
	"bytes"
	"fmt"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/swf"
)

const (
	ErrorTypeUnknownResourceFault                 = "com.amazonaws.swf.base.model#UnknownResourceFault"
	ErrorTypeWorkflowExecutionAlreadyStartedFault = "com.amazonaws.swf.base.model#WorkflowExecutionAlreadyStartedFault"
	ErrorTypeStreamNotFound                       = "ResourceNotFoundException"
)

var eventTypes = map[string]func(swf.HistoryEvent) interface{}{
	swf.EventTypeWorkflowExecutionStarted:                 func(h swf.HistoryEvent) interface{} { return h.WorkflowExecutionStartedEventAttributes },
	swf.EventTypeWorkflowExecutionCancelRequested:         func(h swf.HistoryEvent) interface{} { return h.WorkflowExecutionCancelRequestedEventAttributes },
	swf.EventTypeWorkflowExecutionCompleted:               func(h swf.HistoryEvent) interface{} { return h.WorkflowExecutionCompletedEventAttributes },
	swf.EventTypeCompleteWorkflowExecutionFailed:          func(h swf.HistoryEvent) interface{} { return h.CompleteWorkflowExecutionFailedEventAttributes },
	swf.EventTypeWorkflowExecutionFailed:                  func(h swf.HistoryEvent) interface{} { return h.WorkflowExecutionFailedEventAttributes },
	swf.EventTypeFailWorkflowExecutionFailed:              func(h swf.HistoryEvent) interface{} { return h.FailWorkflowExecutionFailedEventAttributes },
	swf.EventTypeWorkflowExecutionTimedOut:                func(h swf.HistoryEvent) interface{} { return h.WorkflowExecutionTimedOutEventAttributes },
	swf.EventTypeWorkflowExecutionCanceled:                func(h swf.HistoryEvent) interface{} { return h.WorkflowExecutionCanceledEventAttributes },
	swf.EventTypeCancelWorkflowExecutionFailed:            func(h swf.HistoryEvent) interface{} { return h.CancelWorkflowExecutionFailedEventAttributes },
	swf.EventTypeWorkflowExecutionContinuedAsNew:          func(h swf.HistoryEvent) interface{} { return h.WorkflowExecutionContinuedAsNewEventAttributes },
	swf.EventTypeContinueAsNewWorkflowExecutionFailed:     func(h swf.HistoryEvent) interface{} { return h.ContinueAsNewWorkflowExecutionFailedEventAttributes },
	swf.EventTypeWorkflowExecutionTerminated:              func(h swf.HistoryEvent) interface{} { return h.WorkflowExecutionTerminatedEventAttributes },
	swf.EventTypeDecisionTaskScheduled:                    func(h swf.HistoryEvent) interface{} { return h.DecisionTaskScheduledEventAttributes },
	swf.EventTypeDecisionTaskStarted:                      func(h swf.HistoryEvent) interface{} { return h.DecisionTaskStartedEventAttributes },
	swf.EventTypeDecisionTaskCompleted:                    func(h swf.HistoryEvent) interface{} { return h.DecisionTaskCompletedEventAttributes },
	swf.EventTypeDecisionTaskTimedOut:                     func(h swf.HistoryEvent) interface{} { return h.DecisionTaskTimedOutEventAttributes },
	swf.EventTypeActivityTaskScheduled:                    func(h swf.HistoryEvent) interface{} { return h.ActivityTaskScheduledEventAttributes },
	swf.EventTypeScheduleActivityTaskFailed:               func(h swf.HistoryEvent) interface{} { return h.ScheduleActivityTaskFailedEventAttributes },
	swf.EventTypeActivityTaskStarted:                      func(h swf.HistoryEvent) interface{} { return h.ActivityTaskStartedEventAttributes },
	swf.EventTypeActivityTaskCompleted:                    func(h swf.HistoryEvent) interface{} { return h.ActivityTaskCompletedEventAttributes },
	swf.EventTypeActivityTaskFailed:                       func(h swf.HistoryEvent) interface{} { return h.ActivityTaskFailedEventAttributes },
	swf.EventTypeActivityTaskTimedOut:                     func(h swf.HistoryEvent) interface{} { return h.ActivityTaskTimedOutEventAttributes },
	swf.EventTypeActivityTaskCanceled:                     func(h swf.HistoryEvent) interface{} { return h.ActivityTaskCanceledEventAttributes },
	swf.EventTypeActivityTaskCancelRequested:              func(h swf.HistoryEvent) interface{} { return h.ActivityTaskCancelRequestedEventAttributes },
	swf.EventTypeRequestCancelActivityTaskFailed:          func(h swf.HistoryEvent) interface{} { return h.RequestCancelActivityTaskFailedEventAttributes },
	swf.EventTypeWorkflowExecutionSignaled:                func(h swf.HistoryEvent) interface{} { return h.WorkflowExecutionSignaledEventAttributes },
	swf.EventTypeMarkerRecorded:                           func(h swf.HistoryEvent) interface{} { return h.MarkerRecordedEventAttributes },
	swf.EventTypeRecordMarkerFailed:                       func(h swf.HistoryEvent) interface{} { return h.RecordMarkerFailedEventAttributes },
	swf.EventTypeTimerStarted:                             func(h swf.HistoryEvent) interface{} { return h.TimerStartedEventAttributes },
	swf.EventTypeStartTimerFailed:                         func(h swf.HistoryEvent) interface{} { return h.StartTimerFailedEventAttributes },
	swf.EventTypeTimerFired:                               func(h swf.HistoryEvent) interface{} { return h.TimerFiredEventAttributes },
	swf.EventTypeTimerCanceled:                            func(h swf.HistoryEvent) interface{} { return h.TimerCanceledEventAttributes },
	swf.EventTypeCancelTimerFailed:                        func(h swf.HistoryEvent) interface{} { return h.CancelTimerFailedEventAttributes },
	swf.EventTypeStartChildWorkflowExecutionInitiated:     func(h swf.HistoryEvent) interface{} { return h.StartChildWorkflowExecutionInitiatedEventAttributes },
	swf.EventTypeStartChildWorkflowExecutionFailed:        func(h swf.HistoryEvent) interface{} { return h.StartChildWorkflowExecutionFailedEventAttributes },
	swf.EventTypeChildWorkflowExecutionStarted:            func(h swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionStartedEventAttributes },
	swf.EventTypeChildWorkflowExecutionCompleted:          func(h swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionCompletedEventAttributes },
	swf.EventTypeChildWorkflowExecutionFailed:             func(h swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionFailedEventAttributes },
	swf.EventTypeChildWorkflowExecutionTimedOut:           func(h swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionTimedOutEventAttributes },
	swf.EventTypeChildWorkflowExecutionCanceled:           func(h swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionCanceledEventAttributes },
	swf.EventTypeChildWorkflowExecutionTerminated:         func(h swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionTerminatedEventAttributes },
	swf.EventTypeSignalExternalWorkflowExecutionInitiated: func(h swf.HistoryEvent) interface{} { return h.SignalExternalWorkflowExecutionInitiatedEventAttributes },
	swf.EventTypeSignalExternalWorkflowExecutionFailed:    func(h swf.HistoryEvent) interface{} { return h.SignalExternalWorkflowExecutionFailedEventAttributes },
	swf.EventTypeExternalWorkflowExecutionSignaled:        func(h swf.HistoryEvent) interface{} { return h.ExternalWorkflowExecutionSignaledEventAttributes },
	swf.EventTypeRequestCancelExternalWorkflowExecutionInitiated: func(h swf.HistoryEvent) interface{} {
		return h.RequestCancelExternalWorkflowExecutionInitiatedEventAttributes
	},
	swf.EventTypeRequestCancelExternalWorkflowExecutionFailed: func(h swf.HistoryEvent) interface{} {
		return h.RequestCancelExternalWorkflowExecutionFailedEventAttributes
	},
	swf.EventTypeExternalWorkflowExecutionCancelRequested: func(h swf.HistoryEvent) interface{} { return h.ExternalWorkflowExecutionCancelRequestedEventAttributes },
}

func PrettyHistoryEvent(h swf.HistoryEvent) string {
	var buffer bytes.Buffer
	buffer.WriteString("HistoryEvent{ ")
	buffer.WriteString(fmt.Sprintf("EventId: %d,", *h.EventID))
	buffer.WriteString(fmt.Sprintf("EventTimestamp: %s, ", *h.EventTimestamp))
	buffer.WriteString(fmt.Sprintf("EventType:, %s", *h.EventType))
	buffer.WriteString(fmt.Sprintf("%+v", eventTypes[*h.EventType](h)))
	buffer.WriteString(" }")
	return buffer.String()
}

// EventTypes returns a slice containing the valid values of the HistoryEvent.EventType field.
func SWFHistoryEventTypes() []string {
	es := make([]string, 0, len(eventTypes))
	for k := range eventTypes {
		es = append(es, k)
	}
	return es
}

var decisionTypes = map[string]func(swf.Decision) interface{}{
	swf.DecisionTypeScheduleActivityTask:                   func(d swf.Decision) interface{} { return d.ScheduleActivityTaskDecisionAttributes },
	swf.DecisionTypeRequestCancelActivityTask:              func(d swf.Decision) interface{} { return d.RequestCancelActivityTaskDecisionAttributes },
	swf.DecisionTypeCompleteWorkflowExecution:              func(d swf.Decision) interface{} { return d.CompleteWorkflowExecutionDecisionAttributes },
	swf.DecisionTypeFailWorkflowExecution:                  func(d swf.Decision) interface{} { return d.FailWorkflowExecutionDecisionAttributes },
	swf.DecisionTypeCancelWorkflowExecution:                func(d swf.Decision) interface{} { return d.CancelWorkflowExecutionDecisionAttributes },
	swf.DecisionTypeContinueAsNewWorkflowExecution:         func(d swf.Decision) interface{} { return d.ContinueAsNewWorkflowExecutionDecisionAttributes },
	swf.DecisionTypeRecordMarker:                           func(d swf.Decision) interface{} { return d.RecordMarkerDecisionAttributes },
	swf.DecisionTypeStartTimer:                             func(d swf.Decision) interface{} { return d.StartTimerDecisionAttributes },
	swf.DecisionTypeCancelTimer:                            func(d swf.Decision) interface{} { return d.CancelTimerDecisionAttributes },
	swf.DecisionTypeSignalExternalWorkflowExecution:        func(d swf.Decision) interface{} { return d.SignalExternalWorkflowExecutionDecisionAttributes },
	swf.DecisionTypeRequestCancelExternalWorkflowExecution: func(d swf.Decision) interface{} { return d.RequestCancelExternalWorkflowExecutionDecisionAttributes },
	swf.DecisionTypeStartChildWorkflowExecution:            func(d swf.Decision) interface{} { return d.StartChildWorkflowExecutionDecisionAttributes },
}

func PrettyDecision(d swf.Decision) string {
	var buffer bytes.Buffer
	buffer.WriteString("Decision{ ")
	buffer.WriteString(fmt.Sprintf("DecisionType: %s,", *d.DecisionType))
	buffer.WriteString(fmt.Sprintf("%+v", decisionTypes[*d.DecisionType](d)))
	buffer.WriteString(" }")
	return buffer.String()
}

// DecisionTypes returns a slice containing the valid values of the Decision.DecisionType field.
func SWFDecisionTypes() []string {
	ds := make([]string, 0, len(decisionTypes))
	for k := range eventTypes {
		ds = append(ds, k)
	}
	return ds
}

func L(l int64) aws.LongValue {
	return aws.Long(l)
}

func I(i int) aws.LongValue {
	return aws.Long(int64(i))
}

func S(s string) aws.StringValue {
	return aws.String(s)
}
