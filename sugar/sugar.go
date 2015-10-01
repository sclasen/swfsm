package sugar

import (
	"bytes"
	"fmt"

	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/swf"
)

//error code constants
const (
	ErrorTypeUnknownResourceFault                 = "UnknownResourceFault"
	ErrorTypeWorkflowExecutionAlreadyStartedFault = "WorkflowExecutionAlreadyStartedFault"
	ErrorTypeDomainAlreadyExistsFault             = "DomainAlreadyExistsFault"
	ErrorTypeAlreadyExistsFault                   = "TypeAlreadyExistsFault"
	ErrorTypeStreamNotFound                       = "ResourceNotFoundException"
	ErrorTypeStreamAlreadyExists                  = "ResourceInUseException"
)

var eventTypes = map[string]func(*swf.HistoryEvent) interface{}{
	swf.EventTypeWorkflowExecutionStarted:             func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionStartedEventAttributes },
	swf.EventTypeWorkflowExecutionCancelRequested:     func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionCancelRequestedEventAttributes },
	swf.EventTypeWorkflowExecutionCompleted:           func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionCompletedEventAttributes },
	swf.EventTypeCompleteWorkflowExecutionFailed:      func(h *swf.HistoryEvent) interface{} { return h.CompleteWorkflowExecutionFailedEventAttributes },
	swf.EventTypeWorkflowExecutionFailed:              func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionFailedEventAttributes },
	swf.EventTypeFailWorkflowExecutionFailed:          func(h *swf.HistoryEvent) interface{} { return h.FailWorkflowExecutionFailedEventAttributes },
	swf.EventTypeWorkflowExecutionTimedOut:            func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionTimedOutEventAttributes },
	swf.EventTypeWorkflowExecutionCanceled:            func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionCanceledEventAttributes },
	swf.EventTypeCancelWorkflowExecutionFailed:        func(h *swf.HistoryEvent) interface{} { return h.CancelWorkflowExecutionFailedEventAttributes },
	swf.EventTypeWorkflowExecutionContinuedAsNew:      func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionContinuedAsNewEventAttributes },
	swf.EventTypeContinueAsNewWorkflowExecutionFailed: func(h *swf.HistoryEvent) interface{} { return h.ContinueAsNewWorkflowExecutionFailedEventAttributes },
	swf.EventTypeWorkflowExecutionTerminated:          func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionTerminatedEventAttributes },
	swf.EventTypeDecisionTaskScheduled:                func(h *swf.HistoryEvent) interface{} { return h.DecisionTaskScheduledEventAttributes },
	swf.EventTypeDecisionTaskStarted:                  func(h *swf.HistoryEvent) interface{} { return h.DecisionTaskStartedEventAttributes },
	swf.EventTypeDecisionTaskCompleted:                func(h *swf.HistoryEvent) interface{} { return h.DecisionTaskCompletedEventAttributes },
	swf.EventTypeDecisionTaskTimedOut:                 func(h *swf.HistoryEvent) interface{} { return h.DecisionTaskTimedOutEventAttributes },
	swf.EventTypeActivityTaskScheduled:                func(h *swf.HistoryEvent) interface{} { return h.ActivityTaskScheduledEventAttributes },
	swf.EventTypeScheduleActivityTaskFailed:           func(h *swf.HistoryEvent) interface{} { return h.ScheduleActivityTaskFailedEventAttributes },
	swf.EventTypeActivityTaskStarted:                  func(h *swf.HistoryEvent) interface{} { return h.ActivityTaskStartedEventAttributes },
	swf.EventTypeActivityTaskCompleted:                func(h *swf.HistoryEvent) interface{} { return h.ActivityTaskCompletedEventAttributes },
	swf.EventTypeActivityTaskFailed:                   func(h *swf.HistoryEvent) interface{} { return h.ActivityTaskFailedEventAttributes },
	swf.EventTypeActivityTaskTimedOut:                 func(h *swf.HistoryEvent) interface{} { return h.ActivityTaskTimedOutEventAttributes },
	swf.EventTypeActivityTaskCanceled:                 func(h *swf.HistoryEvent) interface{} { return h.ActivityTaskCanceledEventAttributes },
	swf.EventTypeActivityTaskCancelRequested:          func(h *swf.HistoryEvent) interface{} { return h.ActivityTaskCancelRequestedEventAttributes },
	swf.EventTypeRequestCancelActivityTaskFailed:      func(h *swf.HistoryEvent) interface{} { return h.RequestCancelActivityTaskFailedEventAttributes },
	swf.EventTypeWorkflowExecutionSignaled:            func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionSignaledEventAttributes },
	swf.EventTypeMarkerRecorded:                       func(h *swf.HistoryEvent) interface{} { return h.MarkerRecordedEventAttributes },
	swf.EventTypeRecordMarkerFailed:                   func(h *swf.HistoryEvent) interface{} { return h.RecordMarkerFailedEventAttributes },
	swf.EventTypeTimerStarted:                         func(h *swf.HistoryEvent) interface{} { return h.TimerStartedEventAttributes },
	swf.EventTypeStartTimerFailed:                     func(h *swf.HistoryEvent) interface{} { return h.StartTimerFailedEventAttributes },
	swf.EventTypeTimerFired:                           func(h *swf.HistoryEvent) interface{} { return h.TimerFiredEventAttributes },
	swf.EventTypeTimerCanceled:                        func(h *swf.HistoryEvent) interface{} { return h.TimerCanceledEventAttributes },
	swf.EventTypeCancelTimerFailed:                    func(h *swf.HistoryEvent) interface{} { return h.CancelTimerFailedEventAttributes },
	swf.EventTypeStartChildWorkflowExecutionInitiated: func(h *swf.HistoryEvent) interface{} { return h.StartChildWorkflowExecutionInitiatedEventAttributes },
	swf.EventTypeStartChildWorkflowExecutionFailed:    func(h *swf.HistoryEvent) interface{} { return h.StartChildWorkflowExecutionFailedEventAttributes },
	swf.EventTypeChildWorkflowExecutionStarted:        func(h *swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionStartedEventAttributes },
	swf.EventTypeChildWorkflowExecutionCompleted:      func(h *swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionCompletedEventAttributes },
	swf.EventTypeChildWorkflowExecutionFailed:         func(h *swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionFailedEventAttributes },
	swf.EventTypeChildWorkflowExecutionTimedOut:       func(h *swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionTimedOutEventAttributes },
	swf.EventTypeChildWorkflowExecutionCanceled:       func(h *swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionCanceledEventAttributes },
	swf.EventTypeChildWorkflowExecutionTerminated:     func(h *swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionTerminatedEventAttributes },
	swf.EventTypeSignalExternalWorkflowExecutionInitiated: func(h *swf.HistoryEvent) interface{} {
		return h.SignalExternalWorkflowExecutionInitiatedEventAttributes
	},
	swf.EventTypeSignalExternalWorkflowExecutionFailed: func(h *swf.HistoryEvent) interface{} { return h.SignalExternalWorkflowExecutionFailedEventAttributes },
	swf.EventTypeExternalWorkflowExecutionSignaled:     func(h *swf.HistoryEvent) interface{} { return h.ExternalWorkflowExecutionSignaledEventAttributes },
	swf.EventTypeRequestCancelExternalWorkflowExecutionInitiated: func(h *swf.HistoryEvent) interface{} {
		return h.RequestCancelExternalWorkflowExecutionInitiatedEventAttributes
	},
	swf.EventTypeRequestCancelExternalWorkflowExecutionFailed: func(h *swf.HistoryEvent) interface{} {
		return h.RequestCancelExternalWorkflowExecutionFailedEventAttributes
	},
	swf.EventTypeExternalWorkflowExecutionCancelRequested: func(h *swf.HistoryEvent) interface{} {
		return h.ExternalWorkflowExecutionCancelRequestedEventAttributes
	},
}

