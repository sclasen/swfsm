package fsm

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/service/swf"
)

// EventCorrelator is a serialization-friendly struct that is automatically managed by the FSM machinery
// It tracks signal and activity correlation info, so you know how to react when an event that signals the
// end of an activity or signal  hits your Decider.  This is missing from the SWF api.
// Activities and Signals are string instead of int64 beacuse json.
type EventCorrelator struct {
	Activities          map[string]*ActivityInfo     // schedueledEventId -> info
	ActivityAttempts    map[string]int               // activityId -> attempts
	Signals             map[string]*SignalInfo       // schedueledEventId -> info
	SignalAttempts      map[string]int               // workflowId + signalName -> attempts
	Timers              map[string]*TimerInfo        // startedEventId -> info
	Cancellations       map[string]*CancellationInfo // schedueledEventId -> info
	CancelationAttempts map[string]int               // workflowId -> attempts
	Children            map[string]*ChildInfo        // initiatedEventID -> info
	ChildrenAttempts    map[string]int               // workflowID -> attempts
	Serializer          StateSerializer              `json:"-"`
}

// ActivityInfo holds the ActivityId and ActivityType for an activity
type ActivityInfo struct {
	ActivityId string
	*swf.ActivityType
	Input *string
}

// SignalInfo holds the SignalName and Input for an activity
type SignalInfo struct {
	SignalName string
	WorkflowId string
	Input      *string
}

//TimerInfo holds the Control data from a Timer
type TimerInfo struct {
	Control            *string
	TimerId            string
	StartToFireTimeout string
}

//CancellationInfo holds the Control data and workflow that was being canceled
type CancellationInfo struct {
	Control    *string
	WorkflowId string
}

//ChildInfo holds the Input data and Workflow info for the child workflow being started
type ChildInfo struct {
	WorkflowId string
	Input      *string
	*swf.WorkflowType
}

// Track will add or remove entries based on the EventType.
// A new entry is added when there is a new ActivityTask, or an entry is removed when the ActivityTask is terminating.
func (a *EventCorrelator) Track(h *swf.HistoryEvent) {
	a.RemoveCorrelation(h)
	a.Correlate(h)
}

// Correlate establishes a mapping of eventId to ActivityType. The HistoryEvent is expected to be of type EventTypeActivityTaskScheduled.
func (a *EventCorrelator) Correlate(h *swf.HistoryEvent) {
	a.checkInit()

	if a.nilSafeEq(h.EventType, swf.EventTypeActivityTaskScheduled) {
		a.Activities[a.key(h.EventId)] = &ActivityInfo{
			ActivityId:   *h.ActivityTaskScheduledEventAttributes.ActivityId,
			ActivityType: h.ActivityTaskScheduledEventAttributes.ActivityType,
			Input:        h.ActivityTaskScheduledEventAttributes.Input,
		}
	}

	if a.nilSafeEq(h.EventType, swf.EventTypeSignalExternalWorkflowExecutionInitiated) {
		a.Signals[a.key(h.EventId)] = &SignalInfo{
			SignalName: *h.SignalExternalWorkflowExecutionInitiatedEventAttributes.SignalName,
			WorkflowId: *h.SignalExternalWorkflowExecutionInitiatedEventAttributes.WorkflowId,
			Input:      h.SignalExternalWorkflowExecutionInitiatedEventAttributes.Input,
		}
	}

	if a.nilSafeEq(h.EventType, swf.EventTypeRequestCancelExternalWorkflowExecutionInitiated) {
		a.Cancellations[a.key(h.EventId)] = &CancellationInfo{
			WorkflowId: *h.RequestCancelExternalWorkflowExecutionInitiatedEventAttributes.WorkflowId,
			Control:    h.RequestCancelExternalWorkflowExecutionInitiatedEventAttributes.Control,
		}
	}

	if a.nilSafeEq(h.EventType, swf.EventTypeTimerStarted) {
		a.Timers[a.key(h.EventId)] = &TimerInfo{
			Control:            h.TimerStartedEventAttributes.Control,
			TimerId:            *h.TimerStartedEventAttributes.TimerId,
			StartToFireTimeout: *h.TimerStartedEventAttributes.StartToFireTimeout,
		}
	}

	if a.nilSafeEq(h.EventType, swf.EventTypeStartChildWorkflowExecutionInitiated) {
		a.Children[a.key(h.EventId)] = &ChildInfo{
			WorkflowId:   *h.StartChildWorkflowExecutionInitiatedEventAttributes.WorkflowId,
			WorkflowType: h.StartChildWorkflowExecutionInitiatedEventAttributes.WorkflowType,
			Input:        h.StartChildWorkflowExecutionInitiatedEventAttributes.Input,
		}
	}

}

