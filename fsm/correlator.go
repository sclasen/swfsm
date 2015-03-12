package fsm

import (
	"fmt"
	"strconv"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/swf"
)

// EventCorrelator is a serialization-friendly struct that is automatically managed by the FSM machinery
// It tracks signal and activity correlation info, so you know how to react when an event that signals the
// end of an activity or signal  hits your Decider.  This is missing from the SWF api.
// Activities and Signals are string instead of int64 beacuse json.
type EventCorrelator struct {
	Activities       map[string]*ActivityInfo //schedueledEventId -> info
	ActivityAttempts map[string]int           //activityID -> attempts
	Signals          map[string]*SignalInfo   //schedueledEventId -> info
	SignalAttempts   map[string]int           //? workflowID + signalName -> attempts
	Timers           map[string]*TimerInfo    //startedEventID -> info
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

// Track will add or remove entries based on the EventType.
// A new entry is added when there is a new ActivityTask, or an entry is removed when the ActivityTask is terminating.
func (a *EventCorrelator) Track(h swf.HistoryEvent) {
	a.RemoveCorrelation(h)
	a.Correlate(h)
}

// Correlate establishes a mapping of eventId to ActivityType. The HistoryEvent is expected to be of type EventTypeActivityTaskScheduled.
func (a *EventCorrelator) Correlate(h swf.HistoryEvent) {
	a.checkInit()

	if a.nilSafeEq(h.EventType, swf.EventTypeActivityTaskScheduled) {

		a.Activities[a.key(h.EventID)] = &ActivityInfo{
			ActivityID:   *h.ActivityTaskScheduledEventAttributes.ActivityID,
			ActivityType: h.ActivityTaskScheduledEventAttributes.ActivityType,
		}
	}

	if a.nilSafeEq(h.EventType, swf.EventTypeSignalExternalWorkflowExecutionInitiated) {
		a.Signals[a.key(h.EventID)] = &SignalInfo{
			SignalName: *h.SignalExternalWorkflowExecutionInitiatedEventAttributes.SignalName,
			WorkflowID: *h.SignalExternalWorkflowExecutionInitiatedEventAttributes.WorkflowID,
		}
	}

	if a.nilSafeEq(h.EventType, swf.EventTypeTimerStarted) {
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
func (a *EventCorrelator) RemoveCorrelation(h swf.HistoryEvent) {
	a.checkInit()
	if h.EventType == nil {
		return
	}
	switch *h.EventType {
	case swf.EventTypeActivityTaskCompleted:
		delete(a.ActivityAttempts, a.safeActivityID(h))
		delete(a.Activities, a.key(h.ActivityTaskCompletedEventAttributes.ScheduledEventID))
	case swf.EventTypeActivityTaskFailed:
		a.incrementActivityAttempts(h)
		delete(a.Activities, a.key(h.ActivityTaskFailedEventAttributes.ScheduledEventID))
	case swf.EventTypeActivityTaskTimedOut:
		a.incrementActivityAttempts(h)
		delete(a.Activities, a.key(h.ActivityTaskTimedOutEventAttributes.ScheduledEventID))
	case swf.EventTypeActivityTaskCanceled:
		delete(a.ActivityAttempts, a.safeActivityID(h))
		delete(a.Activities, a.key(h.ActivityTaskCanceledEventAttributes.ScheduledEventID))
	case swf.EventTypeExternalWorkflowExecutionSignaled:
		key := a.key(h.ExternalWorkflowExecutionSignaledEventAttributes.InitiatedEventID)
		info := a.Signals[key]
		delete(a.SignalAttempts, a.signalIDFromInfo(info))
		delete(a.Signals, key)
	case swf.EventTypeSignalExternalWorkflowExecutionFailed:
		a.incrementSignalAttempts(h)
		delete(a.Signals, a.key(h.SignalExternalWorkflowExecutionFailedEventAttributes.InitiatedEventID))
	case swf.EventTypeTimerFired:
		delete(a.Timers, a.key(h.TimerFiredEventAttributes.StartedEventID))
	case swf.EventTypeTimerCanceled:
		delete(a.Timers, a.key(h.TimerCanceledEventAttributes.StartedEventID))
	}
}

// ActivityInfo returns the ActivityInfo that is correlates with a given event. The HistoryEvent is expected to be of type EventTypeActivityTaskCompleted,EventTypeActivityTaskFailed,EventTypeActivityTaskTimedOut.
func (a *EventCorrelator) ActivityInfo(h swf.HistoryEvent) *ActivityInfo {
	a.checkInit()
	return a.Activities[a.getID(h)]
}

// SignalInfo returns the SignalInfo that is correlates with a given event. The HistoryEvent is expected to be of type EventTypeSignalExternalWorkflowExecutionFailed,EventTypeExternalWorkflowExecutionSignaled.
func (a *EventCorrelator) SignalInfo(h swf.HistoryEvent) *SignalInfo {
	a.checkInit()
	return a.Signals[a.getID(h)]
}

func (a *EventCorrelator) TimerInfo(h swf.HistoryEvent) *TimerInfo {
	a.checkInit()
	return a.Timers[a.getID(h)]
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
}

func (a *EventCorrelator) getID(h swf.HistoryEvent) (id string) {
	switch *h.EventType {
	case swf.EventTypeActivityTaskCompleted:
		if h.ActivityTaskCompletedEventAttributes != nil {
			id = a.key(h.ActivityTaskCompletedEventAttributes.ScheduledEventID)
		}
	case swf.EventTypeActivityTaskFailed:
		if h.ActivityTaskFailedEventAttributes != nil {
			id = a.key(h.ActivityTaskFailedEventAttributes.ScheduledEventID)
		}
	case swf.EventTypeActivityTaskTimedOut:
		if h.ActivityTaskTimedOutEventAttributes != nil {
			id = a.key(h.ActivityTaskTimedOutEventAttributes.ScheduledEventID)
		}
	case swf.EventTypeActivityTaskCanceled:
		if h.ActivityTaskCanceledEventAttributes != nil {
			id = a.key(h.ActivityTaskCanceledEventAttributes.ScheduledEventID)
		}
	case swf.EventTypeExternalWorkflowExecutionSignaled:
		if h.ExternalWorkflowExecutionSignaledEventAttributes != nil {
			id = a.key(h.ExternalWorkflowExecutionSignaledEventAttributes.InitiatedEventID)
		}
	case swf.EventTypeSignalExternalWorkflowExecutionFailed:
		if h.SignalExternalWorkflowExecutionFailedEventAttributes != nil {
			id = a.key(h.SignalExternalWorkflowExecutionFailedEventAttributes.InitiatedEventID)
		}
	case swf.EventTypeTimerFired:
		if h.TimerFiredEventAttributes != nil {
			id = a.key(h.TimerFiredEventAttributes.StartedEventID)
		}
	case swf.EventTypeTimerCanceled:
		if h.TimerCanceledEventAttributes != nil {
			id = a.key(h.TimerCanceledEventAttributes.StartedEventID)
		}

	}
	return
}

func (a *EventCorrelator) safeActivityID(h swf.HistoryEvent) string {
	info := a.Activities[a.getID(h)]
	if info != nil {
		return info.ActivityID
	}
	return ""
}

func (a *EventCorrelator) safeSignalID(h swf.HistoryEvent) string {
	info := a.Signals[a.getID(h)]
	if info != nil {
		return a.signalIDFromInfo(info)
	}
	return ""
}

func (a *EventCorrelator) signalIDFromInfo(info *SignalInfo) string {
	return fmt.Sprintf("%s->%s", info.SignalName, info.WorkflowID)
}

func (a *EventCorrelator) incrementActivityAttempts(h swf.HistoryEvent) {
	id := a.safeActivityID(h)
	if id != "" {
		a.ActivityAttempts[id]++
	}
}

func (a *EventCorrelator) incrementSignalAttempts(h swf.HistoryEvent) {
	id := a.safeSignalID(h)
	if id != "" {
		a.SignalAttempts[id]++
	}
}

func (a *EventCorrelator) key(eventID aws.LongValue) string {
	return strconv.FormatInt(*eventID, 10)
}

func (a *EventCorrelator) nilSafeEq(sv aws.StringValue, s string) bool {
	if sv == nil {
		return false
	}

	return *sv == s
}