//EventFromPayload will construct swf.HistoryEvent with the correct id, event type and Attributes struct set, based on the type of the data passed to it,
//which should be one of the swf.*EventAttributes structs.
func EventFromPayload(eventId int, data interface{}) *swf.HistoryEvent {
	event := &swf.HistoryEvent{}
	event.EventId = I(eventId)
	switch t := data.(type) {
	case *swf.ActivityTaskCancelRequestedEventAttributes:
		event.ActivityTaskCancelRequestedEventAttributes = t
		event.EventType = S(swf.EventTypeActivityTaskCancelRequested)
	case *swf.ActivityTaskCanceledEventAttributes:
		event.ActivityTaskCanceledEventAttributes = t
		event.EventType = S(swf.EventTypeActivityTaskCanceled)
	case *swf.ActivityTaskCompletedEventAttributes:
		event.ActivityTaskCompletedEventAttributes = t
		event.EventType = S(swf.EventTypeActivityTaskCompleted)
	case *swf.ActivityTaskFailedEventAttributes:
		event.ActivityTaskFailedEventAttributes = t
		event.EventType = S(swf.EventTypeActivityTaskFailed)
	case *swf.ActivityTaskScheduledEventAttributes:
		event.ActivityTaskScheduledEventAttributes = t
		event.EventType = S(swf.EventTypeActivityTaskScheduled)
	case *swf.ActivityTaskStartedEventAttributes:
		event.ActivityTaskStartedEventAttributes = t
		event.EventType = S(swf.EventTypeActivityTaskStarted)
	case *swf.ActivityTaskTimedOutEventAttributes:
		event.ActivityTaskTimedOutEventAttributes = t
		event.EventType = S(swf.EventTypeActivityTaskTimedOut)
	case *swf.CancelTimerFailedEventAttributes:
		event.CancelTimerFailedEventAttributes = t
		event.EventType = S(swf.EventTypeCancelTimerFailed)
	case *swf.CancelWorkflowExecutionFailedEventAttributes:
		event.CancelWorkflowExecutionFailedEventAttributes = t
		event.EventType = S(swf.EventTypeCancelWorkflowExecutionFailed)
	case *swf.ChildWorkflowExecutionCanceledEventAttributes:
		event.ChildWorkflowExecutionCanceledEventAttributes = t
		event.EventType = S(swf.EventTypeChildWorkflowExecutionCanceled)
	case *swf.ChildWorkflowExecutionCompletedEventAttributes:
		event.ChildWorkflowExecutionCompletedEventAttributes = t
		event.EventType = S(swf.EventTypeChildWorkflowExecutionCompleted)
	case *swf.ChildWorkflowExecutionFailedEventAttributes:
		event.ChildWorkflowExecutionFailedEventAttributes = t
		event.EventType = S(swf.EventTypeChildWorkflowExecutionFailed)
	case *swf.ChildWorkflowExecutionStartedEventAttributes:
		event.ChildWorkflowExecutionStartedEventAttributes = t
		event.EventType = S(swf.EventTypeChildWorkflowExecutionStarted)
	case *swf.ChildWorkflowExecutionTerminatedEventAttributes:
		event.ChildWorkflowExecutionTerminatedEventAttributes = t
		event.EventType = S(swf.EventTypeChildWorkflowExecutionTerminated)
	case *swf.ChildWorkflowExecutionTimedOutEventAttributes:
		event.ChildWorkflowExecutionTimedOutEventAttributes = t
		event.EventType = S(swf.EventTypeChildWorkflowExecutionTimedOut)
	case *swf.CompleteWorkflowExecutionFailedEventAttributes:
		event.CompleteWorkflowExecutionFailedEventAttributes = t
		event.EventType = S(swf.EventTypeCompleteWorkflowExecutionFailed)
	case *swf.ContinueAsNewWorkflowExecutionFailedEventAttributes:
		event.ContinueAsNewWorkflowExecutionFailedEventAttributes = t
		event.EventType = S(swf.EventTypeContinueAsNewWorkflowExecutionFailed)
	case *swf.DecisionTaskCompletedEventAttributes:
		event.DecisionTaskCompletedEventAttributes = t
		event.EventType = S(swf.EventTypeDecisionTaskCompleted)
	case *swf.DecisionTaskScheduledEventAttributes:
		event.DecisionTaskScheduledEventAttributes = t
		event.EventType = S(swf.EventTypeDecisionTaskScheduled)
	case *swf.DecisionTaskStartedEventAttributes:
		event.DecisionTaskStartedEventAttributes = t
		event.EventType = S(swf.EventTypeDecisionTaskStarted)
	case *swf.DecisionTaskTimedOutEventAttributes:
		event.DecisionTaskTimedOutEventAttributes = t
		event.EventType = S(swf.EventTypeDecisionTaskTimedOut)
	case *swf.ExternalWorkflowExecutionCancelRequestedEventAttributes:
		event.ExternalWorkflowExecutionCancelRequestedEventAttributes = t
		event.EventType = S(swf.EventTypeExternalWorkflowExecutionCancelRequested)
	case *swf.ExternalWorkflowExecutionSignaledEventAttributes:
		event.ExternalWorkflowExecutionSignaledEventAttributes = t
		event.EventType = S(swf.EventTypeExternalWorkflowExecutionSignaled)
	case *swf.FailWorkflowExecutionFailedEventAttributes:
		event.FailWorkflowExecutionFailedEventAttributes = t
		event.EventType = S(swf.EventTypeFailWorkflowExecutionFailed)
	case *swf.MarkerRecordedEventAttributes:
		event.MarkerRecordedEventAttributes = t
		event.EventType = S(swf.EventTypeMarkerRecorded)
	case *swf.RecordMarkerFailedEventAttributes:
		event.RecordMarkerFailedEventAttributes = t
		event.EventType = S(swf.EventTypeRecordMarkerFailed)
	case *swf.RequestCancelActivityTaskFailedEventAttributes:
		event.RequestCancelActivityTaskFailedEventAttributes = t
		event.EventType = S(swf.EventTypeRequestCancelActivityTaskFailed)
	case *swf.RequestCancelExternalWorkflowExecutionFailedEventAttributes:
		event.RequestCancelExternalWorkflowExecutionFailedEventAttributes = t
		event.EventType = S(swf.EventTypeRequestCancelExternalWorkflowExecutionFailed)
	case *swf.RequestCancelExternalWorkflowExecutionInitiatedEventAttributes:
		event.RequestCancelExternalWorkflowExecutionInitiatedEventAttributes = t
		event.EventType = S(swf.EventTypeRequestCancelExternalWorkflowExecutionInitiated)
	case *swf.ScheduleActivityTaskFailedEventAttributes:
		event.ScheduleActivityTaskFailedEventAttributes = t
		event.EventType = S(swf.EventTypeScheduleActivityTaskFailed)
	case *swf.SignalExternalWorkflowExecutionFailedEventAttributes:
		event.SignalExternalWorkflowExecutionFailedEventAttributes = t
		event.EventType = S(swf.EventTypeSignalExternalWorkflowExecutionFailed)
	case *swf.SignalExternalWorkflowExecutionInitiatedEventAttributes:
		event.SignalExternalWorkflowExecutionInitiatedEventAttributes = t
		event.EventType = S(swf.EventTypeSignalExternalWorkflowExecutionInitiated)
	case *swf.StartChildWorkflowExecutionFailedEventAttributes:
		event.StartChildWorkflowExecutionFailedEventAttributes = t
		event.EventType = S(swf.EventTypeStartChildWorkflowExecutionFailed)
	case *swf.StartChildWorkflowExecutionInitiatedEventAttributes:
		event.StartChildWorkflowExecutionInitiatedEventAttributes = t
		event.EventType = S(swf.EventTypeStartChildWorkflowExecutionInitiated)
	case *swf.StartTimerFailedEventAttributes:
		event.StartTimerFailedEventAttributes = t
		event.EventType = S(swf.EventTypeStartTimerFailed)
	case *swf.TimerCanceledEventAttributes:
		event.TimerCanceledEventAttributes = t
		event.EventType = S(swf.EventTypeTimerCanceled)
	case *swf.TimerFiredEventAttributes:
		event.TimerFiredEventAttributes = t
		event.EventType = S(swf.EventTypeTimerFired)
	case *swf.TimerStartedEventAttributes:
		event.TimerStartedEventAttributes = t
		event.EventType = S(swf.EventTypeTimerStarted)
	case *swf.WorkflowExecutionCancelRequestedEventAttributes:
		event.WorkflowExecutionCancelRequestedEventAttributes = t
		event.EventType = S(swf.EventTypeWorkflowExecutionCancelRequested)
	case *swf.WorkflowExecutionCanceledEventAttributes:
		event.WorkflowExecutionCanceledEventAttributes = t
		event.EventType = S(swf.EventTypeWorkflowExecutionCanceled)
	case *swf.WorkflowExecutionCompletedEventAttributes:
		event.WorkflowExecutionCompletedEventAttributes = t
		event.EventType = S(swf.EventTypeWorkflowExecutionCompleted)
	case *swf.WorkflowExecutionContinuedAsNewEventAttributes:
		event.WorkflowExecutionContinuedAsNewEventAttributes = t
		event.EventType = S(swf.EventTypeWorkflowExecutionContinuedAsNew)
	case *swf.WorkflowExecutionFailedEventAttributes:
		event.WorkflowExecutionFailedEventAttributes = t
		event.EventType = S(swf.EventTypeWorkflowExecutionFailed)
	case *swf.WorkflowExecutionSignaledEventAttributes:
		event.WorkflowExecutionSignaledEventAttributes = t
		event.EventType = S(swf.EventTypeWorkflowExecutionSignaled)
	case *swf.WorkflowExecutionStartedEventAttributes:
		event.WorkflowExecutionStartedEventAttributes = t
		event.EventType = S(swf.EventTypeWorkflowExecutionStarted)
	case *swf.WorkflowExecutionTerminatedEventAttributes:
		event.WorkflowExecutionTerminatedEventAttributes = t
		event.EventType = S(swf.EventTypeWorkflowExecutionTerminated)
	case *swf.WorkflowExecutionTimedOutEventAttributes:
		event.WorkflowExecutionTimedOutEventAttributes = t
		event.EventType = S(swf.EventTypeWorkflowExecutionTimedOut)
	}
	return event
}

