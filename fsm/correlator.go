package fsm

import (
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/sclasen/swfsm/enums/swf"
)

// EventCorrelator is a serialization-friendly struct that is automatically managed by the FSM machinery
// It tracks signal and activity correlation info, so you know how to react when an event that signals the
// end of an activity or signal  hits your Decider.  This is missing from the SWF api.
// Activities and Signals are string instead of int64 beacuse json.
type EventCorrelator struct {
	Activities          map[string]*ActivityInfo     //schedueledEventId -> info
	ActivityAttempts    map[string]int               //activityID -> attempts
	Signals             map[string]*SignalInfo       //schedueledEventId -> info
	SignalAttempts      map[string]int               //? workflowID + signalName -> attempts
	Timers              map[string]*TimerInfo        //startedEventID -> info
	Cancellations       map[string]*CancellationInfo //schedueledEventId -> info
	CancelationAttempts map[string]int               //? workflowID + signalName -> attempts
	Serializer          StateSerializer
}

// ActivityInfo holds the ActivityID and ActivityType for an activity
type ActivityInfo struct {
	ActivityID string
	*swf.ActivityType
}

// SignalInfo holds the SignalName and Input for an activity
type SignalInfo struct {
	SignalName string
	WorkflowID string
}

//TimerInfo holds the Control data from a Timer
type TimerInfo struct {
	Control string
	TimerID string
}