// RemoveCorrelation gcs a mapping of eventId to ActivityType. The HistoryEvent is expected to be of type EventTypeActivityTaskCompleted,EventTypeActivityTaskFailed,EventTypeActivityTaskTimedOut.
func (a *EventCorrelator) RemoveCorrelation(h *swf.HistoryEvent) {
	a.checkInit()
	if h.EventType == nil {
		return
	}
	switch *h.EventType {
	/*Activities*/
	case swf.EventTypeActivityTaskCompleted:
		delete(a.ActivityAttempts, a.safeActivityId(h))
		delete(a.Activities, a.key(h.ActivityTaskCompletedEventAttributes.ScheduledEventId))
	case swf.EventTypeActivityTaskFailed:
		a.incrementActivityAttempts(h)
		delete(a.Activities, a.key(h.ActivityTaskFailedEventAttributes.ScheduledEventId))
	case swf.EventTypeActivityTaskTimedOut:
		a.incrementActivityAttempts(h)
		delete(a.Activities, a.key(h.ActivityTaskTimedOutEventAttributes.ScheduledEventId))
	case swf.EventTypeActivityTaskCanceled:
		delete(a.ActivityAttempts, a.safeActivityId(h))
		delete(a.Activities, a.key(h.ActivityTaskCanceledEventAttributes.ScheduledEventId))
	/*Signals*/
	case swf.EventTypeExternalWorkflowExecutionSignaled:
		key := a.key(h.ExternalWorkflowExecutionSignaledEventAttributes.InitiatedEventId)
		info := a.Signals[key]
		delete(a.SignalAttempts, a.signalIdFromInfo(info))
		delete(a.Signals, key)
	case swf.EventTypeSignalExternalWorkflowExecutionFailed:
		a.incrementSignalAttempts(h)
		delete(a.Signals, a.key(h.SignalExternalWorkflowExecutionFailedEventAttributes.InitiatedEventId))
	/*Timers*/
	case swf.EventTypeTimerFired:
		delete(a.Timers, a.key(h.TimerFiredEventAttributes.StartedEventId))
	case swf.EventTypeTimerCanceled:
		delete(a.Timers, a.key(h.TimerCanceledEventAttributes.StartedEventId))
	/*Cancels*/
	case swf.EventTypeRequestCancelExternalWorkflowExecutionFailed:
		a.incrementCancellationAttempts(h)
		delete(a.Cancellations, a.key(h.RequestCancelExternalWorkflowExecutionFailedEventAttributes.InitiatedEventId))
	case swf.EventTypeExternalWorkflowExecutionCancelRequested:
		key := a.key(h.ExternalWorkflowExecutionCancelRequestedEventAttributes.InitiatedEventId)
		info := a.Cancellations[key]
		delete(a.CancelationAttempts, info.WorkflowId)
		delete(a.Cancellations, key)
	/*Children*/
	case swf.EventTypeStartChildWorkflowExecutionFailed:
		a.incrementChildAttempts(h)
		delete(a.Children, a.key(h.StartChildWorkflowExecutionFailedEventAttributes.InitiatedEventId))
	case swf.EventTypeChildWorkflowExecutionStarted:
		key := a.key(h.ChildWorkflowExecutionStartedEventAttributes.InitiatedEventId)
		info := a.Children[key]
		delete(a.ChildrenAttempts, info.WorkflowId)
		delete(a.Children, key)

	}
}