//PrettyHistoryEvent pretty prints a swf.HistoryEvent in a readable form for logging.
func PrettyHistoryEvent(h *swf.HistoryEvent) string {
	var buffer bytes.Buffer
	buffer.WriteString("HistoryEvent{ ")
	if h.EventId != nil {
		buffer.WriteString(fmt.Sprintf("EventId: %d,", *h.EventId))
	}
	if h.EventTimestamp != nil {
		buffer.WriteString(fmt.Sprintf("EventTimestamp: %s, ", *h.EventTimestamp))
	}
	if h.EventType != nil {
		buffer.WriteString(fmt.Sprintf("EventType: %s,", *h.EventType))
	}
	buffer.WriteString(fmt.Sprintf(" %+v", eventTypes[*h.EventType](h)))
	buffer.WriteString(" }")
	return buffer.String()
}

// SWFHistoryEventTypes returns a slice containing the valid values of the HistoryEvent.EventType field.
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

//PrettyDecision pretty prints a swf.Decision in a readable form for logging.
func PrettyDecision(d swf.Decision) string {
	var buffer bytes.Buffer
	buffer.WriteString("Decision{ ")
	if d.DecisionType != nil {
		buffer.WriteString(fmt.Sprintf("DecisionType: %s,", *d.DecisionType))
		if decisionTypes[*d.DecisionType](d) != nil {
			buffer.WriteString(fmt.Sprintf("%+v", decisionTypes[*d.DecisionType](d)))
		}
	} else {
		buffer.WriteString("NIL DECISION TYPE")
	}

	buffer.WriteString(" }")
	return buffer.String()
}

// SWFDecisionTypes returns a slice containing the valid values of the Decision.DecisionType field.
func SWFDecisionTypes() []string {
	ds := make([]string, 0, len(decisionTypes))
	for k := range eventTypes {
		ds = append(ds, k)
	}
	return ds
}

//L is a helper so you dont have to type aws.Long(myLong)
func L(l int64) *int64 {
	return aws.Int64(l)
}

//I is a helper so you dont have to type aws.Long(int64(myInt))
func I(i int) *int64 {
	return aws.Int64(int64(i))
}

//S is a helper so you dont have to type aws.String(myString)
func S(s string) *string {
	return aws.String(s)
}

//LS is a helper so you dont have to do nil checks before logging aws.StringValue values
func LS(s *string) string {
	if s == nil {
		return "nil"
	}
	return *s
}

//LL is a helper so you dont have to do nil checks before logging aws.LongValue values
func LL(l *int64) string {
	if l == nil {
		return "nil"
	}
	return strconv.FormatInt(*l, 10)
}
