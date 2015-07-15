package sugar

import (
	"bytes"
	"fmt"

	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/sclasen/swfsm/enums/swf"
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
	enums.EventTypeWorkflowExecutionStarted:             func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionStartedEventAttributes },
	enums.EventTypeWorkflowExecutionCancelRequested:     func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionCancelRequestedEventAttributes },
	enums.EventTypeWorkflowExecutionCompleted:           func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionCompletedEventAttributes },
	enums.EventTypeCompleteWorkflowExecutionFailed:      func(h *swf.HistoryEvent) interface{} { return h.CompleteWorkflowExecutionFailedEventAttributes },
	enums.EventTypeWorkflowExecutionFailed:              func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionFailedEventAttributes },
	enums.EventTypeFailWorkflowExecutionFailed:          func(h *swf.HistoryEvent) interface{} { return h.FailWorkflowExecutionFailedEventAttributes },
	enums.EventTypeWorkflowExecutionTimedOut:            func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionTimedOutEventAttributes },
	enums.EventTypeWorkflowExecutionCanceled:            func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionCanceledEventAttributes },
	enums.EventTypeCancelWorkflowExecutionFailed:        func(h *swf.HistoryEvent) interface{} { return h.CancelWorkflowExecutionFailedEventAttributes },
	enums.EventTypeWorkflowExecutionContinuedAsNew:      func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionContinuedAsNewEventAttributes },
	enums.EventTypeContinueAsNewWorkflowExecutionFailed: func(h *swf.HistoryEvent) interface{} { return h.ContinueAsNewWorkflowExecutionFailedEventAttributes },
	enums.EventTypeWorkflowExecutionTerminated:          func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionTerminatedEventAttributes },
	enums.EventTypeDecisionTaskScheduled:                func(h *swf.HistoryEvent) interface{} { return h.DecisionTaskScheduledEventAttributes },
	enums.EventTypeDecisionTaskStarted:                  func(h *swf.HistoryEvent) interface{} { return h.DecisionTaskStartedEventAttributes },
	enums.EventTypeDecisionTaskCompleted:                func(h *swf.HistoryEvent) interface{} { return h.DecisionTaskCompletedEventAttributes },
	enums.EventTypeDecisionTaskTimedOut:                 func(h *swf.HistoryEvent) interface{} { return h.DecisionTaskTimedOutEventAttributes },
	enums.EventTypeActivityTaskScheduled:                func(h *swf.HistoryEvent) interface{} { return h.ActivityTaskScheduledEventAttributes },
	enums.EventTypeScheduleActivityTaskFailed:           func(h *swf.HistoryEvent) interface{} { return h.ScheduleActivityTaskFailedEventAttributes },
	enums.EventTypeActivityTaskStarted:                  func(h *swf.HistoryEvent) interface{} { return h.ActivityTaskStartedEventAttributes },
	enums.EventTypeActivityTaskCompleted:                func(h *swf.HistoryEvent) interface{} { return h.ActivityTaskCompletedEventAttributes },
	enums.EventTypeActivityTaskFailed:                   func(h *swf.HistoryEvent) interface{} { return h.ActivityTaskFailedEventAttributes },
	enums.EventTypeActivityTaskTimedOut:                 func(h *swf.HistoryEvent) interface{} { return h.ActivityTaskTimedOutEventAttributes },
	enums.EventTypeActivityTaskCanceled:                 func(h *swf.HistoryEvent) interface{} { return h.ActivityTaskCanceledEventAttributes },
	enums.EventTypeActivityTaskCancelRequested:          func(h *swf.HistoryEvent) interface{} { return h.ActivityTaskCancelRequestedEventAttributes },
	enums.EventTypeRequestCancelActivityTaskFailed:      func(h *swf.HistoryEvent) interface{} { return h.RequestCancelActivityTaskFailedEventAttributes },
	enums.EventTypeWorkflowExecutionSignaled:            func(h *swf.HistoryEvent) interface{} { return h.WorkflowExecutionSignaledEventAttributes },
	enums.EventTypeMarkerRecorded:                       func(h *swf.HistoryEvent) interface{} { return h.MarkerRecordedEventAttributes },
	enums.EventTypeRecordMarkerFailed:                   func(h *swf.HistoryEvent) interface{} { return h.RecordMarkerFailedEventAttributes },
	enums.EventTypeTimerStarted:                         func(h *swf.HistoryEvent) interface{} { return h.TimerStartedEventAttributes },
	enums.EventTypeStartTimerFailed:                     func(h *swf.HistoryEvent) interface{} { return h.StartTimerFailedEventAttributes },
	enums.EventTypeTimerFired:                           func(h *swf.HistoryEvent) interface{} { return h.TimerFiredEventAttributes },
	enums.EventTypeTimerCanceled:                        func(h *swf.HistoryEvent) interface{} { return h.TimerCanceledEventAttributes },
	enums.EventTypeCancelTimerFailed:                    func(h *swf.HistoryEvent) interface{} { return h.CancelTimerFailedEventAttributes },
	enums.EventTypeStartChildWorkflowExecutionInitiated: func(h *swf.HistoryEvent) interface{} { return h.StartChildWorkflowExecutionInitiatedEventAttributes },
	enums.EventTypeStartChildWorkflowExecutionFailed:    func(h *swf.HistoryEvent) interface{} { return h.StartChildWorkflowExecutionFailedEventAttributes },
	enums.EventTypeChildWorkflowExecutionStarted:        func(h *swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionStartedEventAttributes },
	enums.EventTypeChildWorkflowExecutionCompleted:      func(h *swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionCompletedEventAttributes },
	enums.EventTypeChildWorkflowExecutionFailed:         func(h *swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionFailedEventAttributes },
	enums.EventTypeChildWorkflowExecutionTimedOut:       func(h *swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionTimedOutEventAttributes },
	enums.EventTypeChildWorkflowExecutionCanceled:       func(h *swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionCanceledEventAttributes },
	enums.EventTypeChildWorkflowExecutionTerminated:     func(h *swf.HistoryEvent) interface{} { return h.ChildWorkflowExecutionTerminatedEventAttributes },
	enums.EventTypeSignalExternalWorkflowExecutionInitiated: func(h *swf.HistoryEvent) interface{} {
		return h.SignalExternalWorkflowExecutionInitiatedEventAttributes
	},
	enums.EventTypeSignalExternalWorkflowExecutionFailed: func(h *swf.HistoryEvent) interface{} { return h.SignalExternalWorkflowExecutionFailedEventAttributes },
	enums.EventTypeExternalWorkflowExecutionSignaled:     func(h *swf.HistoryEvent) interface{} { return h.ExternalWorkflowExecutionSignaledEventAttributes },
	enums.EventTypeRequestCancelExternalWorkflowExecutionInitiated: func(h *swf.HistoryEvent) interface{} {
		return h.RequestCancelExternalWorkflowExecutionInitiatedEventAttributes
	},
	enums.EventTypeRequestCancelExternalWorkflowExecutionFailed: func(h *swf.HistoryEvent) interface{} {
		return h.RequestCancelExternalWorkflowExecutionFailedEventAttributes
	},
	enums.EventTypeExternalWorkflowExecutionCancelRequested: func(h *swf.HistoryEvent) interface{} {
		return h.ExternalWorkflowExecutionCancelRequestedEventAttributes
	},
}

