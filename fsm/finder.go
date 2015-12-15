package fsm

import (
	"fmt"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/swf"
	. "github.com/sclasen/swfsm/log"
	. "github.com/sclasen/swfsm/sugar"
)

const (
	FilterStatusAll          = "ALL"
	FilterStatusOpen         = "OPEN"
	FilterStatusOpenPriority = "OPEN_PRIORITY"
	FilterStatusClosed       = "CLOSED"
)

type Finder interface {
	FindAll(*FindInput) (*FindOutput, error)
	FindLatestByWorkflowID(workflowID string) (*swf.WorkflowExecution, error)
}

func NewFinder(domain string, c ClientSWFOps) Finder {
	return &finder{
		domain: domain,
		c:      c,
	}
}

type FindInput struct {
	MaximumPageSize *int64

	OpenNextPageToken   *string
	ClosedNextPageToken *string

	ReverseOrder *bool

	StatusFilter string

	StartTimeFilter *swf.ExecutionTimeFilter
	CloseTimeFilter *swf.ExecutionTimeFilter // only closed

	ExecutionFilter   *swf.WorkflowExecutionFilter
	TagFilter         *swf.TagFilter
	TypeFilter        *swf.WorkflowTypeFilter
	CloseStatusFilter *swf.CloseStatusFilter // only closed
}

type FindOutput struct {
	ExecutionInfos      []*swf.WorkflowExecutionInfo
	OpenNextPageToken   *string
	ClosedNextPageToken *string
}

type finder struct {
	domain string
	c      ClientSWFOps
}

func (f *finder) FindAll(input *FindInput) (output *FindOutput, err error) {
	if input.StatusFilter == "" {
		if (input.CloseStatusFilter != nil && input.CloseStatusFilter.Status != nil) ||
			(input.CloseTimeFilter != nil &&
				(input.CloseTimeFilter.OldestDate != nil || input.CloseTimeFilter.LatestDate != nil)) {
			input.StatusFilter = FilterStatusClosed
		} else {
			input.StatusFilter = FilterStatusOpenPriority
		}
	}

	if input.StatusFilter != FilterStatusAll &&
		input.StatusFilter != FilterStatusOpen &&
		input.StatusFilter != FilterStatusOpenPriority &&
		input.StatusFilter != FilterStatusClosed {
		return nil, fmt.Errorf("Invalid status filter")
	}

	selectiveFilter := f.mostSelectiveFilters(input)
	output = &FindOutput{}

	if input.StatusFilter == FilterStatusAll || input.StatusFilter == FilterStatusOpen || input.StatusFilter == FilterStatusOpenPriority {
		openInput := &swf.ListOpenWorkflowExecutionsInput{
			Domain:          &f.domain,
			ReverseOrder:    input.ReverseOrder,
			MaximumPageSize: input.MaximumPageSize,
			NextPageToken:   input.OpenNextPageToken,
			StartTimeFilter: selectiveFilter.StartTimeFilter,
			ExecutionFilter: selectiveFilter.ExecutionFilter,
			TagFilter:       selectiveFilter.TagFilter,
			TypeFilter:      selectiveFilter.TypeFilter,
		}

		resp, err := f.c.ListOpenWorkflowExecutions(openInput)
		if err != nil {
			return nil, err
		}

		output.ExecutionInfos = append(output.ExecutionInfos, resp.ExecutionInfos...)
		output.OpenNextPageToken = resp.NextPageToken
	}

	if input.StatusFilter == FilterStatusOpenPriority {
		output.ExecutionInfos = f.applyInputLocally(input, output.ExecutionInfos)
		if len(output.ExecutionInfos) > 0 {
			return output, nil
		}
	}

	if input.StatusFilter == FilterStatusAll || input.StatusFilter == FilterStatusClosed || input.StatusFilter == FilterStatusOpenPriority {
		closedInput := &swf.ListClosedWorkflowExecutionsInput{
			Domain:            &f.domain,
			ReverseOrder:      input.ReverseOrder,
			MaximumPageSize:   input.MaximumPageSize,
			NextPageToken:     input.ClosedNextPageToken,
			StartTimeFilter:   selectiveFilter.StartTimeFilter,
			CloseTimeFilter:   selectiveFilter.CloseTimeFilter,
			ExecutionFilter:   selectiveFilter.ExecutionFilter,
			TagFilter:         selectiveFilter.TagFilter,
			TypeFilter:        selectiveFilter.TypeFilter,
			CloseStatusFilter: selectiveFilter.CloseStatusFilter,
		}

		resp, err := f.c.ListClosedWorkflowExecutions(closedInput)

		if err != nil {
			return nil, err
		}

		output.ExecutionInfos = append(output.ExecutionInfos, resp.ExecutionInfos...)
		output.ClosedNextPageToken = resp.NextPageToken
	}

	output.ExecutionInfos = f.applyInputLocally(input, output.ExecutionInfos)

	return output, nil
}

