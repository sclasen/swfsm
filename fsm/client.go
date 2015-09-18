package fsm

import (
	"fmt"
	"log"

	"time"

	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/juju/errors"
	. "github.com/sclasen/swfsm/sugar"
)

type FSMClient interface {
	WalkOpenWorkflowInfos(template *swf.ListOpenWorkflowExecutionsInput, workflowInfosFunc WorkflowInfosFunc) error
	GetState(id string) (string, interface{}, error)
	GetStateForRun(workflow, run string) (string, interface{}, error)
	GetSerializedStateForRun(workflow, run string) (*SerializedState, *swf.GetWorkflowExecutionHistoryOutput, error)
	GetSnapshots(id string) ([]FSMSnapshot, error)
	Signal(id string, signal string, input interface{}) error
	Start(startTemplate swf.StartWorkflowExecutionInput, id string, input interface{}) (*swf.StartWorkflowExecutionOutput, error)
	RequestCancel(id string) error
}

type ClientSWFOps interface {
	ListOpenWorkflowExecutions(req *swf.ListOpenWorkflowExecutionsInput) (resp *swf.WorkflowExecutionInfos, err error)
	ListClosedWorkflowExecutions(req *swf.ListClosedWorkflowExecutionsInput) (resp *swf.WorkflowExecutionInfos, err error)
	GetWorkflowExecutionHistory(req *swf.GetWorkflowExecutionHistoryInput) (resp *swf.GetWorkflowExecutionHistoryOutput, err error)
	SignalWorkflowExecution(req *swf.SignalWorkflowExecutionInput) (resp *swf.SignalWorkflowExecutionOutput, err error)
	StartWorkflowExecution(req *swf.StartWorkflowExecutionInput) (resp *swf.StartWorkflowExecutionOutput, err error)
	TerminateWorkflowExecution(req *swf.TerminateWorkflowExecutionInput) (resp *swf.TerminateWorkflowExecutionOutput, err error)
	RequestCancelWorkflowExecution(req *swf.RequestCancelWorkflowExecutionInput) (resp *swf.RequestCancelWorkflowExecutionOutput, err error)
}

func NewFSMClient(f *FSM, c ClientSWFOps) FSMClient {
	return &client{
		f: f,
		c: c,
	}
}

type client struct {
	f *FSM
	c ClientSWFOps
}

type WorkflowInfosFunc func(infos *swf.WorkflowExecutionInfos) error

type stopWalkingError error

func StopWalking() stopWalkingError {
	return stopWalkingError(fmt.Errorf(""))
}

func IsStopWalking(err error) bool {
	_, ok := err.(stopWalkingError)
	return ok
}

func (c *client) WalkOpenWorkflowInfos(template *swf.ListOpenWorkflowExecutionsInput, workflowInfosFunc WorkflowInfosFunc) error {
	template.Domain = S(c.f.Domain)

	if template.StartTimeFilter == nil {
		template.StartTimeFilter = &swf.ExecutionTimeFilter{OldestDate: aws.Time(time.Unix(0, 0))}
	}

	if template.TypeFilter == nil && template.ExecutionFilter == nil && template.TagFilter == nil {
		template.TypeFilter = &swf.WorkflowTypeFilter{Name: S(c.f.Name)}
	}

	infos, err := c.c.ListOpenWorkflowExecutions(template)

	for {
		if err != nil {
			return err
		}

		err := workflowInfosFunc(infos)

		if err != nil {
			if IsStopWalking(err) {
				return nil
			}
			return err
		}

		if infos.NextPageToken == nil {
			break
		}

		template.NextPageToken = infos.NextPageToken
		infos, err = c.c.ListOpenWorkflowExecutions(template)
	}

	return nil
}

// TODO: make usable again
func (c *client) listIds(executionInfosFunc func() (*swf.WorkflowExecutionInfos, error)) ([]string, string, error) {
	executionInfos, err := executionInfosFunc()

	if err != nil {
		if ae, ok := err.(awserr.Error); ok {
			log.Printf("component=client fn=listIds at=list-infos-func error-type=%s message=%s", ae.Code(), ae.Message())
		} else {
			log.Printf("component=client fn=listIds at=list-infos-func error=%s", err)
		}
		return []string{}, "", err
	}

	ids := []string{}
	for _, info := range executionInfos.ExecutionInfos {
		ids = append(ids, *info.Execution.WorkflowID)
	}

	nextPageToken := ""
	if executionInfos.NextPageToken != nil {
		nextPageToken = *executionInfos.NextPageToken
	}

	return ids, nextPageToken, nil
}