//EventFromPayload will construct swf.HistoryEvent with the correct id, event type and Attributes struct set, based on the type of the data passed to it,
//which should be one of the swf.*EventAttributes structs.
func EventFromPayload(eventID int, data interface{}) *swf.HistoryEvent {
	event := &swf.HistoryEvent{}
	event.EventID = I(eventID)
	switch t := data.(type) {
	case *swf.ActivityTaskCancelRequestedEventAttributes:
		event.ActivityTaskCancelRequestedEventAttributes = t
		event.EventType = S(enums.EventTypeActivityTaskCancelRequested)
	case *swf.ActivityTaskCanceledEventAttributes:
		event.ActivityTaskCanceledEventAttributes = t
		event.EventType = S(enums.EventTypeActivityTaskCanceled)
	case *swf.ActivityTaskCompletedEventAttributes:
		event.ActivityTaskCompletedEventAttributes = t
		event.EventType = S(enums.EventTypeActivityTaskCompleted)
	case *swf.ActivityTaskFailedEventAttributes:
		event.ActivityTaskFailedEventAttributes = t
		event.EventType = S(enums.EventTypeActivityTaskFailed)
	case *swf.ActivityTaskScheduledEventAttributes:
		event.ActivityTaskScheduledEventAttributes = t
		event.EventType = S(enums.EventTypeActivityTaskScheduled)
	case *swf.ActivityTaskStartedEventAttributes:
		event.ActivityTaskStartedEventAttributes = t
		event.EventType = S(enums.EventTypeActivityTaskStarted)
	case *swf.ActivityTaskTimedOutEventAttributes:
		event.ActivityTaskTimedOutEventAttributes = t
		event.EventType = S(enums.EventTypeActivityTaskTimedOut)
	case *swf.CancelTimerFailedEventAttributes:
		event.CancelTimerFailedEventAttributes = t
		event.EventType = S(enums.EventTypeCancelTimerFailed)
	case *swf.CancelWorkflowExecutionFailedEventAttributes:
		event.CancelWorkflowExecutionFailedEventAttributes = t
		event.EventType = S(enums.EventTypeCancelWorkflowExecutionFailed)
	case *swf.ChildWorkflowExecutionCanceledEventAttributes:
		event.ChildWorkflowExecutionCanceledEventAttributes = t
		event.EventType = S(enums.EventTypeChildWorkflowExecutionCanceled)
	case *swf.ChildWorkflowExecutionCompletedEventAttributes:
		event.ChildWorkflowExecutionCompletedEventAttributes = t
		event.EventType = S(enums.EventTypeChildWorkflowExecutionCompleted)
	case *swf.ChildWorkflowExecutionFailedEventAttributes:
		event.ChildWorkflowExecutionFailedEventAttributes = t
		event.EventType = S(enums.EventTypeChildWorkflowExecutionFailed)
	case *swf.ChildWorkflowExecutionStartedEventAttributes:
		event.ChildWorkflowExecutionStartedEventAttributes = t
		event.EventType = S(enums.EventTypeChildWorkflowExecutionStarted)
	case *swf.ChildWorkflowExecutionTerminatedEventAttributes:
		event.ChildWorkflowExecutionTerminatedEventAttributes = t
		event.EventType = S(enums.EventTypeChildWorkflowExecutionTerminated)
	case *swf.ChildWorkflowExecutionTimedOutEventAttributes:
		event.ChildWorkflowExecutionTimedOutEventAttributes = t
		event.EventType = S(enums.EventTypeChildWorkflowExecutionTimedOut)
	case *swf.CompleteWorkflowExecutionFailedEventAttributes:
		event.CompleteWorkflowExecutionFailedEventAttributes = t
		event.EventType = S(enums.EventTypeCompleteWorkflowExecutionFailed)
	case *swf.ContinueAsNewWorkflowExecutionFailedEventAttributes:
		event.ContinueAsNewWorkflowExecutionFailedEventAttributes = t
		event.EventType = S(enums.EventTypeContinueAsNewWorkflowExecutionFailed)
	case *swf.DecisionTaskCompletedEventAttributes:
		event.DecisionTaskCompletedEventAttributes = t
		event.EventType = S(enums.EventTypeDecisionTaskCompleted)
	case *swf.DecisionTaskScheduledEventAttributes:
		event.DecisionTaskScheduledEventAttributes = t
		event.EventType = S(enums.EventTypeDecisionTaskScheduled)
	case *swf.DecisionTaskStartedEventAttributes:
		event.DecisionTaskStartedEventAttributes = t
		event.EventType = S(enums.EventTypeDecisionTaskStarted)
	case *swf.DecisionTaskTimedOutEventAttributes:
		event.DecisionTaskTimedOutEventAttributes = t
		event.EventType = S(enums.EventTypeDecisionTaskTimedOut)
	case *swf.ExternalWorkflowExecutionCancelRequestedEventAttributes:
		event.ExternalWorkflowExecutionCancelRequestedEventAttributes = t
		event.EventType = S(enums.EventTypeExternalWorkflowExecutionCancelRequested)
	case *swf.ExternalWorkflowExecutionSignaledEventAttributes:
		event.ExternalWorkflowExecutionSignaledEventAttributes = t
		event.EventType = S(enums.EventTypeExternalWorkflowExecutionSignaled)
	case *swf.FailWorkflowExecutionFailedEventAttributes:
		event.FailWorkflowExecutionFailedEventAttributes = t
		event.EventType = S(enums.EventTypeFailWorkflowExecutionFailed)
	case *swf.MarkerRecordedEventAttributes:
		event.MarkerRecordedEventAttributes = t
		event.EventType = S(enums.EventTypeMarkerRecorded)
	case *swf.RecordMarkerFailedEventAttributes:
		event.RecordMarkerFailedEventAttributes = t
		event.EventType = S(enums.EventTypeRecordMarkerFailed)
	case *swf.RequestCancelActivityTaskFailedEventAttributes:
		event.RequestCancelActivityTaskFailedEventAttributes = t
		event.EventType = S(enums.EventTypeRequestCancelActivityTaskFailed)
	case *swf.RequestCancelExternalWorkflowExecutionFailedEventAttributes:
		event.RequestCancelExternalWorkflowExecutionFailedEventAttributes = t
		event.EventType = S(enums.EventTypeRequestCancelExternalWorkflowExecutionFailed)
	case *swf.RequestCancelExternalWorkflowExecutionInitiatedEventAttributes:
		event.RequestCancelExternalWorkflowExecutionInitiatedEventAttributes = t
		event.EventType = S(enums.EventTypeRequestCancelExternalWorkflowExecutionInitiated)
	case *swf.ScheduleActivityTaskFailedEventAttributes:
		event.ScheduleActivityTaskFailedEventAttributes = t
		event.EventType = S(enums.EventTypeScheduleActivityTaskFailed)
	case *swf.SignalExternalWorkflowExecutionFailedEventAttributes:
		event.SignalExternalWorkflowExecutionFailedEventAttributes = t
		event.EventType = S(enums.EventTypeSignalExternalWorkflowExecutionFailed)
	case *swf.SignalExternalWorkflowExecutionInitiatedEventAttributes:
		event.SignalExternalWorkflowExecutionInitiatedEventAttributes = t
		event.EventType = S(enums.EventTypeSignalExternalWorkflowExecutionInitiated)
	case *swf.StartChildWorkflowExecutionFailedEventAttributes:
		event.StartChildWorkflowExecutionFailedEventAttributes = t
		event.EventType = S(enums.EventTypeStartChildWorkflowExecutionFailed)
	case *swf.StartChildWorkflowExecutionInitiatedEventAttributes:
		event.StartChildWorkflowExecutionInitiatedEventAttributes = t
		event.EventType = S(enums.EventTypeStartChildWorkflowExecutionInitiated)
	case *swf.StartTimerFailedEventAttributes:
		event.StartTimerFailedEventAttributes = t
		event.EventType = S(enums.EventTypeStartTimerFailed)
	case *swf.TimerCanceledEventAttributes:
		event.TimerCanceledEventAttributes = t
		event.EventType = S(enums.EventTypeTimerCanceled)
	case *swf.TimerFiredEventAttributes:
		event.TimerFiredEventAttributes = t
		event.EventType = S(enums.EventTypeTimerFired)
	case *swf.TimerStartedEventAttributes:
		event.TimerStartedEventAttributes = t
		event.EventType = S(enums.EventTypeTimerStarted)
	case *swf.WorkflowExecutionCancelRequestedEventAttributes:
		event.WorkflowExecutionCancelRequestedEventAttributes = t
		event.EventType = S(enums.EventTypeWorkflowExecutionCancelRequested)
	case *swf.WorkflowExecutionCanceledEventAttributes:
		event.WorkflowExecutionCanceledEventAttributes = t
		event.EventType = S(enums.EventTypeWorkflowExecutionCanceled)
	case *swf.WorkflowExecutionCompletedEventAttributes:
		event.WorkflowExecutionCompletedEventAttributes = t
		event.EventType = S(enums.EventTypeWorkflowExecutionCompleted)
	case *swf.WorkflowExecutionContinuedAsNewEventAttributes:
		event.WorkflowExecutionContinuedAsNewEventAttributes = t
		event.EventType = S(enums.EventTypeWorkflowExecutionContinuedAsNew)
	case *swf.WorkflowExecutionFailedEventAttributes:
		event.WorkflowExecutionFailedEventAttributes = t
		event.EventType = S(enums.EventTypeWorkflowExecutionFailed)
	case *swf.WorkflowExecutionSignaledEventAttributes:
		event.WorkflowExecutionSignaledEventAttributes = t
		event.EventType = S(enums.EventTypeWorkflowExecutionSignaled)
	case *swf.WorkflowExecutionStartedEventAttributes:
		event.WorkflowExecutionStartedEventAttributes = t
		event.EventType = S(enums.EventTypeWorkflowExecutionStarted)
	case *swf.WorkflowExecutionTerminatedEventAttributes:
		event.WorkflowExecutionTerminatedEventAttributes = t
		event.EventType = S(enums.EventTypeWorkflowExecutionTerminated)
	case *swf.WorkflowExecutionTimedOutEventAttributes:
		event.WorkflowExecutionTimedOutEventAttributes = t
		event.EventType = S(enums.EventTypeWorkflowExecutionTimedOut)
	}
	return event
}

