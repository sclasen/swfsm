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
	"github.com/sclasen/swfsm/enums/swf"
	. "github.com/sclasen/swfsm/sugar"
)

type FSMClient interface {
	WalkOpenWorkflowInfos(template *swf.ListOpenWorkflowExecutionsInput, workflowInfosFunc WorkflowInfosFunc) error
	GetState(id string) (string, interface{}, error)
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
		MaximumPageSize: aws.Long(1),
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
			MaximumPageSize: aws.Long(1),
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

func (c *client) GetState(id string) (string, interface{}, error) {
	getState := func() (string, interface{}, error) {
		execution, err := c.findExecution(id)
		if err != nil {
			return "", nil, err
		}

		history, err := c.c.GetWorkflowExecutionHistory(&swf.GetWorkflowExecutionHistoryInput{
			Domain:       S(c.f.Domain),
			Execution:    execution,
			ReverseOrder: aws.Boolean(true),
		})

		if err != nil {
			if ae, ok := err.(awserr.Error); ok {
				log.Printf("component=client fn=GetState at=get-history error-type=%s message=%s", ae.Code(), ae.Message())
			} else {
				log.Printf("component=client fn=GetState at=get-history error=%s", err)
			}
			return "", nil, err
		}

		serialized, err := c.f.findSerializedState(history.Events)

		if err != nil {
			log.Printf("component=client fn=GetState at=find-serialized-state error=%s", err)
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

	var err error
	for i := 0; i < 5; i++ {
		state, data, err := getState()
		if err != nil && strings.HasSuffix(err.Error(), io.EOF.Error()) {
			continue
		} else {
			return state, data, err
		}
	}

	return "", nil, err

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
		ser, err := c.f.Serializer.Serialize(input)
		if err != nil {
			return nil, errors.Trace(err)
		}
		serializedInput = S(ser)
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

	// TODO: extract out history walker and/or nextpagetoken support
	history, err := c.c.GetWorkflowExecutionHistory(&swf.GetWorkflowExecutionHistoryInput{
		Domain:       S(c.f.Domain),
		Execution:    execution,
		ReverseOrder: aws.Boolean(true),
	})

	if err != nil {
		return snapshots, err
	}

	snapshot := FSMSnapshot{}
	for _, historyEvent := range history.Events {
		// TODO: how to deal with failure events?
		switch EventType := *historyEvent.EventType; EventType {
		case enums.EventTypeWorkflowExecutionStarted:
			snapshot.Event = &FSMSnapshotEvent{
				Type:  EventType,
				Name:  "start",
				Input: c.tryDeserialize(historyEvent.WorkflowExecutionStartedEventAttributes.Input),
			}
		case enums.EventTypeWorkflowExecutionSignaled:
			snapshot.Event = &FSMSnapshotEvent{
				Type:  EventType,
				Name:  *historyEvent.WorkflowExecutionSignaledEventAttributes.SignalName,
				Input: c.tryDeserialize(historyEvent.WorkflowExecutionSignaledEventAttributes.Input),
			}
		case enums.EventTypeActivityTaskScheduled:
			if snapshot.Event != nil && snapshot.Event.Input == c.pointerScheduledEventID(historyEvent.EventID) {
				snapshot.Event.Name = *historyEvent.ActivityTaskScheduledEventAttributes.ActivityType.Name
				snapshot.Event.Version = *historyEvent.ActivityTaskScheduledEventAttributes.ActivityType.Version
				snapshot.Event.Input = c.tryDeserialize(historyEvent.ActivityTaskScheduledEventAttributes.Input)
			}
		case enums.EventTypeActivityTaskCompleted:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Input:  c.pointerScheduledEventID(historyEvent.ActivityTaskCompletedEventAttributes.ScheduledEventID),
				Output: c.tryDeserialize(historyEvent.ActivityTaskCompletedEventAttributes.Result),
			}
		case enums.EventTypeStartChildWorkflowExecutionInitiated:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Name:   *historyEvent.StartChildWorkflowExecutionInitiatedEventAttributes.WorkflowType.Name,
				Input:  c.tryDeserialize(historyEvent.StartChildWorkflowExecutionInitiatedEventAttributes.Input),
				Target: *historyEvent.StartChildWorkflowExecutionInitiatedEventAttributes.WorkflowID,
			}
		case enums.EventTypeSignalExternalWorkflowExecutionInitiated:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Name:   *historyEvent.SignalExternalWorkflowExecutionInitiatedEventAttributes.SignalName,
				Input:  c.tryDeserialize(historyEvent.SignalExternalWorkflowExecutionInitiatedEventAttributes.Input),
				Target: *historyEvent.SignalExternalWorkflowExecutionInitiatedEventAttributes.WorkflowID,
			}
		case enums.EventTypeRequestCancelExternalWorkflowExecutionInitiated:
			snapshot.Event = &FSMSnapshotEvent{
				Type:   EventType,
				Target: *historyEvent.RequestCancelExternalWorkflowExecutionInitiatedEventAttributes.WorkflowID,
			}
		case enums.EventTypeTimerStarted:
			if snapshot.Event != nil && snapshot.Event.Input == c.pointerStartedEventID(historyEvent.EventID) {
				snapshot.Event.Input = *historyEvent.TimerStartedEventAttributes.StartToFireTimeout
			}
		case enums.EventTypeTimerFired:
			snapshot.Event = &FSMSnapshotEvent{
				Type:  EventType,
				Name:  *historyEvent.TimerFiredEventAttributes.TimerID,
				Input: c.pointerStartedEventID(historyEvent.TimerFiredEventAttributes.StartedEventID),
			}
		case enums.EventTypeWorkflowExecutionCancelRequested:
			snapshot.Event = &FSMSnapshotEvent{
				Type: EventType,
			}
		}

		state, err := c.f.statefulHistoryEventToSerializedState(historyEvent)
		if err != nil {
			break
		}

		if state != nil {
			snapshot.State = &FSMSnapshotState{
				ID:        *historyEvent.EventID,
				Timestamp: *historyEvent.EventTimestamp,
				Version:   state.StateVersion,
				Name:      state.StateName,
				Data:      c.f.zeroStateData(),
			}
			err = c.f.Serializer.Deserialize(state.StateData, snapshot.State.Data)
			if err != nil {
				return snapshots, err
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