type listAnyWorkflowExecutionsInput swf.ListClosedWorkflowExecutionsInput

func (f *finder) mostSelectiveFilters(input *FindInput) listAnyWorkflowExecutionsInput {
	filter := &listAnyWorkflowExecutionsInput{}
	f.setMostSelectiveMetadataFilter(input, filter)
	f.setMostSelectiveTimeFilter(input, filter)
	return *filter
}

func (f *finder) setMostSelectiveMetadataFilter(input *FindInput, output *listAnyWorkflowExecutionsInput) {
	if input.ExecutionFilter != nil {
		output.ExecutionFilter = input.ExecutionFilter
		return
	}

	if input.TagFilter != nil {
		output.TagFilter = input.TagFilter
		return
	}

	if input.TypeFilter != nil {
		output.TypeFilter = input.TypeFilter
		return
	}

	if input.CloseStatusFilter != nil {
		output.CloseStatusFilter = input.CloseStatusFilter
		return
	}
}

func (f *finder) setMostSelectiveTimeFilter(input *FindInput, output *listAnyWorkflowExecutionsInput) {
	if input.StartTimeFilter != nil {
		output.StartTimeFilter = input.StartTimeFilter
		return
	}

	if input.CloseTimeFilter != nil {
		output.CloseTimeFilter = input.CloseTimeFilter
		return
	}
}

func (f *finder) applyInputLocally(input *FindInput, infos []*swf.WorkflowExecutionInfo) []*swf.WorkflowExecutionInfo {
	applied := []*swf.WorkflowExecutionInfo{}

	for _, info := range infos {
		if input.ExecutionFilter != nil {
			if *input.ExecutionFilter.WorkflowId != *info.Execution.WorkflowId {
				continue
			}
		}

		if input.TagFilter != nil {
			found := false
			for _, tag := range info.TagList {
				if *input.TagFilter.Tag == *tag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if input.TypeFilter != nil && input.TypeFilter.Name != nil {
			if *input.TypeFilter.Name != *info.WorkflowType.Name {
				continue
			}
			if input.TypeFilter.Version != nil && input.TypeFilter.Version != nil {
				if *input.TypeFilter.Version != *info.WorkflowType.Version {
					continue
				}
			}
		}

		if input.CloseStatusFilter != nil && input.CloseStatusFilter.Status != nil && info.CloseStatus != nil {
			if *input.CloseStatusFilter.Status != *info.CloseStatus {
				continue
			}
		}

		if input.StartTimeFilter != nil {
			if input.StartTimeFilter.OldestDate != nil && (*input.StartTimeFilter.OldestDate).After(*info.StartTimestamp) {
				continue
			}

			if input.StartTimeFilter.LatestDate != nil && (*input.StartTimeFilter.LatestDate).Before(*info.StartTimestamp) {
				continue
			}
		}

		if input.CloseTimeFilter != nil && info.CloseTimestamp != nil {
			if input.CloseTimeFilter.OldestDate != nil && (*input.CloseTimeFilter.OldestDate).After(*info.CloseTimestamp) {
				continue
			}

			if input.CloseTimeFilter.LatestDate != nil && (*input.CloseTimeFilter.LatestDate).Before(*info.CloseTimestamp) {
				continue
			}
		}

		applied = append(applied, info)
	}

	// swf default order is descending, so reverseOrder is ascending
	if input.ReverseOrder != nil && *input.ReverseOrder {
		sort.Sort(sortExecutionInfos(applied))
	} else {
		sort.Sort(sort.Reverse(sortExecutionInfos(applied)))
	}

	if input.MaximumPageSize != nil && *input.MaximumPageSize > 0 && int64(len(applied)) > *input.MaximumPageSize {
		applied = applied[:*input.MaximumPageSize]
	}

	return applied
}

type sortExecutionInfos []*swf.WorkflowExecutionInfo

func (e sortExecutionInfos) Len() int      { return len(e) }
func (e sortExecutionInfos) Swap(i, j int) { e[i], e[j] = e[j], e[i] }
func (e sortExecutionInfos) Less(i, j int) bool {
	return (*e[i].StartTimestamp).Before(*e[j].StartTimestamp)
}

func (f *finder) FindLatestByWorkflowID(workflowID string) (exec *swf.WorkflowExecution, err error) {
	output, err := f.FindAll(&FindInput{
		StatusFilter:    FilterStatusOpenPriority,
		MaximumPageSize: I(1),
		ReverseOrder:    aws.Bool(false), // keep descending default
		StartTimeFilter: &swf.ExecutionTimeFilter{OldestDate: aws.Time(time.Unix(0, 0))},
		ExecutionFilter: &swf.WorkflowExecutionFilter{
			WorkflowId: S(workflowID),
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

	if len(output.ExecutionInfos) == 1 {
		return output.ExecutionInfos[0].Execution, nil
	}
	return nil, nil
}
