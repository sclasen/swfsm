package fsm

import (
	"fmt"
	"log"

	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/swf"
	"github.com/juju/errors"
	. "github.com/sclasen/swfsm/sugar"
)

type FSMClient interface {
	ListOpenIds() ([]string, string, error)
	ListNextOpenIds(previousPageToken string) ([]string, string, error)
	ListClosedIds() ([]string, string, error)
	ListNextClosedIds(previousPageToken string) ([]string, string, error)
	GetState(id string) (string, interface{}, error)
	Signal(id string, signal string, input interface{}) error
	Start(startTemplate swf.StartWorkflowExecutionInput, id string, input interface{}) (*swf.Run, error)
}

type ClientSWFOps interface {
	ListOpenWorkflowExecutions(req *swf.ListOpenWorkflowExecutionsInput) (resp *swf.WorkflowExecutionInfos, err error)
	ListClosedWorkflowExecutions(req *swf.ListClosedWorkflowExecutionsInput) (resp *swf.WorkflowExecutionInfos, err error)
	GetWorkflowExecutionHistory(req *swf.GetWorkflowExecutionHistoryInput) (resp *swf.History, err error)
	SignalWorkflowExecution(req *swf.SignalWorkflowExecutionInput) (err error)
	StartWorkflowExecution(req *swf.StartWorkflowExecutionInput) (resp *swf.Run, err error)
	TerminateWorkflowExecution(req *swf.TerminateWorkflowExecutionInput) (err error)
	RequestCancelWorkflowExecution(req *swf.RequestCancelWorkflowExecutionInput) (err error)
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

func (c *client) ListOpenIds() ([]string, string, error) {
	return c.listIds(func() (*swf.WorkflowExecutionInfos, error) {
		return c.c.ListOpenWorkflowExecutions(&swf.ListOpenWorkflowExecutionsInput{
			Domain:          S(c.f.Domain),
			StartTimeFilter: &swf.ExecutionTimeFilter{OldestDate: &aws.UnixTimestamp{time.Unix(0, 0)}},
			TypeFilter:      &swf.WorkflowTypeFilter{Name: S(c.f.Name)},
		})
	})
}

func (c *client) ListNextOpenIds(previousPageToken string) ([]string, string, error) {
	return c.listIds(func() (*swf.WorkflowExecutionInfos, error) {
		return c.c.ListOpenWorkflowExecutions(&swf.ListOpenWorkflowExecutionsInput{
			Domain:          S(c.f.Domain),
			StartTimeFilter: &swf.ExecutionTimeFilter{OldestDate: &aws.UnixTimestamp{time.Unix(0, 0)}},
			TypeFilter:      &swf.WorkflowTypeFilter{Name: S(c.f.Name)},
			NextPageToken:   S(previousPageToken),
		})
	})
}

func (c *client) ListClosedIds() ([]string, string, error) {
	return c.listIds(func() (*swf.WorkflowExecutionInfos, error) {
		return c.c.ListClosedWorkflowExecutions(&swf.ListClosedWorkflowExecutionsInput{
			Domain:          S(c.f.Domain),
			StartTimeFilter: &swf.ExecutionTimeFilter{OldestDate: &aws.UnixTimestamp{time.Unix(0, 0)}},
			TypeFilter:      &swf.WorkflowTypeFilter{Name: S(c.f.Name)},
		})
	})
}

func (c *client) ListNextClosedIds(previousPageToken string) ([]string, string, error) {
	return c.listIds(func() (*swf.WorkflowExecutionInfos, error) {
		return c.c.ListClosedWorkflowExecutions(&swf.ListClosedWorkflowExecutionsInput{
			Domain:          S(c.f.Domain),
			StartTimeFilter: &swf.ExecutionTimeFilter{OldestDate: &aws.UnixTimestamp{time.Unix(0, 0)}},
			TypeFilter:      &swf.WorkflowTypeFilter{Name: S(c.f.Name)},
			NextPageToken:   S(previousPageToken),
		})
	})
}

func (c *client) listIds(executionInfosFunc func() (*swf.WorkflowExecutionInfos, error)) ([]string, string, error) {
	executionInfos, err := executionInfosFunc()

	if err != nil {
		if ae, ok := err.(aws.APIError); ok {
			log.Printf("component=client fn=listIds at=list-infos-func error-type=%s message=%s", ae.Type, ae.Message)
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

func (c *client) GetState(id string) (string, interface{}, error) {
	var execution *swf.WorkflowExecution
	open, err := c.c.ListOpenWorkflowExecutions(&swf.ListOpenWorkflowExecutionsInput{
		Domain:          S(c.f.Domain),
		MaximumPageSize: aws.Integer(1),
		StartTimeFilter: &swf.ExecutionTimeFilter{OldestDate: &aws.UnixTimestamp{time.Unix(0, 0)}},
		ExecutionFilter: &swf.WorkflowExecutionFilter{
			WorkflowID: S(id),
		},
	})

	if err != nil {
		if ae, ok := err.(aws.APIError); ok {
			log.Printf("component=client fn=GetState at=list-open error-type=%s message=%s", ae.Type, ae.Message)
		} else {
			log.Printf("component=client fn=GetState at=list-open error=%s", err)
		}
		return "", nil, err
	}

	if len(open.ExecutionInfos) == 1 {
		execution = open.ExecutionInfos[0].Execution
	} else {
		closed, err := c.c.ListClosedWorkflowExecutions(&swf.ListClosedWorkflowExecutionsInput{
			Domain:          S(c.f.Domain),
			MaximumPageSize: aws.Integer(1),
			StartTimeFilter: &swf.ExecutionTimeFilter{OldestDate: &aws.UnixTimestamp{time.Unix(0, 0)}},
			ExecutionFilter: &swf.WorkflowExecutionFilter{
				WorkflowID: S(id),
			},
		})

		if err != nil {
			if ae, ok := err.(aws.APIError); ok {
				log.Printf("component=client fn=GetState at=list-closed error-type=%s message=%s", ae.Type, ae.Message)
			} else {
				log.Printf("component=client fn=GetState at=list-closed error=%s", err)
			}
			return "", nil, err
		}

		if len(closed.ExecutionInfos) > 0 {
			execution = closed.ExecutionInfos[0].Execution
		} else {
			return "", nil, errors.Trace(fmt.Errorf("workflow not found for id %s", id))
		}
	}

	history, err := c.c.GetWorkflowExecutionHistory(&swf.GetWorkflowExecutionHistoryInput{
		Domain:       S(c.f.Domain),
		Execution:    execution,
		ReverseOrder: aws.True(),
	})

	if err != nil {
		if ae, ok := err.(aws.APIError); ok {
			log.Printf("component=client fn=GetState at=get-history error-type=%s message=%s", ae.Type, ae.Message)
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

func (c *client) Signal(id string, signal string, input interface{}) error {
	var serializedInput aws.StringValue
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
	return c.c.SignalWorkflowExecution(&swf.SignalWorkflowExecutionInput{
		Domain:     S(c.f.Domain),
		SignalName: S(signal),
		Input:      serializedInput,
		WorkflowID: S(id),
	})
}

func (c *client) Start(startTemplate swf.StartWorkflowExecutionInput, id string, input interface{}) (*swf.Run, error) {
	var serializedInput aws.StringValue
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
	return c.c.StartWorkflowExecution(&startTemplate)
}