func (c *client) findExecution(id string) (*swf.WorkflowExecution, error) {
	open, err := c.c.ListOpenWorkflowExecutions(&swf.ListOpenWorkflowExecutionsInput{
		Domain:          S(c.f.Domain),
		MaximumPageSize: aws.Int64(1),
		StartTimeFilter: &swf.ExecutionTimeFilter{OldestDate: aws.Time(time.Unix(0, 0))},
		ExecutionFilter: &swf.WorkflowExecutionFilter{
			WorkflowID: S(id),
		},
	})

	if err != nil {
		if ae, ok := err.(awserr.Error); ok {
			log.Printf("component=client fn=findExecution at=list-open error-type=%s message=%s", ae.Code(), ae.Message())
		} else {
			log.Printf("component=client fn=findExecution at=list-open error=%s", err)
		}
		return nil, err
	}

	if len(open.ExecutionInfos) == 1 {
		return open.ExecutionInfos[0].Execution, nil
	} else {
		closed, err := c.c.ListClosedWorkflowExecutions(&swf.ListClosedWorkflowExecutionsInput{
			Domain:          S(c.f.Domain),
			MaximumPageSize: aws.Int64(1),
			StartTimeFilter: &swf.ExecutionTimeFilter{OldestDate: aws.Time(time.Unix(0, 0))},
			ExecutionFilter: &swf.WorkflowExecutionFilter{
				WorkflowID: S(id),
			},
		})

		if err != nil {
			if ae, ok := err.(awserr.Error); ok {
				log.Printf("component=client fn=findExecution at=list-closed error-type=%s message=%s", ae.Code(), ae.Message())
			} else {
				log.Printf("component=client fn=findExecution at=list-closed error=%s", err)
			}
			return nil, err
		}

		if len(closed.ExecutionInfos) > 0 {
			return closed.ExecutionInfos[0].Execution, nil
		} else {
			return nil, errors.Trace(fmt.Errorf("workflow not found for id %s", id))
		}
	}
}

func (c *client) GetSerializedStateForRun(id, run string) (*SerializedState, *swf.GetWorkflowExecutionHistoryOutput, error) {
	getState := func() (*SerializedState, *swf.GetWorkflowExecutionHistoryOutput, error) {

		history, err := c.c.GetWorkflowExecutionHistory(&swf.GetWorkflowExecutionHistoryInput{
			Domain: S(c.f.Domain),
			Execution: &swf.WorkflowExecution{
				WorkflowID: S(id),
				RunID:      S(run),
			},
			ReverseOrder: aws.Bool(true),
		})

		if err != nil {
			if ae, ok := err.(awserr.Error); ok {
				log.Printf("component=client fn=GetState at=get-history error-type=%s message=%s", ae.Code(), ae.Message())
			} else {
				log.Printf("component=client fn=GetState at=get-history error=%s", err)
			}
			return nil, nil, err
		}

		ss, err := c.f.findSerializedState(history.Events)
		return ss, history, err

	}

	var err error
	for i := 0; i < 5; i++ {
		state, history, err := getState()
		if err != nil && strings.HasSuffix(err.Error(), io.EOF.Error()) {
			continue
		} else {
			return state, history, err
		}
	}

	return nil, nil, err
}

func (c *client) GetStateForRun(id, run string) (string, interface{}, error) {
	serialized, _, err := c.GetSerializedStateForRun(id, run)
	if err != nil {
		log.Printf("component=client fn=GetState at=get-serialized-state error=%s", err)
		return "", nil, err
	}
	data := c.f.zeroStateData()
	err = c.f.Serializer.Deserialize(serialized.StateData, data)
	if err != nil {
		log.Printf("component=client fn=GetState at=deserialize-serialized-state error=%s", err)
		return "", nil, err
	}

	return serialized.StateName, data, nil
}

func (c *client) GetState(id string) (string, interface{}, error) {
	execution, err := c.findExecution(id)
	if err != nil {
		return "", nil, err
	}
	return c.GetStateForRun(id, *execution.RunID)
}

func (c *client) Signal(id string, signal string, input interface{}) error {
	var serializedInput *string
	if input != nil {
		switch it := input.(type) {
		case string:
			serializedInput = S(it)
		default:
			ser, err := c.f.Serializer.Serialize(input)
			if err != nil {
				return errors.Trace(err)
			}
			serializedInput = S(ser)
		}
	}
	_, err := c.c.SignalWorkflowExecution(&swf.SignalWorkflowExecutionInput{
		Domain:     S(c.f.Domain),
		SignalName: S(signal),
		Input:      serializedInput,
		WorkflowID: S(id),
	})
	return err
}