// ActivityInfo returns the ActivityInfo that is correlates with a given event. The HistoryEvent is expected to be of type EventTypeActivityTaskCompleted,EventTypeActivityTaskFailed,EventTypeActivityTaskTimedOut.
func (a *EventCorrelator) ActivityInfo(h *swf.HistoryEvent) *ActivityInfo {
	a.checkInit()
	return a.Activities[a.getId(h)]
}

// SignalInfo returns the SignalInfo that is correlates with a given event. The HistoryEvent is expected to be of type EventTypeSignalExternalWorkflowExecutionFailed,EventTypeExternalWorkflowExecutionSignaled.
func (a *EventCorrelator) SignalInfo(h *swf.HistoryEvent) *SignalInfo {
	a.checkInit()
	return a.Signals[a.getId(h)]
}

func (a *EventCorrelator) TimerInfo(h *swf.HistoryEvent) *TimerInfo {
	a.checkInit()
	return a.Timers[a.getId(h)]
}

func (a *EventCorrelator) TimerScheduled(timerId string) bool {
	a.checkInit()
	for _, i := range a.Timers {
		if i.TimerId == timerId {
			return true
		}
	}
	return false
}

func (a *EventCorrelator) CancellationInfo(h *swf.HistoryEvent) *CancellationInfo {
	a.checkInit()
	return a.Cancellations[a.getId(h)]
}

func (a *EventCorrelator) ChildInfo(h *swf.HistoryEvent) *ChildInfo {
	a.checkInit()
	return a.Children[a.getId(h)]
}

//AttemptsForActivity returns the number of times a given activity has been attempted.
//It will return 0 if the activity has never failed, has been canceled, or has been completed successfully
func (a *EventCorrelator) AttemptsForActivity(info *ActivityInfo) int {
	a.checkInit()
	if info == nil || info.ActivityId == "" {
		return 0
	}
	return a.ActivityAttempts[info.ActivityId]
}

//AttemptsForSignal returns the number of times a given signal has been attempted.
//It will return 0 if the signal has never failed, or has been completed successfully
func (a *EventCorrelator) AttemptsForSignal(signalInfo *SignalInfo) int {
	a.checkInit()
	if signalInfo == nil {
		return 0
	}
	return a.SignalAttempts[a.signalIdFromInfo(signalInfo)]
}

//AttemptsForCancellation returns the number of times a given signal has been attempted.
//It will return 0 if the signal has never failed, or has been completed successfully
func (a *EventCorrelator) AttemptsForCancellation(info *CancellationInfo) int {
	a.checkInit()
	if info == nil || info.WorkflowId == "" {
		return 0
	}
	return a.CancelationAttempts[info.WorkflowId]
}

//AttemptsForCancellation returns the number of times a given signal has been attempted.
//It will return 0 if the signal has never failed, or has been completed successfully
func (a *EventCorrelator) AttemptsForChild(info *ChildInfo) int {
	a.checkInit()
	if info == nil || info.WorkflowId == "" {
		return 0
	}
	return a.ChildrenAttempts[info.WorkflowId]
}

func (a *EventCorrelator) Attempts(h *swf.HistoryEvent) int {
	switch *h.EventType {
	/*Activities*/
	case swf.EventTypeActivityTaskCanceled, swf.EventTypeActivityTaskCancelRequested, swf.EventTypeActivityTaskTimedOut,
		swf.EventTypeActivityTaskCompleted, swf.EventTypeActivityTaskFailed, swf.EventTypeActivityTaskStarted:
		return a.AttemptsForActivity(a.ActivityInfo(h))
	/*Signals*/
	case swf.EventTypeSignalExternalWorkflowExecutionInitiated, swf.EventTypeSignalExternalWorkflowExecutionFailed,
		swf.EventTypeExternalWorkflowExecutionSignaled:
		return a.AttemptsForSignal(a.SignalInfo(h))
	/*Timers*/
	case swf.EventTypeTimerCanceled, swf.EventTypeTimerFired, swf.EventTypeTimerStarted:
		return 0 //should we count timer fails somehow?
	/*Children*/
	case swf.EventTypeChildWorkflowExecutionStarted, swf.EventTypeStartChildWorkflowExecutionFailed, swf.EventTypeStartChildWorkflowExecutionInitiated:
		return a.AttemptsForChild(a.ChildInfo(h))
	/*Cancels*/
	case swf.EventTypeRequestCancelExternalWorkflowExecutionFailed, swf.EventTypeRequestCancelExternalWorkflowExecutionInitiated,
		swf.EventTypeExternalWorkflowExecutionCancelRequested:
		return a.AttemptsForCancellation(a.CancellationInfo(h))
	}

	return 0
}

