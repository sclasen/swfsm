package fsm

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/juju/errors"
	. "github.com/sclasen/swfsm/sugar"
)

type FSMClient interface {
	WalkOpenWorkflowInfos(template *swf.ListOpenWorkflowExecutionsInput, fn func(*swf.WorkflowExecutionInfos, bool) bool) error
	GetState(id string) (string, interface{}, error)
	GetStateForRun(workflow, run string) (string, interface{}, error)
	GetSerializedStateForRun(workflow, run string) (*SerializedState, *swf.GetWorkflowExecutionHistoryOutput, error)
	GetSnapshots(id string) ([]FSMSnapshot, error)
	Signal(id string, signal string, input interface{}) error
	Start(startTemplate swf.StartWorkflowExecutionInput, id string, input interface{}) (*swf.StartWorkflowExecutionOutput, error)
	RequestCancel(id string) error
}

type ClientSWFOps interface {
	ListOpenWorkflowExecutionsPages(*swf.ListOpenWorkflowExecutionsInput, func(*swf.WorkflowExecutionInfos, bool) bool) error
	ListClosedWorkflowExecutionsPages(*swf.ListClosedWorkflowExecutionsInput, func(*swf.WorkflowExecutionInfos, bool) bool) error
	GetWorkflowExecutionHistoryPages(*swf.GetWorkflowExecutionHistoryInput, func(*swf.GetWorkflowExecutionHistoryOutput, bool) bool) error
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

func (c *client) WalkOpenWorkflowInfos(template *swf.ListOpenWorkflowExecutionsInput, fn func(*swf.WorkflowExecutionInfos, bool) bool) error {
	template.Domain = S(c.f.Domain)

	if template.StartTimeFilter == nil {
		template.StartTimeFilter = &swf.ExecutionTimeFilter{OldestDate: aws.Time(time.Unix(0, 0))}
	}

	if template.TypeFilter == nil && template.ExecutionFilter == nil && template.TagFilter == nil {
		template.TypeFilter = &swf.WorkflowTypeFilter{Name: S(c.f.Name)}
	}

	return c.c.ListOpenWorkflowExecutionsPages(template, fn)
}

func (c *client) findExecution(id string) (*swf.WorkflowExecution, error) {
	var execution *swf.WorkflowExecution
	var err error

	pageFunc := func(p *swf.WorkflowExecutionInfos, lastPage bool) (shouldContinue bool) {
		if len(p.ExecutionInfos) == 1 {
			execution = p.ExecutionInfos[0].Execution
		}
		return false
	}

	shouldReturn := func() bool {
		if err != nil {
			if ae, ok := err.(awserr.Error); ok {
				log.Printf("component=client fn=findExecution at=list error-type=%s message=%s", ae.Code(), ae.Message())
			} else {
				log.Printf("component=client fn=findExecution at=list error=%s", err)
			}
			return true
		}

		return execution != nil
	}

	err = c.c.ListOpenWorkflowExecutionsPages(&swf.ListOpenWorkflowExecutionsInput{
		Domain:          S(c.f.Domain),
		MaximumPageSize: aws.Int64(1),
		StartTimeFilter: &swf.ExecutionTimeFilter{OldestDate: aws.Time(time.Unix(0, 0))},
		ExecutionFilter: &swf.WorkflowExecutionFilter{
			WorkflowID: S(id),
		},
	}, pageFunc)

	if shouldReturn() {
		return execution, err
	}

	err = c.c.ListClosedWorkflowExecutionsPages(&swf.ListClosedWorkflowExecutionsInput{
		Domain:          S(c.f.Domain),
		MaximumPageSize: aws.Int64(1),
		StartTimeFilter: &swf.ExecutionTimeFilter{OldestDate: aws.Time(time.Unix(0, 0))},
		ExecutionFilter: &swf.WorkflowExecutionFilter{
			WorkflowID: S(id),
		},
	}, pageFunc)

	if shouldReturn() {
		return execution, err
	}

	return nil, errors.Trace(fmt.Errorf("workflow not found for id %s", id))
}

func (c *client) GetSerializedStateForRun(id, run string) (*SerializedState, *swf.GetWorkflowExecutionHistoryOutput, error) {
	var ss *SerializedState
	var latestPage *swf.GetWorkflowExecutionHistoryOutput
	var ssErr error

	hErr := c.c.GetWorkflowExecutionHistoryPages(&swf.GetWorkflowExecutionHistoryInput{
		Domain: S(c.f.Domain),
		Execution: &swf.WorkflowExecution{
			WorkflowID: S(id),
			RunID:      S(run),
		},
		ReverseOrder: aws.Bool(true),
	}, func(p *swf.GetWorkflowExecutionHistoryOutput, lastPage bool) (shouldContinue bool) {
		if latestPage == nil {
			latestPage = p // in reverser order, fo first is latest
		}

		ss, ssErr = c.f.findSerializedState(p.Events)
		return ss == nil
	})

	if hErr != nil {
		if ae, ok := hErr.(awserr.Error); ok {
			log.Printf("component=client fn=GetState at=get-history error-type=%s message=%s", ae.Code(), ae.Message())
		} else {
			log.Printf("component=client fn=GetState at=get-history error=%s", hErr)
		}
		return nil, nil, hErr
	}

	return ss, latestPage, ssErr
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

	var sErr error
	hErr := c.c.GetWorkflowExecutionHistoryPages(&swf.GetWorkflowExecutionHistoryInput{
		Domain:       S(c.f.Domain),
		Execution:    execution,
		ReverseOrder: aws.Bool(true),
	}, func(p *swf.GetWorkflowExecutionHistoryOutput, lastPage bool) (shouldContinue bool) {
		snapshot := FSMSnapshot{}
		for _, historyEvent := range p.Events {
			// TODO: how to deal with failure events?
			switch EventType := *historyEvent.EventType; EventType {
			case swf.EventTypeWorkflowExecutionStarted:
				snapshot.Event = &FSMSnapshotEvent{
					Type:  EventType,
					Name:  "start",
					Input: c.tryDeserialize(historyEvent.WorkflowExecutionStartedEventAttributes.Input),
				}
			case swf.EventTypeWorkflowExecutionSignaled:
				snapshot.Event = &FSMSnapshotEvent{
					Type:  EventType,
					Name:  *historyEvent.WorkflowExecutionSignaledEventAttributes.SignalName,
					Input: c.tryDeserialize(historyEvent.WorkflowExecutionSignaledEventAttributes.Input),
				}
			case swf.EventTypeActivityTaskScheduled:
				if snapshot.Event != nil && snapshot.Event.Input == c.pointerScheduledEventID(historyEvent.EventID) {
					snapshot.Event.Name = *historyEvent.ActivityTaskScheduledEventAttributes.ActivityType.Name
					snapshot.Event.Version = *historyEvent.ActivityTaskScheduledEventAttributes.ActivityType.Version
					snapshot.Event.Input = c.tryDeserialize(historyEvent.ActivityTaskScheduledEventAttributes.Input)
				}
			case swf.EventTypeActivityTaskCompleted:
				snapshot.Event = &FSMSnapshotEvent{
					Type:   EventType,
					Input:  c.pointerScheduledEventID(historyEvent.ActivityTaskCompletedEventAttributes.ScheduledEventID),
					Output: c.tryDeserialize(historyEvent.ActivityTaskCompletedEventAttributes.Result),
				}
			case swf.EventTypeStartChildWorkflowExecutionInitiated:
				snapshot.Event = &FSMSnapshotEvent{
					Type:   EventType,
					Name:   *historyEvent.StartChildWorkflowExecutionInitiatedEventAttributes.WorkflowType.Name,
					Input:  c.tryDeserialize(historyEvent.StartChildWorkflowExecutionInitiatedEventAttributes.Input),
					Target: *historyEvent.StartChildWorkflowExecutionInitiatedEventAttributes.WorkflowID,
				}
			case swf.EventTypeSignalExternalWorkflowExecutionInitiated:
				snapshot.Event = &FSMSnapshotEvent{
					Type:   EventType,
					Name:   *historyEvent.SignalExternalWorkflowExecutionInitiatedEventAttributes.SignalName,
					Input:  c.tryDeserialize(historyEvent.SignalExternalWorkflowExecutionInitiatedEventAttributes.Input),
					Target: *historyEvent.SignalExternalWorkflowExecutionInitiatedEventAttributes.WorkflowID,
				}
			case swf.EventTypeRequestCancelExternalWorkflowExecutionInitiated:
				snapshot.Event = &FSMSnapshotEvent{
					Type:   EventType,
					Target: *historyEvent.RequestCancelExternalWorkflowExecutionInitiatedEventAttributes.WorkflowID,
				}
			case swf.EventTypeTimerStarted:
				if snapshot.Event != nil && snapshot.Event.Input == c.pointerStartedEventID(historyEvent.EventID) {
					snapshot.Event.Input = *historyEvent.TimerStartedEventAttributes.StartToFireTimeout
				}
			case swf.EventTypeTimerFired:
				snapshot.Event = &FSMSnapshotEvent{
					Type:  EventType,
					Name:  *historyEvent.TimerFiredEventAttributes.TimerID,
					Input: c.pointerStartedEventID(historyEvent.TimerFiredEventAttributes.StartedEventID),
				}
			case swf.EventTypeWorkflowExecutionCancelRequested:
				snapshot.Event = &FSMSnapshotEvent{
					Type: EventType,
				}
			}

			state, sErr := c.f.statefulHistoryEventToSerializedState(historyEvent)
			if sErr != nil {
				return false
			}

			if state != nil {
				snapshot.State = &FSMSnapshotState{
					ID:        *historyEvent.EventID,
					Timestamp: *historyEvent.EventTimestamp,
					Version:   state.StateVersion,
					Name:      state.StateName,
					Data:      c.f.zeroStateData(),
				}
				sErr = c.f.Serializer.Deserialize(state.StateData, snapshot.State.Data)
				if sErr != nil {
					return false
				}

				snapshots = append(snapshots, snapshot)
				snapshot = FSMSnapshot{}
			}
		}

		return true
	})

	for _, err := range []error{sErr, hErr} {
		if err != nil {
			return snapshots, nil
		}
	}
	return snapshots, nil
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
