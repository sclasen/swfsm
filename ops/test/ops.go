package test

import (
	. "github.com/awslabs/aws-sdk-go/gen/swf"
	"github.com/juju/errors"
)

//MockSWF is a mock swf cloent that can easily be extended to mock specfic ops in your tests.
type MockSWF struct{}

func (*MockSWF) CountClosedWorkflowExecutions(req *CountClosedWorkflowExecutionsInput) (resp *WorkflowExecutionCount, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) CountOpenWorkflowExecutions(req *CountOpenWorkflowExecutionsInput) (resp *WorkflowExecutionCount, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) CountPendingActivityTasks(req *CountPendingActivityTasksInput) (resp *PendingTaskCount, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) CountPendingDecisionTasks(req *CountPendingDecisionTasksInput) (resp *PendingTaskCount, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) DeprecateActivityType(req *DeprecateActivityTypeInput) (err error) {
	return errors.New("unmocked operation")
}
func (*MockSWF) DeprecateDomain(req *DeprecateDomainInput) (err error) {
	return errors.New("unmocked operation")
}
func (*MockSWF) DeprecateWorkflowType(req *DeprecateWorkflowTypeInput) (err error) {
	return errors.New("unmocked operation")
}
func (*MockSWF) DescribeActivityType(req *DescribeActivityTypeInput) (resp *ActivityTypeDetail, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) DescribeDomain(req *DescribeDomainInput) (resp *DomainDetail, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) DescribeWorkflowExecution(req *DescribeWorkflowExecutionInput) (resp *WorkflowExecutionDetail, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) DescribeWorkflowType(req *DescribeWorkflowTypeInput) (resp *WorkflowTypeDetail, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) GetWorkflowExecutionHistory(req *GetWorkflowExecutionHistoryInput) (resp *History, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) ListActivityTypes(req *ListActivityTypesInput) (resp *ActivityTypeInfos, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) ListClosedWorkflowExecutions(req *ListClosedWorkflowExecutionsInput) (resp *WorkflowExecutionInfos, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) ListDomains(req *ListDomainsInput) (resp *DomainInfos, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) ListOpenWorkflowExecutions(req *ListOpenWorkflowExecutionsInput) (resp *WorkflowExecutionInfos, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) ListWorkflowTypes(req *ListWorkflowTypesInput) (resp *WorkflowTypeInfos, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) PollForActivityTask(req *PollForActivityTaskInput) (resp *ActivityTask, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) PollForDecisionTask(req *PollForDecisionTaskInput) (resp *DecisionTask, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) RecordActivityTaskHeartbeat(req *RecordActivityTaskHeartbeatInput) (resp *ActivityTaskStatus, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) RegisterActivityType(req *RegisterActivityTypeInput) (err error) {
	return errors.New("unmocked operation")
}
func (*MockSWF) RegisterDomain(req *RegisterDomainInput) (err error) {
	return errors.New("unmocked operation")
}
func (*MockSWF) RegisterWorkflowType(req *RegisterWorkflowTypeInput) (err error) {
	return errors.New("unmocked operation")
}
func (*MockSWF) RequestCancelWorkflowExecution(req *RequestCancelWorkflowExecutionInput) (err error) {
	return errors.New("unmocked operation")
}
func (*MockSWF) RespondActivityTaskCanceled(req *RespondActivityTaskCanceledInput) (err error) {
	return errors.New("unmocked operation")
}
func (*MockSWF) RespondActivityTaskCompleted(req *RespondActivityTaskCompletedInput) (err error) {
	return errors.New("unmocked operation")
}
func (*MockSWF) RespondActivityTaskFailed(req *RespondActivityTaskFailedInput) (err error) {
	return errors.New("unmocked operation")
}
func (*MockSWF) RespondDecisionTaskCompleted(req *RespondDecisionTaskCompletedInput) (err error) {
	return errors.New("unmocked operation")
}
func (*MockSWF) SignalWorkflowExecution(req *SignalWorkflowExecutionInput) (err error) {
	return errors.New("unmocked operation")
}
func (*MockSWF) StartWorkflowExecution(req *StartWorkflowExecutionInput) (resp *Run, err error) {
	return nil, errors.New("unmocked operation")
}
func (*MockSWF) TerminateWorkflowExecution(req *TerminateWorkflowExecutionInput) (err error) {
	return errors.New("unmocked operation")
}