func (c *client) Start(startTemplate swf.StartWorkflowExecutionInput, id string, input interface{}) (*swf.StartWorkflowExecutionOutput, error) {
	var serializedInput *string
	if input != nil {
		serializedInput = StartFSMWorkflowInput(c.f, input)
	}
	startTemplate.Domain = S(c.f.Domain)
	startTemplate.WorkflowID = S(id)
	startTemplate.Input = serializedInput
	if len(startTemplate.TagList) == 0 {
		startTemplate.TagList = GetTagsIfTaggable(input)
	}
	return c.c.StartWorkflowExecution(&startTemplate)
}

func (c *client) RequestCancel(id string) error {
	_, err := c.c.RequestCancelWorkflowExecution(&swf.RequestCancelWorkflowExecutionInput{
		Domain:     S(c.f.Domain),
		WorkflowID: S(id),
	})
	return err
}

func (c *client) GetSnapshots(id string) ([]FSMSnapshot, error) {
	snapshots := []FSMSnapshot{}

	execution, err := c.findExecution(id)
	if err != nil {
		return snapshots, err
	}

	req := &swf.GetWorkflowExecutionHistoryInput{
		Domain:       S(c.f.Domain),
		Execution:    execution,
		ReverseOrder: aws.Bool(true),
	}

	history, err := c.c.GetWorkflowExecutionHistory(req)
	if err != nil {
		return snapshots, err
	}

	i := 0
	var iErr error
	snapshots, err = c.snapshotsFromHistoryEventIterator(func() *swf.HistoryEvent {
		if i < len(history.Events) {
			e := history.Events[i]
			i++
			return e
		}

		if history.NextPageToken != nil {
			req.NextPageToken = history.NextPageToken
			history, iErr = c.c.GetWorkflowExecutionHistory(req)
			if iErr != nil {
				return nil
			}

			i = 1
			return history.Events[0]
		}

		return nil
	})

	if iErr != nil {
		return snapshots, iErr
	}

	return snapshots, err
}

