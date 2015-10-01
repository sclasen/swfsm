package fsm

import (
	"fmt"

	"time"

	"io"
	"strings"

	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/juju/errors"
	"github.com/sclasen/swfsm/awsinternal/jsonutil"
	. "github.com/sclasen/swfsm/log"
	. "github.com/sclasen/swfsm/sugar"
)

type FSMClient interface {
	WalkOpenWorkflowInfos(template *swf.ListOpenWorkflowExecutionsInput, workflowInfosFunc WorkflowInfosFunc) error
	GetState(id string) (string, interface{}, error)
	GetStateForRun(workflow, run string) (string, interface{}, error)
	GetSerializedStateForRun(workflow, run string) (*SerializedState, *swf.GetWorkflowExecutionHistoryOutput, error)
	Signal(id string, signal string, input interface{}) error
	Start(startTemplate swf.StartWorkflowExecutionInput, id string, input interface{}) (*swf.StartWorkflowExecutionOutput, error)
	RequestCancel(id string) error
	GetHistoryEventIteratorFromWorkflowId(workflowId string) (HistoryEventIterator, error)
	GetHistoryEventIteratorFromWorkflowExecution(execution *swf.WorkflowExecution) (HistoryEventIterator, error)
	GetHistoryEventIteratorFromReader(reader io.Reader) (HistoryEventIterator, error)
	NewSnapshotter() Snapshotter

	// DEPRECATED
	// TODO: remove after clients have stopped using this
	GetSnapshots(id string) ([]FSMSnapshot, error)
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

type HistoryEventIterator func() (*swf.HistoryEvent, error)

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
			Log.Printf("component=client fn=listIds at=list-infos-func error-type=%s message=%s", ae.Code(), ae.Message())
		} else {
			Log.Printf("component=client fn=listIds at=list-infos-func error=%s", err)
		}
		return []string{}, "", err
	}

	ids := []string{}
	for _, info := range executionInfos.ExecutionInfos {
		ids = append(ids, *info.Execution.WorkflowId)
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
			WorkflowId: S(id),
		},
	})

	if err != nil {
		if ae, ok := err.(awserr.Error); ok {
			Log.Printf("component=client fn=findExecution at=list-open error-type=%s message=%s", ae.Code(), ae.Message())
		} else {
			Log.Printf("component=client fn=findExecution at=list-open error=%s", err)
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
				WorkflowId: S(id),
			},
		})

		if err != nil {
			if ae, ok := err.(awserr.Error); ok {
				Log.Printf("component=client fn=findExecution at=list-closed error-type=%s message=%s", ae.Code(), ae.Message())
			} else {
				Log.Printf("component=client fn=findExecution at=list-closed error=%s", err)
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
				WorkflowId: S(id),
				RunId:      S(run),
			},
			ReverseOrder: aws.Bool(true),
		})

		if err != nil {
			if ae, ok := err.(awserr.Error); ok {
				Log.Printf("component=client fn=GetState at=get-history error-type=%s message=%s", ae.Code(), ae.Message())
			} else {
				Log.Printf("component=client fn=GetState at=get-history error=%s", err)
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
		Log.Printf("component=client fn=GetState at=get-serialized-state error=%s", err)
		return "", nil, err
	}
	data := c.f.zeroStateData()
	err = c.f.Serializer.Deserialize(serialized.StateData, data)
	if err != nil {
		Log.Printf("component=client fn=GetState at=deserialize-serialized-state error=%s", err)
		return "", nil, err
	}

	return serialized.StateName, data, nil
}

func (c *client) GetState(id string) (string, interface{}, error) {
	execution, err := c.findExecution(id)
	if err != nil {
		return "", nil, err
	}
	return c.GetStateForRun(id, *execution.RunId)
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
		WorkflowId: S(id),
	})
	return err
}

func (c *client) Start(startTemplate swf.StartWorkflowExecutionInput, id string, input interface{}) (*swf.StartWorkflowExecutionOutput, error) {
	var serializedInput *string
	if input != nil {
		serializedInput = StartFSMWorkflowInput(c.f, input)
	}
	startTemplate.Domain = S(c.f.Domain)
	startTemplate.WorkflowId = S(id)
	startTemplate.Input = serializedInput
	if len(startTemplate.TagList) == 0 {
		startTemplate.TagList = GetTagsIfTaggable(input)
	}
	return c.c.StartWorkflowExecution(&startTemplate)
}

func (c *client) RequestCancel(id string) error {
	_, err := c.c.RequestCancelWorkflowExecution(&swf.RequestCancelWorkflowExecutionInput{
		Domain:     S(c.f.Domain),
		WorkflowId: S(id),
	})
	return err
}

func (c *client) GetHistoryEventIteratorFromWorkflowId(workflowId string) (HistoryEventIterator, error) {
	execution, err := c.findExecution(workflowId)
	if err != nil {
		return nil, err
	}
	return c.GetHistoryEventIteratorFromWorkflowExecution(execution)
}

func (c *client) GetHistoryEventIteratorFromWorkflowExecution(execution *swf.WorkflowExecution) (HistoryEventIterator, error) {
	req := &swf.GetWorkflowExecutionHistoryInput{
		Domain:       S(c.f.Domain),
		Execution:    execution,
		ReverseOrder: aws.Bool(true),
	}

	history, err := c.c.GetWorkflowExecutionHistory(req)
	if err != nil {
		return nil, err
	}

	i := 0
	return func() (*swf.HistoryEvent, error) {
		if i < len(history.Events) {
			e := history.Events[i]
			i++
			return e, nil
		}

		if history.NextPageToken != nil {
			req.NextPageToken = history.NextPageToken
			history, err = c.c.GetWorkflowExecutionHistory(req)
			if err != nil {
				return nil, err
			}

			i = 1
			return history.Events[0], nil
		}

		return nil, nil
	}, nil
}

type sortEventsDescending []*swf.HistoryEvent

func (es sortEventsDescending) Len() int           { return len(es) }
func (es sortEventsDescending) Swap(i, j int)      { es[i], es[j] = es[j], es[i] }
func (es sortEventsDescending) Less(i, j int) bool { return *es[i].EventId > *es[j].EventId }

func (c *client) GetHistoryEventIteratorFromReader(reader io.Reader) (HistoryEventIterator, error) {
	history := swf.GetWorkflowExecutionHistoryOutput{}
	err := jsonutil.UnmarshalJSON(&history, reader)
	if err != nil {
		return nil, err
	}

	sort.Sort(sortEventsDescending(history.Events))

	i := 0
	return func() (*swf.HistoryEvent, error) {
		if i < len(history.Events) {
			e := history.Events[i]
			i++
			return e, nil
		}
		return nil, nil
	}, nil
}

func (c *client) NewSnapshotter() Snapshotter {
	return newSnapshotter(c)
}

// DEPRECATED
// TODO: remove after clients have stopped using this
func (c *client) GetSnapshots(id string) ([]FSMSnapshot, error) {
	return c.NewSnapshotter().FromWorkflowId(id)
}
