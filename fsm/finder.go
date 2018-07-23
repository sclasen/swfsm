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
	FilterStatusAll                  = "ALL"                    // open + closed
	FilterStatusOpen                 = "OPEN"                   // open only
	FilterStatusOpenPriority         = "OPEN_PRIORITY"          // open (+ closed, only if open is totally empty)
	FilterStatusOpenPriorityWorkflow = "OPEN_PRIORITY_WORKFLOW" // open (+ closed, only if open not present workflow-by-workflow)
	FilterStatusClosed               = "CLOSED"                 // closed only
)

type Finder interface {
	FindAll(*FindInput) (*FindOutput, error)
	FindLatestByWorkflowID(workflowID string) (*swf.WorkflowExecution, error)
	Reset()
}

func NewFinder(domain string, c ClientSWFOps) Finder {
	return &finder{
		domain:          domain,
		c:               c,
		workflowIdIndex: make(map[string]struct{}),
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
	domain          string
	c               ClientSWFOps
	workflowIdIndex map[string]struct{}
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

	if !stringsContain([]string{
		FilterStatusAll,
		FilterStatusOpen,
		FilterStatusOpenPriority,
		FilterStatusOpenPriorityWorkflow,
		FilterStatusClosed,
	},
		input.StatusFilter) {
		return nil, fmt.Errorf("Invalid status filter")
	}

	selectiveFilter := f.mostSelectiveFilters(input)
	output = &FindOutput{}

	if input.OpenNextPageToken != nil || (input.OpenNextPageToken == nil && input.ClosedNextPageToken == nil) {
		if stringsContain([]string{
			FilterStatusAll,
			FilterStatusOpen,
			FilterStatusOpenPriority,
			FilterStatusOpenPriorityWorkflow,
		}, input.StatusFilter) {
			openInput := &swf.ListOpenWorkflowExecutionsInput{
				Domain:          &f.domain,
				ReverseOrder:    input.ReverseOrder,
				MaximumPageSize: input.MaximumPageSize,
				NextPageToken:   input.OpenNextPageToken,
				StartTimeFilter: input.StartTimeFilter,
				ExecutionFilter: selectiveFilter.ExecutionFilter,
				TagFilter:       selectiveFilter.TagFilter,
				TypeFilter:      selectiveFilter.TypeFilter,
			}

			resp, err := f.c.ListOpenWorkflowExecutions(openInput)
			if err != nil {
				return nil, err
			}

			f.append(input, output, resp.ExecutionInfos)
			output.OpenNextPageToken = resp.NextPageToken
		}
	}

	if input.StatusFilter == FilterStatusOpenPriority {
		output.ExecutionInfos = f.applyInputLocally(input, output.ExecutionInfos)
		if len(output.ExecutionInfos) > 0 {
			return output, nil
		}
	}

	if input.ClosedNextPageToken != nil || (input.OpenNextPageToken == nil && input.ClosedNextPageToken == nil) {
		if stringsContain([]string{
			FilterStatusAll,
			FilterStatusClosed,
			FilterStatusOpenPriority,
			FilterStatusOpenPriorityWorkflow,
		}, input.StatusFilter) {
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

			f.append(input, output, resp.ExecutionInfos)
			output.ClosedNextPageToken = resp.NextPageToken
		}
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

// result only used for for closed list. open list uses input.StartTimeFilter directly
func (f *finder) setMostSelectiveTimeFilter(input *FindInput, output *listAnyWorkflowExecutionsInput) {
	if input.CloseTimeFilter != nil {
		output.CloseTimeFilter = input.CloseTimeFilter
		return
	}

	if input.StartTimeFilter != nil {
		output.StartTimeFilter = input.StartTimeFilter
		return
	}
}

func (f *finder) append(input *FindInput, output *FindOutput, infos []*swf.WorkflowExecutionInfo) {
	for _, info := range infos {
		if input.StatusFilter == FilterStatusOpenPriorityWorkflow {
			if _, ok := f.workflowIdIndex[*info.Execution.WorkflowId]; ok {
				continue
			}
			f.workflowIdIndex[*info.Execution.WorkflowId] = struct{}{}
		}

		output.ExecutionInfos = append(output.ExecutionInfos, info)
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
			if input.TypeFilter.Version != nil {
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
			Log.Printf("component=client fn=findExecution at=list-open error=%q", err)
		}
		return nil, err
	}

	if len(output.ExecutionInfos) == 1 {
		return output.ExecutionInfos[0].Execution, nil
	}
	return nil, nil
}

func (f *finder) Reset() {
	f.workflowIdIndex = make(map[string]struct{})
}