func (c *client) snapshotsFromHistoryEventIterator(next func() *swf.HistoryEvent) ([]FSMSnapshot, error) {
	snapshots := []FSMSnapshot{}
	var err error

	snapshot := FSMSnapshot{}
	for event := next(); event != nil; event = next() {

		switch EventType := *event.EventType; EventType {

		// starting
		case swf.EventTypeWorkflowExecutionStarted:
			snapshot.Event = &FSMSnapshotEvent{
				Type:  EventType,
				Name:  "start",
				Input: c.tryDeserialize(event.WorkflowExecutionStartedEventAttributes.Input),
			}

		// signals
		case swf.EventTypeWorkflowExecutionSignaled:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Name:   *event.WorkflowExecutionSignaledEventAttributes.SignalName,
				Source: *event.WorkflowExecutionSignaledEventAttributes.ExternalWorkflowExecution.WorkflowID,
				Input:  c.tryDeserialize(event.WorkflowExecutionSignaledEventAttributes.Input),
			}

		// activities
		case swf.EventTypeActivityTaskScheduled:
			if snapshot.Event != nil && snapshot.Event.Input == c.pointerScheduledEventID(event.EventID) {
				snapshot.Event.Name = *event.ActivityTaskScheduledEventAttributes.ActivityType.Name
				snapshot.Event.Version = *event.ActivityTaskScheduledEventAttributes.ActivityType.Version
				snapshot.Event.Input = c.tryDeserialize(event.ActivityTaskScheduledEventAttributes.Input)
			}
		case swf.EventTypeScheduleActivityTaskFailed:
			snapshot.Event = &FSMSnapshotEvent{
				Type:    EventType,
				Name:    *event.ScheduleActivityTaskFailedEventAttributes.ActivityType.Name,
				Version: *event.ScheduleActivityTaskFailedEventAttributes.ActivityType.Version,
				Target:  *event.ScheduleActivityTaskFailedEventAttributes.ActivityID,
				Output:  *event.ScheduleActivityTaskFailedEventAttributes.Cause,
			}
		case swf.EventTypeActivityTaskCompleted:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Input:  c.pointerScheduledEventID(event.ActivityTaskCompletedEventAttributes.ScheduledEventID),
				Output: c.tryDeserialize(event.ActivityTaskCompletedEventAttributes.Result),
			}
		case swf.EventTypeActivityTaskFailed:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Input:  c.pointerScheduledEventID(event.ActivityTaskFailedEventAttributes.ScheduledEventID),
				Output: c.tryDeserialize(event.ActivityTaskFailedEventAttributes.Details),
			}
		case swf.EventTypeActivityTaskCanceled:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Input:  c.pointerScheduledEventID(event.ActivityTaskCanceledEventAttributes.ScheduledEventID),
				Output: c.tryDeserialize(event.ActivityTaskCanceledEventAttributes.Details),
			}
		case swf.EventTypeActivityTaskTimedOut:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Input:  c.pointerScheduledEventID(event.ActivityTaskTimedOutEventAttributes.ScheduledEventID),
				Output: c.tryDeserialize(event.ActivityTaskTimedOutEventAttributes.Details),
			}

		// children
		case swf.EventTypeStartChildWorkflowExecutionInitiated:
			snapshot.Event = &FSMSnapshotEvent{
				Type:    EventType,
				Name:    *event.StartChildWorkflowExecutionInitiatedEventAttributes.WorkflowType.Name,
				Version: *event.StartChildWorkflowExecutionInitiatedEventAttributes.WorkflowType.Version,
				Input:   c.tryDeserialize(event.StartChildWorkflowExecutionInitiatedEventAttributes.Input),
				Target:  *event.StartChildWorkflowExecutionInitiatedEventAttributes.WorkflowID,
			}
		case swf.EventTypeStartChildWorkflowExecutionFailed:
			snapshot.Event = &FSMSnapshotEvent{
				Type:    EventType,
				Name:    *event.StartChildWorkflowExecutionFailedEventAttributes.WorkflowType.Name,
				Version: *event.StartChildWorkflowExecutionFailedEventAttributes.WorkflowType.Version,
				Target:  *event.StartChildWorkflowExecutionFailedEventAttributes.WorkflowID,
				Output:  *event.StartChildWorkflowExecutionFailedEventAttributes.Cause,
			}
		case swf.EventTypeChildWorkflowExecutionFailed:
			snapshot.Event = &FSMSnapshotEvent{
				Type:    EventType,
				Name:    *event.ChildWorkflowExecutionFailedEventAttributes.WorkflowType.Name,
				Version: *event.ChildWorkflowExecutionFailedEventAttributes.WorkflowType.Version,
				Target:  *event.ChildWorkflowExecutionFailedEventAttributes.WorkflowExecution.WorkflowID,
			}
		case swf.EventTypeChildWorkflowExecutionTimedOut:
			snapshot.Event = &FSMSnapshotEvent{
				Type:    EventType,
				Name:    *event.ChildWorkflowExecutionTimedOutEventAttributes.WorkflowType.Name,
				Version: *event.ChildWorkflowExecutionTimedOutEventAttributes.WorkflowType.Version,
				Output:  *event.ChildWorkflowExecutionTimedOutEventAttributes.TimeoutType,
				Target:  *event.ChildWorkflowExecutionTimedOutEventAttributes.WorkflowExecution.WorkflowID,
			}
		case swf.EventTypeChildWorkflowExecutionCanceled:
			snapshot.Event = &FSMSnapshotEvent{
				Type:    EventType,
				Name:    *event.ChildWorkflowExecutionCanceledEventAttributes.WorkflowType.Name,
				Version: *event.ChildWorkflowExecutionCanceledEventAttributes.WorkflowType.Version,
				Output:  *event.ChildWorkflowExecutionCanceledEventAttributes.Details,
				Target:  *event.ChildWorkflowExecutionCanceledEventAttributes.WorkflowExecution.WorkflowID,
			}
		case swf.EventTypeChildWorkflowExecutionTerminated:
			snapshot.Event = &FSMSnapshotEvent{
				Type:    EventType,
				Name:    *event.ChildWorkflowExecutionTerminatedEventAttributes.WorkflowType.Name,
				Version: *event.ChildWorkflowExecutionTerminatedEventAttributes.WorkflowType.Version,
				Target:  *event.ChildWorkflowExecutionTerminatedEventAttributes.WorkflowExecution.WorkflowID,
			}

		// signal externals
		case swf.EventTypeSignalExternalWorkflowExecutionInitiated:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Name:   *event.SignalExternalWorkflowExecutionInitiatedEventAttributes.SignalName,
				Input:  c.tryDeserialize(event.SignalExternalWorkflowExecutionInitiatedEventAttributes.Input),
				Target: *event.SignalExternalWorkflowExecutionInitiatedEventAttributes.WorkflowID,
			}
		case swf.EventTypeSignalExternalWorkflowExecutionFailed:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Target: *event.SignalExternalWorkflowExecutionFailedEventAttributes.WorkflowID,
				Output: *event.SignalExternalWorkflowExecutionFailedEventAttributes.Cause,
			}

		// cancel externals
		case swf.EventTypeRequestCancelExternalWorkflowExecutionInitiated:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Target: *event.RequestCancelExternalWorkflowExecutionInitiatedEventAttributes.WorkflowID,
			}

		// timers
		case swf.EventTypeTimerStarted:
			if snapshot.Event != nil && snapshot.Event.Input == c.pointerStartedEventID(event.EventID) {
				snapshot.Event.Input = *event.TimerStartedEventAttributes.StartToFireTimeout
			}
		case swf.EventTypeTimerFired:
			snapshot.Event = &FSMSnapshotEvent{
				Type:  EventType,
				Name:  *event.TimerFiredEventAttributes.TimerID,
				Input: c.pointerStartedEventID(event.TimerFiredEventAttributes.StartedEventID),
			}

		// cancellations
		case swf.EventTypeWorkflowExecutionCancelRequested:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Source: *swf.WorkflowExecutionCancelRequestedEventAttributes.ExternalWorkflowExecution.WorkflowID,
				Input:  *swf.WorkflowExecutionCancelRequestedEventAttributes.Cause,
			}
		case swf.EventTypeWorkflowExecutionCanceled:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Output: *swf.WorkflowExecutionCanceledEventAttributes.Details,
			}
		case swf.EventTypeCancelWorkflowExecutionFailed:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Output: *swf.CancelWorkflowExecutionFailedEventAttributes.Cause,
			}

		// completions
		case swf.EventTypeWorkflowExecutionCompleted:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Output: c.tryDeserialize(swf.WorkflowExecutionCompletedEventAttributes.Result),
			}
		case swf.EventTypeCompleteWorkflowExecutionFailed:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Output: swf.CompleteWorkflowExecutionFailedEventAttributes.Cause,
			}

		// failed
		case swf.EventTypeWorkflowExecutionFailed:
			snapshot.Event = &FSMSnapshotEvent{
				Type: EventType,
				Output: *swf.WorkflowExecutionFailedEventAttributes.Reason + ": " +
					swf.WorkflowExecutionFailedEventAttributes.Details,
			}
		case swf.EventTypeFailWorkflowExecutionFailed:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Output: *swf.FailWorkflowExecutionFailedEventAttributes.Cause,
			}

		// time outs
		case swf.EventTypeWorkflowExecutionTimedOut:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Output: *swf.WorkflowExecutionTimedOutEventAttributes.TimeoutType,
			}

		// continuations
		case swf.EventTypeWorkflowExecutionContinuedAsNew:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Input:  c.tryDeserialize(swf.WorkflowExecutionContinuedAsNewEventAttributes.Input),
				Output: *swf.WorkflowExecutionContinuedAsNewEventAttributes.NewExecutionRunID,
			}
		case swf.EventTypeContinueAsNewWorkflowExecutionFailed:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Output: *swf.ContinueAsNewWorkflowExecutionFailedEventAttributes.Cause,
			}

		// terminations
		case swf.EventTypeWorkflowExecutionTerminated:
			snapshot.Event = &FSMSnapshotEvent{
				Type: EventType,
				Output: *swf.WorkflowExecutionTerminatedEventAttributes.Cause + ": " +
					*swf.WorkflowExecutionTerminatedEventAttributes.Reason + ": " +
					*swf.WorkflowExecutionTerminatedEventAttributes.Details,
			}

		}

		state, err := c.f.statefulHistoryEventToSerializedState(event)
		if err != nil {
			break
		}

		if state != nil {
			snapshot.State = &FSMSnapshotState{
				ID:        *event.EventID,
				Timestamp: *event.EventTimestamp,
				Version:   state.StateVersion,
				Name:      state.StateName,
				Data:      c.f.zeroStateData(),
			}
			err = c.f.Serializer.Deserialize(state.StateData, snapshot.State.Data)
			if err != nil {
				break
			}

			snapshots = append(snapshots, snapshot)
			snapshot = FSMSnapshot{}
		}
	}

	return snapshots, err
}

func (c *client) tryDeserialize(serialized *string) interface{} {
	if serialized == nil || *serialized == "" {
		return ""
	}

	tryMap := make(map[string]interface{})
	err := c.f.systemSerializer.Deserialize(*serialized, &tryMap)
	if err == nil {
		return tryMap
	}
	log.Printf("component=client fn=trySystemDeserialize at=deserialize-map error=%s", err)

	return *serialized
}

func (c *client) pointerStartedEventID(id *int64) string {
	return fmt.Sprintf("StartedEventID->%d", *id)
}

func (c *client) pointerScheduledEventID(id *int64) string {
	return fmt.Sprintf("ScheduledEventID->%d", *id)
}
