package fsm

import (
	"strconv"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/swf"
)

// ActivityCorrelator is a serialization-friendly struct that can be used as a field in your main StateData struct in an FSM.
// You can use it to track the type of a given activity, so you know how to react when an event that signals the
// end or an activity hits your Decider.  This is missing from the SWF api.
// Activities and Signals are string instead of int64 beacuse json.
type EventCorrelator struct {
	Activities       map[string]*ActivityInfo //schedueledEventId -> info
	ActivityAttempts map[string]int           //activityID -> attempts
	Signals          map[string]*SignalInfo   //schedueledEventId -> info
	SignalAttempts   map[string]int           //? workflowID + signalName -> attempts
}

// ActivityInfo holds the ActivityID and ActivityType for an activity
type ActivityInfo struct {
	ActivityID string
	*swf.ActivityType
}

// ActivityInfo holds the ActivityID and ActivityType for an activity
type SignalInfo struct {
	SignalName string
	Input      string
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

	if *h.EventType == swf.EventTypeActivityTaskScheduled {
		a.Activities[a.key(h.EventID)] = &ActivityInfo{
			ActivityID:   *h.ActivityTaskScheduledEventAttributes.ActivityID,
			ActivityType: h.ActivityTaskScheduledEventAttributes.ActivityType,
		}
	}

	if *h.EventType == swf.EventTypeSignalExternalWorkflowExecutionInitiated {
		a.Signals[a.key(h.EventID)] = &SignalInfo{
			SignalName: *h.SignalExternalWorkflowExecutionInitiatedEventAttributes.SignalName,
			Input:      *h.SignalExternalWorkflowExecutionInitiatedEventAttributes.Input,
		}
	}
}

// RemoveCorrelation gcs a mapping of eventId to ActivityType. The HistoryEvent is expected to be of type EventTypeActivityTaskCompleted,EventTypeActivityTaskFailed,EventTypeActivityTaskTimedOut.
func (a *EventCorrelator) RemoveCorrelation(h swf.HistoryEvent) {
	a.checkInit()

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
		a.incrementActivityAttempts(h)
		delete(a.Signals, a.key(h.ExternalWorkflowExecutionSignaledEventAttributes.InitiatedEventID))
	case swf.EventTypeSignalExternalWorkflowExecutionFailed:
		delete(a.SignalAttempts, a.key(h.SignalExternalWorkflowExecutionFailedEventAttributes.InitiatedEventID))
	}
}

// ActivityType returns the ActivityType that is correlates with a given event. The HistoryEvent is expected to be of type EventTypeActivityTaskCompleted,EventTypeActivityTaskFailed,EventTypeActivityTaskTimedOut.
func (a *EventCorrelator) ActivityType(h swf.HistoryEvent) *ActivityInfo {
	a.checkInit()
	return a.Activities[a.getID(h)]
}

//AttemptsForID returns the number of times a given activityID has been attempted.
//It will return 0 if the activity has never failed, has been canceled, or has been completed successfully
func (a *EventCorrelator) AttemptsForActivity(activityID string) int {
	a.checkInit()
	return a.ActivityAttempts[activityID]
}

func (a *EventCorrelator) safeActivityID(h swf.HistoryEvent) string {
	info := a.Activities[a.key(h.EventID)]
	if info != nil {
		return info.ActivityID
	}
	return ""
}

func (a *EventCorrelator) safeSignalID(h swf.HistoryEvent) string {
	info := a.Signals[a.key(h.EventID)]
	if info != nil {
		return info.SignalName
	}
	return ""
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
}

func (a *EventCorrelator) getID(h swf.HistoryEvent) (id string) {
	switch *h.EventType {
	case swf.EventTypeActivityTaskCompleted:
		id = a.key(h.ActivityTaskCompletedEventAttributes.ScheduledEventID)
	case swf.EventTypeActivityTaskFailed:
		id = a.key(h.ActivityTaskFailedEventAttributes.ScheduledEventID)
	case swf.EventTypeActivityTaskTimedOut:
		id = a.key(h.ActivityTaskTimedOutEventAttributes.ScheduledEventID)
	case swf.EventTypeActivityTaskCanceled:
		id = a.key(h.ActivityTaskCanceledEventAttributes.ScheduledEventID)
	}
	return
}

func (a *EventCorrelator) incrementActivityAttempts(h swf.HistoryEvent) {
	id := a.safeActivityID(h)
	if id != "" {
		a.ActivityAttempts[id]++
	}
}

func (a *EventCorrelator) incrementSignalAttempts(h swf.HistoryEvent) {
	id := a.safeActivityID(h)
	if id != "" {
		a.ActivityAttempts[id]++
	}
}

func (a *EventCorrelator) key(eventId aws.LongValue) string {
	return strconv.FormatInt(*eventId, 10)
}
