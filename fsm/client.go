package fsm

import (
	"fmt"

	"io"
	"strings"

	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/private/protocol/json/jsonutil"
	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/juju/errors"
	. "github.com/sclasen/swfsm/log"
	. "github.com/sclasen/swfsm/sugar"
)

type FSMClient interface {
	GetState(id string) (string, interface{}, error)
	GetStateForRun(workflow, run string) (string, interface{}, error)
	GetSerializedStateForRun(workflow, run string) (*SerializedState, *swf.GetWorkflowExecutionHistoryOutput, error)
	Signal(id string, signal string, input interface{}) error
	Start(startTemplate swf.StartWorkflowExecutionInput, id string, input interface{}) (*swf.StartWorkflowExecutionOutput, error)
	RequestCancel(id string) error
	GetWorkflowExecutionHistoryPages(execution *swf.WorkflowExecution, fn func(p *swf.GetWorkflowExecutionHistoryOutput, lastPage bool) (shouldContinue bool)) error
	GetWorkflowExecutionHistoryFromReader(reader io.Reader) (*swf.GetWorkflowExecutionHistoryOutput, error)
	FindAll(input *FindInput) (output *FindOutput, err error)
	FindAllWalk(input *FindInput, fn func(info *swf.WorkflowExecutionInfo, done bool) (cont bool)) (err error)
	FindLatestByWorkflowID(workflowID string) (exec *swf.WorkflowExecution, err error)
	NewHistorySegmentor() HistorySegmentor
}

type ClientSWFOps interface {
	ListOpenWorkflowExecutions(req *swf.ListOpenWorkflowExecutionsInput) (resp *swf.WorkflowExecutionInfos, err error)
	ListClosedWorkflowExecutions(req *swf.ListClosedWorkflowExecutionsInput) (resp *swf.WorkflowExecutionInfos, err error)
	GetWorkflowExecutionHistory(req *swf.GetWorkflowExecutionHistoryInput) (resp *swf.GetWorkflowExecutionHistoryOutput, err error)
	GetWorkflowExecutionHistoryPages(input *swf.GetWorkflowExecutionHistoryInput, fn func(p *swf.GetWorkflowExecutionHistoryOutput, lastPage bool) (shouldContinue bool)) error
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
				Log.Printf("component=client fn=GetState at=get-history error=%q", err)
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
		Log.Printf("component=client fn=GetState at=get-serialized-state error=%q", err)
		return "", nil, err
	}
	data := c.f.zeroStateData()
	err = c.f.Serializer.Deserialize(serialized.StateData, data)
	if err != nil {
		Log.Printf("component=client fn=GetState at=deserialize-serialized-state error=%q", err)
		return "", nil, err
	}

	return serialized.StateName, data, nil
}

func (c *client) GetState(id string) (string, interface{}, error) {
	execution, err := c.FindLatestByWorkflowID(id)
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

func (c *client) GetWorkflowExecutionHistoryPages(execution *swf.WorkflowExecution, fn func(p *swf.GetWorkflowExecutionHistoryOutput, lastPage bool) (shouldContinue bool)) error {
	req := &swf.GetWorkflowExecutionHistoryInput{
		Domain:       S(c.f.Domain),
		Execution:    execution,
		ReverseOrder: aws.Bool(true),
	}

	return c.c.GetWorkflowExecutionHistoryPages(req, fn)
}

type sortHistoryEvents []*swf.HistoryEvent

func (es sortHistoryEvents) Len() int           { return len(es) }
func (es sortHistoryEvents) Swap(i, j int)      { es[i], es[j] = es[j], es[i] }
func (es sortHistoryEvents) Less(i, j int) bool { return *es[i].EventId < *es[j].EventId }

func (c *client) GetWorkflowExecutionHistoryFromReader(reader io.Reader) (*swf.GetWorkflowExecutionHistoryOutput, error) {
	history := &swf.GetWorkflowExecutionHistoryOutput{}
	err := jsonutil.UnmarshalJSON(history, reader)
	if err != nil {
		return nil, err
	}

	sort.Sort(sort.Reverse(sortHistoryEvents(history.Events)))
	return history, nil
}

func (c *client) NewHistorySegmentor() HistorySegmentor {
	return NewHistorySegmentor(c)
}

func (c *client) FindAll(input *FindInput) (output *FindOutput, err error) {
	return NewFinder(c.f.Domain, c.c).FindAll(input)
}

func (c *client) FindAllWalk(input *FindInput, fn func(info *swf.WorkflowExecutionInfo, done bool) (cont bool)) error {
	f := NewFinder(c.f.Domain, c.c)

	var done bool
	for !done {
		output, err := f.FindAll(input)
		if err != nil {
			return err
		}

		hasNextPage := output.OpenNextPageToken != nil || output.ClosedNextPageToken != nil

		for i, info := range output.ExecutionInfos {
			done = !hasNextPage && i+1 == len(output.ExecutionInfos) // done because paging+iteration
			done = !fn(info, done) || done                           // done because fn
			if done {
				break
			}
		}

		input.OpenNextPageToken = output.OpenNextPageToken
		input.ClosedNextPageToken = output.ClosedNextPageToken
		done = done || !hasNextPage // done because paging (didn't iterate)
	}

	return nil
}

func (c *client) FindLatestByWorkflowID(workflowID string) (exec *swf.WorkflowExecution, err error) {
	ex, err := NewFinder(c.f.Domain, c.c).FindLatestByWorkflowID(workflowID)
	if err == nil && ex == nil {
		return nil, errors.Trace(fmt.Errorf("workflow not found for id %s", workflowID))
	}
	return ex, err
}