func (a *EventCorrelator) checkInit() {
	if a.Activities == nil {
		a.Activities = make(map[string]*ActivityInfo)
	}
	if a.ActivityAttempts == nil {
		a.ActivityAttempts = make(map[string]int)
	}
	if a.Signals == nil {
		a.Signals = make(map[string]*SignalInfo)
	}
	if a.SignalAttempts == nil {
		a.SignalAttempts = make(map[string]int)
	}
	if a.Timers == nil {
		a.Timers = make(map[string]*TimerInfo)
	}
	if a.Cancellations == nil {
		a.Cancellations = make(map[string]*CancellationInfo)
	}
	if a.CancelationAttempts == nil {
		a.CancelationAttempts = make(map[string]int)
	}
	if a.Children == nil {
		a.Children = make(map[string]*ChildInfo)
	}
	if a.ChildrenAttempts == nil {
		a.ChildrenAttempts = make(map[string]int)
	}
}

func (a *EventCorrelator) getId(h *swf.HistoryEvent) (id string) {
	switch *h.EventType {
	/*Activities*/
	case swf.EventTypeActivityTaskScheduled:
		if h.EventId != nil {
			id = a.key(h.EventId)
		}
	case swf.EventTypeActivityTaskCompleted:
		if h.ActivityTaskCompletedEventAttributes != nil {
			id = a.key(h.ActivityTaskCompletedEventAttributes.ScheduledEventId)
		}
	case swf.EventTypeActivityTaskFailed:
		if h.ActivityTaskFailedEventAttributes != nil {
			id = a.key(h.ActivityTaskFailedEventAttributes.ScheduledEventId)
		}
	case swf.EventTypeActivityTaskTimedOut:
		if h.ActivityTaskTimedOutEventAttributes != nil {
			id = a.key(h.ActivityTaskTimedOutEventAttributes.ScheduledEventId)
		}
	case swf.EventTypeActivityTaskCanceled:
		if h.ActivityTaskCanceledEventAttributes != nil {
			id = a.key(h.ActivityTaskCanceledEventAttributes.ScheduledEventId)
		}
	case swf.EventTypeActivityTaskStarted:
		if h.ActivityTaskStartedEventAttributes != nil {
			id = a.key(h.ActivityTaskStartedEventAttributes.ScheduledEventId)
		}
	/*Signals*/
	case swf.EventTypeExternalWorkflowExecutionSignaled:
		if h.ExternalWorkflowExecutionSignaledEventAttributes != nil {
			id = a.key(h.ExternalWorkflowExecutionSignaledEventAttributes.InitiatedEventId)
		}
	case swf.EventTypeSignalExternalWorkflowExecutionInitiated:
		if h.EventId != nil {
			id = a.key(h.EventId)
		}
	case swf.EventTypeSignalExternalWorkflowExecutionFailed:
		if h.SignalExternalWorkflowExecutionFailedEventAttributes != nil {
			id = a.key(h.SignalExternalWorkflowExecutionFailedEventAttributes.InitiatedEventId)
		}
	/*Cancels*/
	case swf.EventTypeRequestCancelExternalWorkflowExecutionInitiated:
		if h.EventId != nil {
			id = a.key(h.EventId)
		}
	case swf.EventTypeRequestCancelExternalWorkflowExecutionFailed:
		if h.RequestCancelExternalWorkflowExecutionFailedEventAttributes != nil {
			id = a.key(h.RequestCancelExternalWorkflowExecutionFailedEventAttributes.InitiatedEventId)
		}
	case swf.EventTypeExternalWorkflowExecutionCancelRequested:
		if h.ExternalWorkflowExecutionCancelRequestedEventAttributes != nil {
			id = a.key(h.ExternalWorkflowExecutionCancelRequestedEventAttributes.InitiatedEventId)
		}
	/*Timers*/
	case swf.EventTypeTimerFired:
		if h.TimerFiredEventAttributes != nil {
			id = a.key(h.TimerFiredEventAttributes.StartedEventId)
		}
	case swf.EventTypeTimerCanceled:
		if h.TimerCanceledEventAttributes != nil {
			id = a.key(h.TimerCanceledEventAttributes.StartedEventId)
		}
	/*Children*/
	case swf.EventTypeChildWorkflowExecutionStarted:
		if h.ChildWorkflowExecutionStartedEventAttributes != nil {
			id = a.key(h.ChildWorkflowExecutionStartedEventAttributes.InitiatedEventId)
		}
	case swf.EventTypeStartChildWorkflowExecutionFailed:
		if h.StartChildWorkflowExecutionFailedEventAttributes != nil {
			id = a.key(h.StartChildWorkflowExecutionFailedEventAttributes.InitiatedEventId)
		}
	case swf.EventTypeStartChildWorkflowExecutionInitiated:
		if h.EventId != nil {
			id = a.key(h.EventId)
		}
	/*Received Signal*/
	case swf.EventTypeWorkflowExecutionSignaled:
		event := h.WorkflowExecutionSignaledEventAttributes
		if event != nil && event.SignalName != nil && event.Input != nil {
			switch *event.SignalName {
			case ActivityStartedSignal, ActivityUpdatedSignal:
				state := new(SerializedActivityState)
				a.Serializer.Deserialize(*event.Input, state)
				for key, info := range a.Activities {
					if info.ActivityId == state.ActivityId {
						id = key
						break
					}
				}
			default:
				id = a.key(event.ExternalInitiatedEventId)
			}
		}
	}

	return
}

