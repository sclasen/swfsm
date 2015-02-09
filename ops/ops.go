package ops

import . "github.com/awslabs/aws-sdk-go/gen/swf"

/*SWFOps is an interface extracted from the swf.SWF client, to enable easier mocking. it will be subsumed if https://github.com/awslabs/aws-sdk-go/pull/79
gets merged*/
type SWFOperations interface {
	CountClosedWorkflowExecutions(req *CountClosedWorkflowExecutionsInput) (resp *WorkflowExecutionCount, err error)
	CountOpenWorkflowExecutions(req *CountOpenWorkflowExecutionsInput) (resp *WorkflowExecutionCount, err error)
	CountPendingActivityTasks(req *CountPendingActivityTasksInput) (resp *PendingTaskCount, err error)
	CountPendingDecisionTasks(req *CountPendingDecisionTasksInput) (resp *PendingTaskCount, err error)
	DeprecateActivityType(req *DeprecateActivityTypeInput) (err error)
	DeprecateDomain(req *DeprecateDomainInput) (err error)
	DeprecateWorkflowType(req *DeprecateWorkflowTypeInput) (err error)
	DescribeActivityType(req *DescribeActivityTypeInput) (resp *ActivityTypeDetail, err error)
	DescribeDomain(req *DescribeDomainInput) (resp *DomainDetail, err error)
	DescribeWorkflowExecution(req *DescribeWorkflowExecutionInput) (resp *WorkflowExecutionDetail, err error)
	DescribeWorkflowType(req *DescribeWorkflowTypeInput) (resp *WorkflowTypeDetail, err error)
	GetWorkflowExecutionHistory(req *GetWorkflowExecutionHistoryInput) (resp *History, err error)
	ListActivityTypes(req *ListActivityTypesInput) (resp *ActivityTypeInfos, err error)
	ListClosedWorkflowExecutions(req *ListClosedWorkflowExecutionsInput) (resp *WorkflowExecutionInfos, err error)
	ListDomains(req *ListDomainsInput) (resp *DomainInfos, err error)
	ListOpenWorkflowExecutions(req *ListOpenWorkflowExecutionsInput) (resp *WorkflowExecutionInfos, err error)
	ListWorkflowTypes(req *ListWorkflowTypesInput) (resp *WorkflowTypeInfos, err error)
	PollForActivityTask(req *PollForActivityTaskInput) (resp *ActivityTask, err error)
	PollForDecisionTask(req *PollForDecisionTaskInput) (resp *DecisionTask, err error)
	RecordActivityTaskHeartbeat(req *RecordActivityTaskHeartbeatInput) (resp *ActivityTaskStatus, err error)
	RegisterActivityType(req *RegisterActivityTypeInput) (err error)
	RegisterDomain(req *RegisterDomainInput) (err error)
	RegisterWorkflowType(req *RegisterWorkflowTypeInput) (err error)
	RequestCancelWorkflowExecution(req *RequestCancelWorkflowExecutionInput) (err error)
	RespondActivityTaskCanceled(req *RespondActivityTaskCanceledInput) (err error)
	RespondActivityTaskCompleted(req *RespondActivityTaskCompletedInput) (err error)
	RespondActivityTaskFailed(req *RespondActivityTaskFailedInput) (err error)
	RespondDecisionTaskCompleted(req *RespondDecisionTaskCompletedInput) (err error)
	SignalWorkflowExecution(req *SignalWorkflowExecutionInput) (err error)
	StartWorkflowExecution(req *StartWorkflowExecutionInput) (resp *Run, err error)
	TerminateWorkflowExecution(req *TerminateWorkflowExecutionInput) (err error)
}