//PrettyHistoryEvent pretty prints a swf.HistoryEvent in a readable form for logging.
func PrettyHistoryEvent(h *swf.HistoryEvent) string {
	var buffer bytes.Buffer
	buffer.WriteString("HistoryEvent{ ")
	if h.EventID != nil {
		buffer.WriteString(fmt.Sprintf("EventId: %d,", *h.EventID))
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
	enums.DecisionTypeScheduleActivityTask:                   func(d swf.Decision) interface{} { return d.ScheduleActivityTaskDecisionAttributes },
	enums.DecisionTypeRequestCancelActivityTask:              func(d swf.Decision) interface{} { return d.RequestCancelActivityTaskDecisionAttributes },
	enums.DecisionTypeCompleteWorkflowExecution:              func(d swf.Decision) interface{} { return d.CompleteWorkflowExecutionDecisionAttributes },
	enums.DecisionTypeFailWorkflowExecution:                  func(d swf.Decision) interface{} { return d.FailWorkflowExecutionDecisionAttributes },
	enums.DecisionTypeCancelWorkflowExecution:                func(d swf.Decision) interface{} { return d.CancelWorkflowExecutionDecisionAttributes },
	enums.DecisionTypeContinueAsNewWorkflowExecution:         func(d swf.Decision) interface{} { return d.ContinueAsNewWorkflowExecutionDecisionAttributes },
	enums.DecisionTypeRecordMarker:                           func(d swf.Decision) interface{} { return d.RecordMarkerDecisionAttributes },
	enums.DecisionTypeStartTimer:                             func(d swf.Decision) interface{} { return d.StartTimerDecisionAttributes },
	enums.DecisionTypeCancelTimer:                            func(d swf.Decision) interface{} { return d.CancelTimerDecisionAttributes },
	enums.DecisionTypeSignalExternalWorkflowExecution:        func(d swf.Decision) interface{} { return d.SignalExternalWorkflowExecutionDecisionAttributes },
	enums.DecisionTypeRequestCancelExternalWorkflowExecution: func(d swf.Decision) interface{} { return d.RequestCancelExternalWorkflowExecutionDecisionAttributes },
	enums.DecisionTypeStartChildWorkflowExecution:            func(d swf.Decision) interface{} { return d.StartChildWorkflowExecutionDecisionAttributes },
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
	return aws.Long(l)
}

//I is a helper so you dont have to type aws.Long(int64(myInt))
func I(i int) *int64 {
	return aws.Long(int64(i))
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