func (a *EventCorrelator) safeActivityId(h *swf.HistoryEvent) string {
	info := a.Activities[a.getId(h)]
	if info != nil {
		return info.ActivityId
	}
	return ""
}

func (a *EventCorrelator) safeSignalId(h *swf.HistoryEvent) string {
	info := a.Signals[a.getId(h)]
	if info != nil {
		return a.signalIdFromInfo(info)
	}
	return ""
}

func (a *EventCorrelator) safeCancellationId(h *swf.HistoryEvent) string {
	info := a.Cancellations[a.getId(h)]
	if info != nil {
		return info.WorkflowId
	}
	return ""
}

func (a *EventCorrelator) safeChildId(h *swf.HistoryEvent) string {
	info := a.Children[a.getId(h)]
	if info != nil {
		return info.WorkflowId
	}
	return ""
}

func (a *EventCorrelator) signalIdFromInfo(info *SignalInfo) string {
	return fmt.Sprintf("%s->%s", info.SignalName, info.WorkflowId)
}

func (a *EventCorrelator) incrementActivityAttempts(h *swf.HistoryEvent) {
	id := a.safeActivityId(h)
	if id != "" {
		a.ActivityAttempts[id]++
	}
}

func (a *EventCorrelator) incrementSignalAttempts(h *swf.HistoryEvent) {
	id := a.safeSignalId(h)
	if id != "" {
		a.SignalAttempts[id]++
	}
}

func (a *EventCorrelator) incrementCancellationAttempts(h *swf.HistoryEvent) {
	id := a.safeCancellationId(h)
	if id != "" {
		a.CancelationAttempts[id]++
	}
}

func (a *EventCorrelator) incrementChildAttempts(h *swf.HistoryEvent) {
	id := a.safeChildId(h)
	if id != "" {
		a.ChildrenAttempts[id]++
	}
}

func (a *EventCorrelator) key(eventId *int64) string {
	return strconv.FormatInt(*eventId, 10)
}

func (a *EventCorrelator) nilSafeEq(sv *string, s string) bool {
	if sv == nil {
		return false
	}

	return *sv == s
}