//CancellationInfo holds the Control data and workflow that was being canceled
type CancellationInfo struct {
	WorkflowID string
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

	if a.nilSafeEq(h.EventType, enums.EventTypeActivityTaskScheduled) {
		a.Activities[a.key(h.EventID)] = &ActivityInfo{
			ActivityID:   *h.ActivityTaskScheduledEventAttributes.ActivityID,
			ActivityType: h.ActivityTaskScheduledEventAttributes.ActivityType,
		}
	}

	if a.nilSafeEq(h.EventType, enums.EventTypeSignalExternalWorkflowExecutionInitiated) {
		a.Signals[a.key(h.EventID)] = &SignalInfo{
			SignalName: *h.SignalExternalWorkflowExecutionInitiatedEventAttributes.SignalName,
			WorkflowID: *h.SignalExternalWorkflowExecutionInitiatedEventAttributes.WorkflowID,
		}
	}

	if a.nilSafeEq(h.EventType, enums.EventTypeRequestCancelExternalWorkflowExecutionInitiated) {
		a.Cancellations[a.key(h.EventID)] = &CancellationInfo{
			WorkflowID: *h.RequestCancelExternalWorkflowExecutionInitiatedEventAttributes.WorkflowID,
		}
	}

	if a.nilSafeEq(h.EventType, enums.EventTypeTimerStarted) {
		control := ""
		if h.TimerStartedEventAttributes.Control != nil {
			control = *h.TimerStartedEventAttributes.Control
		}

		a.Timers[a.key(h.EventID)] = &TimerInfo{
			Control: control,
			TimerID: *h.TimerStartedEventAttributes.TimerID,
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
	case enums.EventTypeActivityTaskCompleted:
		delete(a.ActivityAttempts, a.safeActivityID(h))
		delete(a.Activities, a.key(h.ActivityTaskCompletedEventAttributes.ScheduledEventID))
	case enums.EventTypeActivityTaskFailed:
		a.incrementActivityAttempts(h)
		delete(a.Activities, a.key(h.ActivityTaskFailedEventAttributes.ScheduledEventID))
	case enums.EventTypeActivityTaskTimedOut:
		a.incrementActivityAttempts(h)
		delete(a.Activities, a.key(h.ActivityTaskTimedOutEventAttributes.ScheduledEventID))
	case enums.EventTypeActivityTaskCanceled:
		delete(a.ActivityAttempts, a.safeActivityID(h))
		delete(a.Activities, a.key(h.ActivityTaskCanceledEventAttributes.ScheduledEventID))
	case enums.EventTypeExternalWorkflowExecutionSignaled:
		key := a.key(h.ExternalWorkflowExecutionSignaledEventAttributes.InitiatedEventID)
		info := a.Signals[key]
		delete(a.SignalAttempts, a.signalIDFromInfo(info))
		delete(a.Signals, key)
	case enums.EventTypeSignalExternalWorkflowExecutionFailed:
		a.incrementSignalAttempts(h)
		delete(a.Signals, a.key(h.SignalExternalWorkflowExecutionFailedEventAttributes.InitiatedEventID))
	case enums.EventTypeTimerFired:
		delete(a.Timers, a.key(h.TimerFiredEventAttributes.StartedEventID))
	case enums.EventTypeTimerCanceled:
		delete(a.Timers, a.key(h.TimerCanceledEventAttributes.StartedEventID))
	case enums.EventTypeRequestCancelExternalWorkflowExecutionFailed:
		a.incrementCancellationAttempts(h)
		delete(a.Activities, a.key(h.RequestCancelExternalWorkflowExecutionFailedEventAttributes.InitiatedEventID))
	case enums.EventTypeExternalWorkflowExecutionCancelRequested:
		key := a.key(h.ExternalWorkflowExecutionCancelRequestedEventAttributes.InitiatedEventID)
		info := a.Cancellations[key]
		delete(a.CancelationAttempts, info.WorkflowID)
		delete(a.Cancellations, key)
	}
}

// ActivityInfo returns the ActivityInfo that is correlates with a given event. The HistoryEvent is expected to be of type EventTypeActivityTaskCompleted,EventTypeActivityTaskFailed,EventTypeActivityTaskTimedOut.
func (a *EventCorrelator) ActivityInfo(h *swf.HistoryEvent) *ActivityInfo {
	a.checkInit()
	return a.Activities[a.getID(h)]
}

// SignalInfo returns the SignalInfo that is correlates with a given event. The HistoryEvent is expected to be of type EventTypeSignalExternalWorkflowExecutionFailed,EventTypeExternalWorkflowExecutionSignaled.
func (a *EventCorrelator) SignalInfo(h *swf.HistoryEvent) *SignalInfo {
	a.checkInit()
	return a.Signals[a.getID(h)]
}

func (a *EventCorrelator) TimerInfo(h *swf.HistoryEvent) *TimerInfo {
	a.checkInit()
	return a.Timers[a.getID(h)]
}

func (a *EventCorrelator) CancellationInfo(h *swf.HistoryEvent) *CancellationInfo {
	a.checkInit()
	return a.Cancellations[a.getID(h)]
}

//AttemptsForActivity returns the number of times a given activity has been attempted.
//It will return 0 if the activity has never failed, has been canceled, or has been completed successfully
func (a *EventCorrelator) AttemptsForActivity(info *ActivityInfo) int {
	a.checkInit()
	return a.ActivityAttempts[info.ActivityID]
}

//AttemptsForSignal returns the number of times a given signal has been attempted.
//It will return 0 if the signal has never failed, or has been completed successfully
func (a *EventCorrelator) AttemptsForSignal(signalInfo *SignalInfo) int {
	a.checkInit()
	return a.SignalAttempts[a.signalIDFromInfo(signalInfo)]
}

//AttemptsForCancellation returns the number of times a given signal has been attempted.
//It will return 0 if the signal has never failed, or has been completed successfully
func (a *EventCorrelator) AttemptsForCancellation(info *CancellationInfo) int {
	a.checkInit()
	if info == nil || info.WorkflowID == "" {
		return 0
	}
	return a.CancelationAttempts[info.WorkflowID]
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
}

func (a *EventCorrelator) getID(h *swf.HistoryEvent) (id string) {
	switch *h.EventType {
	case enums.EventTypeActivityTaskCompleted:
		if h.ActivityTaskCompletedEventAttributes != nil {
			id = a.key(h.ActivityTaskCompletedEventAttributes.ScheduledEventID)
		}
	case enums.EventTypeActivityTaskFailed:
		if h.ActivityTaskFailedEventAttributes != nil {
			id = a.key(h.ActivityTaskFailedEventAttributes.ScheduledEventID)
		}
	case enums.EventTypeActivityTaskTimedOut:
		if h.ActivityTaskTimedOutEventAttributes != nil {
			id = a.key(h.ActivityTaskTimedOutEventAttributes.ScheduledEventID)
		}
	case enums.EventTypeActivityTaskCanceled:
		if h.ActivityTaskCanceledEventAttributes != nil {
			id = a.key(h.ActivityTaskCanceledEventAttributes.ScheduledEventID)
		}
	//might want to get info on started too
	case enums.EventTypeActivityTaskStarted:
		if h.ActivityTaskStartedEventAttributes != nil {
			id = a.key(h.ActivityTaskStartedEventAttributes.ScheduledEventID)
		}
	case enums.EventTypeExternalWorkflowExecutionSignaled:
		if h.ExternalWorkflowExecutionSignaledEventAttributes != nil {
			id = a.key(h.ExternalWorkflowExecutionSignaledEventAttributes.InitiatedEventID)
		}
	case enums.EventTypeSignalExternalWorkflowExecutionFailed:
		if h.SignalExternalWorkflowExecutionFailedEventAttributes != nil {
			id = a.key(h.SignalExternalWorkflowExecutionFailedEventAttributes.InitiatedEventID)
		}
	case enums.EventTypeRequestCancelExternalWorkflowExecutionFailed:
		if h.RequestCancelExternalWorkflowExecutionFailedEventAttributes != nil {
			id = a.key(h.RequestCancelExternalWorkflowExecutionFailedEventAttributes.InitiatedEventID)
		}
	case enums.EventTypeExternalWorkflowExecutionCancelRequested:
		if h.ExternalWorkflowExecutionCancelRequestedEventAttributes != nil {
			id = a.key(h.ExternalWorkflowExecutionCancelRequestedEventAttributes.InitiatedEventID)
		}
	case enums.EventTypeTimerFired:
		if h.TimerFiredEventAttributes != nil {
			id = a.key(h.TimerFiredEventAttributes.StartedEventID)
		}
	case enums.EventTypeTimerCanceled:
		if h.TimerCanceledEventAttributes != nil {
			id = a.key(h.TimerCanceledEventAttributes.StartedEventID)
		}
	case enums.EventTypeWorkflowExecutionSignaled:
		event := h.WorkflowExecutionSignaledEventAttributes
		if event != nil && event.SignalName != nil && event.Input != nil {
			switch *event.SignalName {
			case ActivityStartedSignal, ActivityUpdatedSignal:
				state := new(SerializedActivityState)
				a.Serializer.Deserialize(*event.Input, state)
				id = state.ActivityID
			default:
				id = a.key(event.ExternalInitiatedEventID)
			}
		}
	}
	return
}

func (a *EventCorrelator) safeActivityID(h *swf.HistoryEvent) string {
	info := a.Activities[a.getID(h)]
	if info != nil {
		return info.ActivityID
	}
	return ""
}

func (a *EventCorrelator) safeSignalID(h *swf.HistoryEvent) string {
	info := a.Signals[a.getID(h)]
	if info != nil {
		return a.signalIDFromInfo(info)
	}
	return ""
}

func (a *EventCorrelator) safeCancellationID(h *swf.HistoryEvent) string {
	info := a.Cancellations[a.getID(h)]
	if info != nil {
		return info.WorkflowID
	}
	return ""
}

func (a *EventCorrelator) signalIDFromInfo(info *SignalInfo) string {
	return fmt.Sprintf("%s->%s", info.SignalName, info.WorkflowID)
}

func (a *EventCorrelator) incrementActivityAttempts(h *swf.HistoryEvent) {
	id := a.safeActivityID(h)
	if id != "" {
		a.ActivityAttempts[id]++
	}
}

func (a *EventCorrelator) incrementSignalAttempts(h *swf.HistoryEvent) {
	id := a.safeSignalID(h)
	if id != "" {
		a.SignalAttempts[id]++
	}
}

func (a *EventCorrelator) incrementCancellationAttempts(h *swf.HistoryEvent) {
	id := a.safeCancellationID(h)
	if id != "" {
		a.CancelationAttempts[id]++
	}
}

func (a *EventCorrelator) key(eventID *int64) string {
	return strconv.FormatInt(*eventID, 10)
}

func (a *EventCorrelator) nilSafeEq(sv *string, s string) bool {
	if sv == nil {
		return false
	}

	return *sv == s
}
