package mocks

import "github.com/stretchr/testify/mock"

import "github.com/aws/aws-sdk-go/aws/request"
import "github.com/aws/aws-sdk-go/service/swf"

// AUTO-GENERATED MOCK. DO NOT EDIT.
// USE make mocks TO REGENERATE.

type SWFAPI struct {
	mock.Mock
}

func (m *SWFAPI) Name_CountClosedWorkflowExecutionsRequest() string {
	return "CountClosedWorkflowExecutionsRequest"
}
func (m *SWFAPI) MockOn_CountClosedWorkflowExecutionsRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("CountClosedWorkflowExecutionsRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_CountClosedWorkflowExecutionsRequest(_a0 *swf.CountClosedWorkflowExecutionsInput) *mock.Mock {
	return m.Mock.On("CountClosedWorkflowExecutionsRequest", _a0)
}
func (m *SWFAPI) MockOnAny_CountClosedWorkflowExecutionsRequest() *mock.Mock {
	return m.Mock.On("CountClosedWorkflowExecutionsRequest", mock.Anything)
}
func (m *SWFAPI) CountClosedWorkflowExecutionsRequest(_a0 *swf.CountClosedWorkflowExecutionsInput) (*request.Request, *swf.WorkflowExecutionCount) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.CountClosedWorkflowExecutionsInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.WorkflowExecutionCount
	if rf, ok := ret.Get(1).(func(*swf.CountClosedWorkflowExecutionsInput) *swf.WorkflowExecutionCount); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.WorkflowExecutionCount)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_CountClosedWorkflowExecutions() string {
	return "CountClosedWorkflowExecutions"
}
func (m *SWFAPI) MockOn_CountClosedWorkflowExecutions(_a0 interface{}) *mock.Mock {
	return m.Mock.On("CountClosedWorkflowExecutions", _a0)
}
func (m *SWFAPI) MockOnTyped_CountClosedWorkflowExecutions(_a0 *swf.CountClosedWorkflowExecutionsInput) *mock.Mock {
	return m.Mock.On("CountClosedWorkflowExecutions", _a0)
}
func (m *SWFAPI) MockOnAny_CountClosedWorkflowExecutions() *mock.Mock {
	return m.Mock.On("CountClosedWorkflowExecutions", mock.Anything)
}
func (m *SWFAPI) CountClosedWorkflowExecutions(_a0 *swf.CountClosedWorkflowExecutionsInput) (*swf.WorkflowExecutionCount, error) {
	ret := m.Called(_a0)

	var r0 *swf.WorkflowExecutionCount
	if rf, ok := ret.Get(0).(func(*swf.CountClosedWorkflowExecutionsInput) *swf.WorkflowExecutionCount); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.WorkflowExecutionCount)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.CountClosedWorkflowExecutionsInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_CountOpenWorkflowExecutionsRequest() string {
	return "CountOpenWorkflowExecutionsRequest"
}
func (m *SWFAPI) MockOn_CountOpenWorkflowExecutionsRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("CountOpenWorkflowExecutionsRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_CountOpenWorkflowExecutionsRequest(_a0 *swf.CountOpenWorkflowExecutionsInput) *mock.Mock {
	return m.Mock.On("CountOpenWorkflowExecutionsRequest", _a0)
}
func (m *SWFAPI) MockOnAny_CountOpenWorkflowExecutionsRequest() *mock.Mock {
	return m.Mock.On("CountOpenWorkflowExecutionsRequest", mock.Anything)
}
func (m *SWFAPI) CountOpenWorkflowExecutionsRequest(_a0 *swf.CountOpenWorkflowExecutionsInput) (*request.Request, *swf.WorkflowExecutionCount) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.CountOpenWorkflowExecutionsInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.WorkflowExecutionCount
	if rf, ok := ret.Get(1).(func(*swf.CountOpenWorkflowExecutionsInput) *swf.WorkflowExecutionCount); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.WorkflowExecutionCount)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_CountOpenWorkflowExecutions() string {
	return "CountOpenWorkflowExecutions"
}
func (m *SWFAPI) MockOn_CountOpenWorkflowExecutions(_a0 interface{}) *mock.Mock {
	return m.Mock.On("CountOpenWorkflowExecutions", _a0)
}
func (m *SWFAPI) MockOnTyped_CountOpenWorkflowExecutions(_a0 *swf.CountOpenWorkflowExecutionsInput) *mock.Mock {
	return m.Mock.On("CountOpenWorkflowExecutions", _a0)
}
func (m *SWFAPI) MockOnAny_CountOpenWorkflowExecutions() *mock.Mock {
	return m.Mock.On("CountOpenWorkflowExecutions", mock.Anything)
}
func (m *SWFAPI) CountOpenWorkflowExecutions(_a0 *swf.CountOpenWorkflowExecutionsInput) (*swf.WorkflowExecutionCount, error) {
	ret := m.Called(_a0)

	var r0 *swf.WorkflowExecutionCount
	if rf, ok := ret.Get(0).(func(*swf.CountOpenWorkflowExecutionsInput) *swf.WorkflowExecutionCount); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.WorkflowExecutionCount)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.CountOpenWorkflowExecutionsInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_CountPendingActivityTasksRequest() string {
	return "CountPendingActivityTasksRequest"
}
func (m *SWFAPI) MockOn_CountPendingActivityTasksRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("CountPendingActivityTasksRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_CountPendingActivityTasksRequest(_a0 *swf.CountPendingActivityTasksInput) *mock.Mock {
	return m.Mock.On("CountPendingActivityTasksRequest", _a0)
}
func (m *SWFAPI) MockOnAny_CountPendingActivityTasksRequest() *mock.Mock {
	return m.Mock.On("CountPendingActivityTasksRequest", mock.Anything)
}
func (m *SWFAPI) CountPendingActivityTasksRequest(_a0 *swf.CountPendingActivityTasksInput) (*request.Request, *swf.PendingTaskCount) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.CountPendingActivityTasksInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.PendingTaskCount
	if rf, ok := ret.Get(1).(func(*swf.CountPendingActivityTasksInput) *swf.PendingTaskCount); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.PendingTaskCount)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_CountPendingActivityTasks() string {
	return "CountPendingActivityTasks"
}
func (m *SWFAPI) MockOn_CountPendingActivityTasks(_a0 interface{}) *mock.Mock {
	return m.Mock.On("CountPendingActivityTasks", _a0)
}
func (m *SWFAPI) MockOnTyped_CountPendingActivityTasks(_a0 *swf.CountPendingActivityTasksInput) *mock.Mock {
	return m.Mock.On("CountPendingActivityTasks", _a0)
}
func (m *SWFAPI) MockOnAny_CountPendingActivityTasks() *mock.Mock {
	return m.Mock.On("CountPendingActivityTasks", mock.Anything)
}
func (m *SWFAPI) CountPendingActivityTasks(_a0 *swf.CountPendingActivityTasksInput) (*swf.PendingTaskCount, error) {
	ret := m.Called(_a0)

	var r0 *swf.PendingTaskCount
	if rf, ok := ret.Get(0).(func(*swf.CountPendingActivityTasksInput) *swf.PendingTaskCount); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.PendingTaskCount)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.CountPendingActivityTasksInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_CountPendingDecisionTasksRequest() string {
	return "CountPendingDecisionTasksRequest"
}
func (m *SWFAPI) MockOn_CountPendingDecisionTasksRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("CountPendingDecisionTasksRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_CountPendingDecisionTasksRequest(_a0 *swf.CountPendingDecisionTasksInput) *mock.Mock {
	return m.Mock.On("CountPendingDecisionTasksRequest", _a0)
}
func (m *SWFAPI) MockOnAny_CountPendingDecisionTasksRequest() *mock.Mock {
	return m.Mock.On("CountPendingDecisionTasksRequest", mock.Anything)
}
func (m *SWFAPI) CountPendingDecisionTasksRequest(_a0 *swf.CountPendingDecisionTasksInput) (*request.Request, *swf.PendingTaskCount) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.CountPendingDecisionTasksInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.PendingTaskCount
	if rf, ok := ret.Get(1).(func(*swf.CountPendingDecisionTasksInput) *swf.PendingTaskCount); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.PendingTaskCount)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_CountPendingDecisionTasks() string {
	return "CountPendingDecisionTasks"
}
func (m *SWFAPI) MockOn_CountPendingDecisionTasks(_a0 interface{}) *mock.Mock {
	return m.Mock.On("CountPendingDecisionTasks", _a0)
}
func (m *SWFAPI) MockOnTyped_CountPendingDecisionTasks(_a0 *swf.CountPendingDecisionTasksInput) *mock.Mock {
	return m.Mock.On("CountPendingDecisionTasks", _a0)
}
func (m *SWFAPI) MockOnAny_CountPendingDecisionTasks() *mock.Mock {
	return m.Mock.On("CountPendingDecisionTasks", mock.Anything)
}
func (m *SWFAPI) CountPendingDecisionTasks(_a0 *swf.CountPendingDecisionTasksInput) (*swf.PendingTaskCount, error) {
	ret := m.Called(_a0)

	var r0 *swf.PendingTaskCount
	if rf, ok := ret.Get(0).(func(*swf.CountPendingDecisionTasksInput) *swf.PendingTaskCount); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.PendingTaskCount)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.CountPendingDecisionTasksInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_DeprecateActivityTypeRequest() string {
	return "DeprecateActivityTypeRequest"
}
func (m *SWFAPI) MockOn_DeprecateActivityTypeRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DeprecateActivityTypeRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_DeprecateActivityTypeRequest(_a0 *swf.DeprecateActivityTypeInput) *mock.Mock {
	return m.Mock.On("DeprecateActivityTypeRequest", _a0)
}
func (m *SWFAPI) MockOnAny_DeprecateActivityTypeRequest() *mock.Mock {
	return m.Mock.On("DeprecateActivityTypeRequest", mock.Anything)
}
func (m *SWFAPI) DeprecateActivityTypeRequest(_a0 *swf.DeprecateActivityTypeInput) (*request.Request, *swf.DeprecateActivityTypeOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.DeprecateActivityTypeInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.DeprecateActivityTypeOutput
	if rf, ok := ret.Get(1).(func(*swf.DeprecateActivityTypeInput) *swf.DeprecateActivityTypeOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.DeprecateActivityTypeOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_DeprecateActivityType() string {
	return "DeprecateActivityType"
}
func (m *SWFAPI) MockOn_DeprecateActivityType(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DeprecateActivityType", _a0)
}
func (m *SWFAPI) MockOnTyped_DeprecateActivityType(_a0 *swf.DeprecateActivityTypeInput) *mock.Mock {
	return m.Mock.On("DeprecateActivityType", _a0)
}
func (m *SWFAPI) MockOnAny_DeprecateActivityType() *mock.Mock {
	return m.Mock.On("DeprecateActivityType", mock.Anything)
}
func (m *SWFAPI) DeprecateActivityType(_a0 *swf.DeprecateActivityTypeInput) (*swf.DeprecateActivityTypeOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.DeprecateActivityTypeOutput
	if rf, ok := ret.Get(0).(func(*swf.DeprecateActivityTypeInput) *swf.DeprecateActivityTypeOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.DeprecateActivityTypeOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.DeprecateActivityTypeInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_DeprecateDomainRequest() string {
	return "DeprecateDomainRequest"
}
func (m *SWFAPI) MockOn_DeprecateDomainRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DeprecateDomainRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_DeprecateDomainRequest(_a0 *swf.DeprecateDomainInput) *mock.Mock {
	return m.Mock.On("DeprecateDomainRequest", _a0)
}
func (m *SWFAPI) MockOnAny_DeprecateDomainRequest() *mock.Mock {
	return m.Mock.On("DeprecateDomainRequest", mock.Anything)
}
func (m *SWFAPI) DeprecateDomainRequest(_a0 *swf.DeprecateDomainInput) (*request.Request, *swf.DeprecateDomainOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.DeprecateDomainInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.DeprecateDomainOutput
	if rf, ok := ret.Get(1).(func(*swf.DeprecateDomainInput) *swf.DeprecateDomainOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.DeprecateDomainOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_DeprecateDomain() string {
	return "DeprecateDomain"
}
func (m *SWFAPI) MockOn_DeprecateDomain(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DeprecateDomain", _a0)
}
func (m *SWFAPI) MockOnTyped_DeprecateDomain(_a0 *swf.DeprecateDomainInput) *mock.Mock {
	return m.Mock.On("DeprecateDomain", _a0)
}
func (m *SWFAPI) MockOnAny_DeprecateDomain() *mock.Mock {
	return m.Mock.On("DeprecateDomain", mock.Anything)
}
func (m *SWFAPI) DeprecateDomain(_a0 *swf.DeprecateDomainInput) (*swf.DeprecateDomainOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.DeprecateDomainOutput
	if rf, ok := ret.Get(0).(func(*swf.DeprecateDomainInput) *swf.DeprecateDomainOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.DeprecateDomainOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.DeprecateDomainInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_DeprecateWorkflowTypeRequest() string {
	return "DeprecateWorkflowTypeRequest"
}
func (m *SWFAPI) MockOn_DeprecateWorkflowTypeRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DeprecateWorkflowTypeRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_DeprecateWorkflowTypeRequest(_a0 *swf.DeprecateWorkflowTypeInput) *mock.Mock {
	return m.Mock.On("DeprecateWorkflowTypeRequest", _a0)
}
func (m *SWFAPI) MockOnAny_DeprecateWorkflowTypeRequest() *mock.Mock {
	return m.Mock.On("DeprecateWorkflowTypeRequest", mock.Anything)
}
func (m *SWFAPI) DeprecateWorkflowTypeRequest(_a0 *swf.DeprecateWorkflowTypeInput) (*request.Request, *swf.DeprecateWorkflowTypeOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.DeprecateWorkflowTypeInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.DeprecateWorkflowTypeOutput
	if rf, ok := ret.Get(1).(func(*swf.DeprecateWorkflowTypeInput) *swf.DeprecateWorkflowTypeOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.DeprecateWorkflowTypeOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_DeprecateWorkflowType() string {
	return "DeprecateWorkflowType"
}
func (m *SWFAPI) MockOn_DeprecateWorkflowType(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DeprecateWorkflowType", _a0)
}
func (m *SWFAPI) MockOnTyped_DeprecateWorkflowType(_a0 *swf.DeprecateWorkflowTypeInput) *mock.Mock {
	return m.Mock.On("DeprecateWorkflowType", _a0)
}
func (m *SWFAPI) MockOnAny_DeprecateWorkflowType() *mock.Mock {
	return m.Mock.On("DeprecateWorkflowType", mock.Anything)
}
func (m *SWFAPI) DeprecateWorkflowType(_a0 *swf.DeprecateWorkflowTypeInput) (*swf.DeprecateWorkflowTypeOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.DeprecateWorkflowTypeOutput
	if rf, ok := ret.Get(0).(func(*swf.DeprecateWorkflowTypeInput) *swf.DeprecateWorkflowTypeOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.DeprecateWorkflowTypeOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.DeprecateWorkflowTypeInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_DescribeActivityTypeRequest() string {
	return "DescribeActivityTypeRequest"
}
func (m *SWFAPI) MockOn_DescribeActivityTypeRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DescribeActivityTypeRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_DescribeActivityTypeRequest(_a0 *swf.DescribeActivityTypeInput) *mock.Mock {
	return m.Mock.On("DescribeActivityTypeRequest", _a0)
}
func (m *SWFAPI) MockOnAny_DescribeActivityTypeRequest() *mock.Mock {
	return m.Mock.On("DescribeActivityTypeRequest", mock.Anything)
}
func (m *SWFAPI) DescribeActivityTypeRequest(_a0 *swf.DescribeActivityTypeInput) (*request.Request, *swf.DescribeActivityTypeOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.DescribeActivityTypeInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.DescribeActivityTypeOutput
	if rf, ok := ret.Get(1).(func(*swf.DescribeActivityTypeInput) *swf.DescribeActivityTypeOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.DescribeActivityTypeOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_DescribeActivityType() string {
	return "DescribeActivityType"
}
func (m *SWFAPI) MockOn_DescribeActivityType(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DescribeActivityType", _a0)
}
func (m *SWFAPI) MockOnTyped_DescribeActivityType(_a0 *swf.DescribeActivityTypeInput) *mock.Mock {
	return m.Mock.On("DescribeActivityType", _a0)
}
func (m *SWFAPI) MockOnAny_DescribeActivityType() *mock.Mock {
	return m.Mock.On("DescribeActivityType", mock.Anything)
}
func (m *SWFAPI) DescribeActivityType(_a0 *swf.DescribeActivityTypeInput) (*swf.DescribeActivityTypeOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.DescribeActivityTypeOutput
	if rf, ok := ret.Get(0).(func(*swf.DescribeActivityTypeInput) *swf.DescribeActivityTypeOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.DescribeActivityTypeOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.DescribeActivityTypeInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_DescribeDomainRequest() string {
	return "DescribeDomainRequest"
}
func (m *SWFAPI) MockOn_DescribeDomainRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DescribeDomainRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_DescribeDomainRequest(_a0 *swf.DescribeDomainInput) *mock.Mock {
	return m.Mock.On("DescribeDomainRequest", _a0)
}
func (m *SWFAPI) MockOnAny_DescribeDomainRequest() *mock.Mock {
	return m.Mock.On("DescribeDomainRequest", mock.Anything)
}
func (m *SWFAPI) DescribeDomainRequest(_a0 *swf.DescribeDomainInput) (*request.Request, *swf.DescribeDomainOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.DescribeDomainInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.DescribeDomainOutput
	if rf, ok := ret.Get(1).(func(*swf.DescribeDomainInput) *swf.DescribeDomainOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.DescribeDomainOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_DescribeDomain() string {
	return "DescribeDomain"
}
func (m *SWFAPI) MockOn_DescribeDomain(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DescribeDomain", _a0)
}
func (m *SWFAPI) MockOnTyped_DescribeDomain(_a0 *swf.DescribeDomainInput) *mock.Mock {
	return m.Mock.On("DescribeDomain", _a0)
}
func (m *SWFAPI) MockOnAny_DescribeDomain() *mock.Mock {
	return m.Mock.On("DescribeDomain", mock.Anything)
}
func (m *SWFAPI) DescribeDomain(_a0 *swf.DescribeDomainInput) (*swf.DescribeDomainOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.DescribeDomainOutput
	if rf, ok := ret.Get(0).(func(*swf.DescribeDomainInput) *swf.DescribeDomainOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.DescribeDomainOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.DescribeDomainInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_DescribeWorkflowExecutionRequest() string {
	return "DescribeWorkflowExecutionRequest"
}
func (m *SWFAPI) MockOn_DescribeWorkflowExecutionRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DescribeWorkflowExecutionRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_DescribeWorkflowExecutionRequest(_a0 *swf.DescribeWorkflowExecutionInput) *mock.Mock {
	return m.Mock.On("DescribeWorkflowExecutionRequest", _a0)
}
func (m *SWFAPI) MockOnAny_DescribeWorkflowExecutionRequest() *mock.Mock {
	return m.Mock.On("DescribeWorkflowExecutionRequest", mock.Anything)
}
func (m *SWFAPI) DescribeWorkflowExecutionRequest(_a0 *swf.DescribeWorkflowExecutionInput) (*request.Request, *swf.DescribeWorkflowExecutionOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.DescribeWorkflowExecutionInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.DescribeWorkflowExecutionOutput
	if rf, ok := ret.Get(1).(func(*swf.DescribeWorkflowExecutionInput) *swf.DescribeWorkflowExecutionOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.DescribeWorkflowExecutionOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_DescribeWorkflowExecution() string {
	return "DescribeWorkflowExecution"
}
func (m *SWFAPI) MockOn_DescribeWorkflowExecution(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DescribeWorkflowExecution", _a0)
}
func (m *SWFAPI) MockOnTyped_DescribeWorkflowExecution(_a0 *swf.DescribeWorkflowExecutionInput) *mock.Mock {
	return m.Mock.On("DescribeWorkflowExecution", _a0)
}
func (m *SWFAPI) MockOnAny_DescribeWorkflowExecution() *mock.Mock {
	return m.Mock.On("DescribeWorkflowExecution", mock.Anything)
}
func (m *SWFAPI) DescribeWorkflowExecution(_a0 *swf.DescribeWorkflowExecutionInput) (*swf.DescribeWorkflowExecutionOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.DescribeWorkflowExecutionOutput
	if rf, ok := ret.Get(0).(func(*swf.DescribeWorkflowExecutionInput) *swf.DescribeWorkflowExecutionOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.DescribeWorkflowExecutionOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.DescribeWorkflowExecutionInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_DescribeWorkflowTypeRequest() string {
	return "DescribeWorkflowTypeRequest"
}
func (m *SWFAPI) MockOn_DescribeWorkflowTypeRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DescribeWorkflowTypeRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_DescribeWorkflowTypeRequest(_a0 *swf.DescribeWorkflowTypeInput) *mock.Mock {
	return m.Mock.On("DescribeWorkflowTypeRequest", _a0)
}
func (m *SWFAPI) MockOnAny_DescribeWorkflowTypeRequest() *mock.Mock {
	return m.Mock.On("DescribeWorkflowTypeRequest", mock.Anything)
}
func (m *SWFAPI) DescribeWorkflowTypeRequest(_a0 *swf.DescribeWorkflowTypeInput) (*request.Request, *swf.DescribeWorkflowTypeOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.DescribeWorkflowTypeInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.DescribeWorkflowTypeOutput
	if rf, ok := ret.Get(1).(func(*swf.DescribeWorkflowTypeInput) *swf.DescribeWorkflowTypeOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.DescribeWorkflowTypeOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_DescribeWorkflowType() string {
	return "DescribeWorkflowType"
}
func (m *SWFAPI) MockOn_DescribeWorkflowType(_a0 interface{}) *mock.Mock {
	return m.Mock.On("DescribeWorkflowType", _a0)
}
func (m *SWFAPI) MockOnTyped_DescribeWorkflowType(_a0 *swf.DescribeWorkflowTypeInput) *mock.Mock {
	return m.Mock.On("DescribeWorkflowType", _a0)
}
func (m *SWFAPI) MockOnAny_DescribeWorkflowType() *mock.Mock {
	return m.Mock.On("DescribeWorkflowType", mock.Anything)
}
func (m *SWFAPI) DescribeWorkflowType(_a0 *swf.DescribeWorkflowTypeInput) (*swf.DescribeWorkflowTypeOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.DescribeWorkflowTypeOutput
	if rf, ok := ret.Get(0).(func(*swf.DescribeWorkflowTypeInput) *swf.DescribeWorkflowTypeOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.DescribeWorkflowTypeOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.DescribeWorkflowTypeInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_GetWorkflowExecutionHistoryRequest() string {
	return "GetWorkflowExecutionHistoryRequest"
}
func (m *SWFAPI) MockOn_GetWorkflowExecutionHistoryRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("GetWorkflowExecutionHistoryRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_GetWorkflowExecutionHistoryRequest(_a0 *swf.GetWorkflowExecutionHistoryInput) *mock.Mock {
	return m.Mock.On("GetWorkflowExecutionHistoryRequest", _a0)
}
func (m *SWFAPI) MockOnAny_GetWorkflowExecutionHistoryRequest() *mock.Mock {
	return m.Mock.On("GetWorkflowExecutionHistoryRequest", mock.Anything)
}
func (m *SWFAPI) GetWorkflowExecutionHistoryRequest(_a0 *swf.GetWorkflowExecutionHistoryInput) (*request.Request, *swf.GetWorkflowExecutionHistoryOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.GetWorkflowExecutionHistoryInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.GetWorkflowExecutionHistoryOutput
	if rf, ok := ret.Get(1).(func(*swf.GetWorkflowExecutionHistoryInput) *swf.GetWorkflowExecutionHistoryOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.GetWorkflowExecutionHistoryOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_GetWorkflowExecutionHistory() string {
	return "GetWorkflowExecutionHistory"
}
func (m *SWFAPI) MockOn_GetWorkflowExecutionHistory(_a0 interface{}) *mock.Mock {
	return m.Mock.On("GetWorkflowExecutionHistory", _a0)
}
func (m *SWFAPI) MockOnTyped_GetWorkflowExecutionHistory(_a0 *swf.GetWorkflowExecutionHistoryInput) *mock.Mock {
	return m.Mock.On("GetWorkflowExecutionHistory", _a0)
}
func (m *SWFAPI) MockOnAny_GetWorkflowExecutionHistory() *mock.Mock {
	return m.Mock.On("GetWorkflowExecutionHistory", mock.Anything)
}
func (m *SWFAPI) GetWorkflowExecutionHistory(_a0 *swf.GetWorkflowExecutionHistoryInput) (*swf.GetWorkflowExecutionHistoryOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.GetWorkflowExecutionHistoryOutput
	if rf, ok := ret.Get(0).(func(*swf.GetWorkflowExecutionHistoryInput) *swf.GetWorkflowExecutionHistoryOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.GetWorkflowExecutionHistoryOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.GetWorkflowExecutionHistoryInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_GetWorkflowExecutionHistoryPages() string {
	return "GetWorkflowExecutionHistoryPages"
}
func (m *SWFAPI) MockOn_GetWorkflowExecutionHistoryPages(_a0 interface{}, _a1 interface{}) *mock.Mock {
	return m.Mock.On("GetWorkflowExecutionHistoryPages", _a0, _a1)
}
func (m *SWFAPI) MockOnTyped_GetWorkflowExecutionHistoryPages(_a0 *swf.GetWorkflowExecutionHistoryInput, _a1 func(*swf.GetWorkflowExecutionHistoryOutput, bool) bool) *mock.Mock {
	return m.Mock.On("GetWorkflowExecutionHistoryPages", _a0, _a1)
}
func (m *SWFAPI) MockOnAny_GetWorkflowExecutionHistoryPages() *mock.Mock {
	return m.Mock.On("GetWorkflowExecutionHistoryPages", mock.Anything, mock.Anything)
}
func (m *SWFAPI) GetWorkflowExecutionHistoryPages(_a0 *swf.GetWorkflowExecutionHistoryInput, _a1 func(*swf.GetWorkflowExecutionHistoryOutput, bool) bool) error {
	ret := m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(*swf.GetWorkflowExecutionHistoryInput, func(*swf.GetWorkflowExecutionHistoryOutput, bool) bool) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (m *SWFAPI) Name_ListActivityTypesRequest() string {
	return "ListActivityTypesRequest"
}
func (m *SWFAPI) MockOn_ListActivityTypesRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("ListActivityTypesRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_ListActivityTypesRequest(_a0 *swf.ListActivityTypesInput) *mock.Mock {
	return m.Mock.On("ListActivityTypesRequest", _a0)
}
func (m *SWFAPI) MockOnAny_ListActivityTypesRequest() *mock.Mock {
	return m.Mock.On("ListActivityTypesRequest", mock.Anything)
}
func (m *SWFAPI) ListActivityTypesRequest(_a0 *swf.ListActivityTypesInput) (*request.Request, *swf.ListActivityTypesOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.ListActivityTypesInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.ListActivityTypesOutput
	if rf, ok := ret.Get(1).(func(*swf.ListActivityTypesInput) *swf.ListActivityTypesOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.ListActivityTypesOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_ListActivityTypes() string {
	return "ListActivityTypes"
}
func (m *SWFAPI) MockOn_ListActivityTypes(_a0 interface{}) *mock.Mock {
	return m.Mock.On("ListActivityTypes", _a0)
}
func (m *SWFAPI) MockOnTyped_ListActivityTypes(_a0 *swf.ListActivityTypesInput) *mock.Mock {
	return m.Mock.On("ListActivityTypes", _a0)
}
func (m *SWFAPI) MockOnAny_ListActivityTypes() *mock.Mock {
	return m.Mock.On("ListActivityTypes", mock.Anything)
}
func (m *SWFAPI) ListActivityTypes(_a0 *swf.ListActivityTypesInput) (*swf.ListActivityTypesOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.ListActivityTypesOutput
	if rf, ok := ret.Get(0).(func(*swf.ListActivityTypesInput) *swf.ListActivityTypesOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.ListActivityTypesOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.ListActivityTypesInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_ListActivityTypesPages() string {
	return "ListActivityTypesPages"
}
func (m *SWFAPI) MockOn_ListActivityTypesPages(_a0 interface{}, _a1 interface{}) *mock.Mock {
	return m.Mock.On("ListActivityTypesPages", _a0, _a1)
}
func (m *SWFAPI) MockOnTyped_ListActivityTypesPages(_a0 *swf.ListActivityTypesInput, _a1 func(*swf.ListActivityTypesOutput, bool) bool) *mock.Mock {
	return m.Mock.On("ListActivityTypesPages", _a0, _a1)
}
func (m *SWFAPI) MockOnAny_ListActivityTypesPages() *mock.Mock {
	return m.Mock.On("ListActivityTypesPages", mock.Anything, mock.Anything)
}
func (m *SWFAPI) ListActivityTypesPages(_a0 *swf.ListActivityTypesInput, _a1 func(*swf.ListActivityTypesOutput, bool) bool) error {
	ret := m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(*swf.ListActivityTypesInput, func(*swf.ListActivityTypesOutput, bool) bool) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (m *SWFAPI) Name_ListClosedWorkflowExecutionsRequest() string {
	return "ListClosedWorkflowExecutionsRequest"
}
func (m *SWFAPI) MockOn_ListClosedWorkflowExecutionsRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("ListClosedWorkflowExecutionsRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_ListClosedWorkflowExecutionsRequest(_a0 *swf.ListClosedWorkflowExecutionsInput) *mock.Mock {
	return m.Mock.On("ListClosedWorkflowExecutionsRequest", _a0)
}
func (m *SWFAPI) MockOnAny_ListClosedWorkflowExecutionsRequest() *mock.Mock {
	return m.Mock.On("ListClosedWorkflowExecutionsRequest", mock.Anything)
}
func (m *SWFAPI) ListClosedWorkflowExecutionsRequest(_a0 *swf.ListClosedWorkflowExecutionsInput) (*request.Request, *swf.WorkflowExecutionInfos) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.ListClosedWorkflowExecutionsInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.WorkflowExecutionInfos
	if rf, ok := ret.Get(1).(func(*swf.ListClosedWorkflowExecutionsInput) *swf.WorkflowExecutionInfos); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.WorkflowExecutionInfos)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_ListClosedWorkflowExecutions() string {
	return "ListClosedWorkflowExecutions"
}
func (m *SWFAPI) MockOn_ListClosedWorkflowExecutions(_a0 interface{}) *mock.Mock {
	return m.Mock.On("ListClosedWorkflowExecutions", _a0)
}
func (m *SWFAPI) MockOnTyped_ListClosedWorkflowExecutions(_a0 *swf.ListClosedWorkflowExecutionsInput) *mock.Mock {
	return m.Mock.On("ListClosedWorkflowExecutions", _a0)
}
func (m *SWFAPI) MockOnAny_ListClosedWorkflowExecutions() *mock.Mock {
	return m.Mock.On("ListClosedWorkflowExecutions", mock.Anything)
}
func (m *SWFAPI) ListClosedWorkflowExecutions(_a0 *swf.ListClosedWorkflowExecutionsInput) (*swf.WorkflowExecutionInfos, error) {
	ret := m.Called(_a0)

	var r0 *swf.WorkflowExecutionInfos
	if rf, ok := ret.Get(0).(func(*swf.ListClosedWorkflowExecutionsInput) *swf.WorkflowExecutionInfos); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.WorkflowExecutionInfos)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.ListClosedWorkflowExecutionsInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_ListClosedWorkflowExecutionsPages() string {
	return "ListClosedWorkflowExecutionsPages"
}
func (m *SWFAPI) MockOn_ListClosedWorkflowExecutionsPages(_a0 interface{}, _a1 interface{}) *mock.Mock {
	return m.Mock.On("ListClosedWorkflowExecutionsPages", _a0, _a1)
}
func (m *SWFAPI) MockOnTyped_ListClosedWorkflowExecutionsPages(_a0 *swf.ListClosedWorkflowExecutionsInput, _a1 func(*swf.WorkflowExecutionInfos, bool) bool) *mock.Mock {
	return m.Mock.On("ListClosedWorkflowExecutionsPages", _a0, _a1)
}
func (m *SWFAPI) MockOnAny_ListClosedWorkflowExecutionsPages() *mock.Mock {
	return m.Mock.On("ListClosedWorkflowExecutionsPages", mock.Anything, mock.Anything)
}
func (m *SWFAPI) ListClosedWorkflowExecutionsPages(_a0 *swf.ListClosedWorkflowExecutionsInput, _a1 func(*swf.WorkflowExecutionInfos, bool) bool) error {
	ret := m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(*swf.ListClosedWorkflowExecutionsInput, func(*swf.WorkflowExecutionInfos, bool) bool) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (m *SWFAPI) Name_ListDomainsRequest() string {
	return "ListDomainsRequest"
}
func (m *SWFAPI) MockOn_ListDomainsRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("ListDomainsRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_ListDomainsRequest(_a0 *swf.ListDomainsInput) *mock.Mock {
	return m.Mock.On("ListDomainsRequest", _a0)
}
func (m *SWFAPI) MockOnAny_ListDomainsRequest() *mock.Mock {
	return m.Mock.On("ListDomainsRequest", mock.Anything)
}
func (m *SWFAPI) ListDomainsRequest(_a0 *swf.ListDomainsInput) (*request.Request, *swf.ListDomainsOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.ListDomainsInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.ListDomainsOutput
	if rf, ok := ret.Get(1).(func(*swf.ListDomainsInput) *swf.ListDomainsOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.ListDomainsOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_ListDomains() string {
	return "ListDomains"
}
func (m *SWFAPI) MockOn_ListDomains(_a0 interface{}) *mock.Mock {
	return m.Mock.On("ListDomains", _a0)
}
func (m *SWFAPI) MockOnTyped_ListDomains(_a0 *swf.ListDomainsInput) *mock.Mock {
	return m.Mock.On("ListDomains", _a0)
}
func (m *SWFAPI) MockOnAny_ListDomains() *mock.Mock {
	return m.Mock.On("ListDomains", mock.Anything)
}
func (m *SWFAPI) ListDomains(_a0 *swf.ListDomainsInput) (*swf.ListDomainsOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.ListDomainsOutput
	if rf, ok := ret.Get(0).(func(*swf.ListDomainsInput) *swf.ListDomainsOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.ListDomainsOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.ListDomainsInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_ListDomainsPages() string {
	return "ListDomainsPages"
}
func (m *SWFAPI) MockOn_ListDomainsPages(_a0 interface{}, _a1 interface{}) *mock.Mock {
	return m.Mock.On("ListDomainsPages", _a0, _a1)
}
func (m *SWFAPI) MockOnTyped_ListDomainsPages(_a0 *swf.ListDomainsInput, _a1 func(*swf.ListDomainsOutput, bool) bool) *mock.Mock {
	return m.Mock.On("ListDomainsPages", _a0, _a1)
}
func (m *SWFAPI) MockOnAny_ListDomainsPages() *mock.Mock {
	return m.Mock.On("ListDomainsPages", mock.Anything, mock.Anything)
}
func (m *SWFAPI) ListDomainsPages(_a0 *swf.ListDomainsInput, _a1 func(*swf.ListDomainsOutput, bool) bool) error {
	ret := m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(*swf.ListDomainsInput, func(*swf.ListDomainsOutput, bool) bool) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (m *SWFAPI) Name_ListOpenWorkflowExecutionsRequest() string {
	return "ListOpenWorkflowExecutionsRequest"
}
func (m *SWFAPI) MockOn_ListOpenWorkflowExecutionsRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("ListOpenWorkflowExecutionsRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_ListOpenWorkflowExecutionsRequest(_a0 *swf.ListOpenWorkflowExecutionsInput) *mock.Mock {
	return m.Mock.On("ListOpenWorkflowExecutionsRequest", _a0)
}
func (m *SWFAPI) MockOnAny_ListOpenWorkflowExecutionsRequest() *mock.Mock {
	return m.Mock.On("ListOpenWorkflowExecutionsRequest", mock.Anything)
}
func (m *SWFAPI) ListOpenWorkflowExecutionsRequest(_a0 *swf.ListOpenWorkflowExecutionsInput) (*request.Request, *swf.WorkflowExecutionInfos) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.ListOpenWorkflowExecutionsInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.WorkflowExecutionInfos
	if rf, ok := ret.Get(1).(func(*swf.ListOpenWorkflowExecutionsInput) *swf.WorkflowExecutionInfos); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.WorkflowExecutionInfos)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_ListOpenWorkflowExecutions() string {
	return "ListOpenWorkflowExecutions"
}
func (m *SWFAPI) MockOn_ListOpenWorkflowExecutions(_a0 interface{}) *mock.Mock {
	return m.Mock.On("ListOpenWorkflowExecutions", _a0)
}
func (m *SWFAPI) MockOnTyped_ListOpenWorkflowExecutions(_a0 *swf.ListOpenWorkflowExecutionsInput) *mock.Mock {
	return m.Mock.On("ListOpenWorkflowExecutions", _a0)
}
func (m *SWFAPI) MockOnAny_ListOpenWorkflowExecutions() *mock.Mock {
	return m.Mock.On("ListOpenWorkflowExecutions", mock.Anything)
}
func (m *SWFAPI) ListOpenWorkflowExecutions(_a0 *swf.ListOpenWorkflowExecutionsInput) (*swf.WorkflowExecutionInfos, error) {
	ret := m.Called(_a0)

	var r0 *swf.WorkflowExecutionInfos
	if rf, ok := ret.Get(0).(func(*swf.ListOpenWorkflowExecutionsInput) *swf.WorkflowExecutionInfos); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.WorkflowExecutionInfos)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.ListOpenWorkflowExecutionsInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_ListOpenWorkflowExecutionsPages() string {
	return "ListOpenWorkflowExecutionsPages"
}
func (m *SWFAPI) MockOn_ListOpenWorkflowExecutionsPages(_a0 interface{}, _a1 interface{}) *mock.Mock {
	return m.Mock.On("ListOpenWorkflowExecutionsPages", _a0, _a1)
}
func (m *SWFAPI) MockOnTyped_ListOpenWorkflowExecutionsPages(_a0 *swf.ListOpenWorkflowExecutionsInput, _a1 func(*swf.WorkflowExecutionInfos, bool) bool) *mock.Mock {
	return m.Mock.On("ListOpenWorkflowExecutionsPages", _a0, _a1)
}
func (m *SWFAPI) MockOnAny_ListOpenWorkflowExecutionsPages() *mock.Mock {
	return m.Mock.On("ListOpenWorkflowExecutionsPages", mock.Anything, mock.Anything)
}
func (m *SWFAPI) ListOpenWorkflowExecutionsPages(_a0 *swf.ListOpenWorkflowExecutionsInput, _a1 func(*swf.WorkflowExecutionInfos, bool) bool) error {
	ret := m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(*swf.ListOpenWorkflowExecutionsInput, func(*swf.WorkflowExecutionInfos, bool) bool) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (m *SWFAPI) Name_ListWorkflowTypesRequest() string {
	return "ListWorkflowTypesRequest"
}
func (m *SWFAPI) MockOn_ListWorkflowTypesRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("ListWorkflowTypesRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_ListWorkflowTypesRequest(_a0 *swf.ListWorkflowTypesInput) *mock.Mock {
	return m.Mock.On("ListWorkflowTypesRequest", _a0)
}
func (m *SWFAPI) MockOnAny_ListWorkflowTypesRequest() *mock.Mock {
	return m.Mock.On("ListWorkflowTypesRequest", mock.Anything)
}
func (m *SWFAPI) ListWorkflowTypesRequest(_a0 *swf.ListWorkflowTypesInput) (*request.Request, *swf.ListWorkflowTypesOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.ListWorkflowTypesInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.ListWorkflowTypesOutput
	if rf, ok := ret.Get(1).(func(*swf.ListWorkflowTypesInput) *swf.ListWorkflowTypesOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.ListWorkflowTypesOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_ListWorkflowTypes() string {
	return "ListWorkflowTypes"
}
func (m *SWFAPI) MockOn_ListWorkflowTypes(_a0 interface{}) *mock.Mock {
	return m.Mock.On("ListWorkflowTypes", _a0)
}
func (m *SWFAPI) MockOnTyped_ListWorkflowTypes(_a0 *swf.ListWorkflowTypesInput) *mock.Mock {
	return m.Mock.On("ListWorkflowTypes", _a0)
}
func (m *SWFAPI) MockOnAny_ListWorkflowTypes() *mock.Mock {
	return m.Mock.On("ListWorkflowTypes", mock.Anything)
}
func (m *SWFAPI) ListWorkflowTypes(_a0 *swf.ListWorkflowTypesInput) (*swf.ListWorkflowTypesOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.ListWorkflowTypesOutput
	if rf, ok := ret.Get(0).(func(*swf.ListWorkflowTypesInput) *swf.ListWorkflowTypesOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.ListWorkflowTypesOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.ListWorkflowTypesInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_ListWorkflowTypesPages() string {
	return "ListWorkflowTypesPages"
}
func (m *SWFAPI) MockOn_ListWorkflowTypesPages(_a0 interface{}, _a1 interface{}) *mock.Mock {
	return m.Mock.On("ListWorkflowTypesPages", _a0, _a1)
}
func (m *SWFAPI) MockOnTyped_ListWorkflowTypesPages(_a0 *swf.ListWorkflowTypesInput, _a1 func(*swf.ListWorkflowTypesOutput, bool) bool) *mock.Mock {
	return m.Mock.On("ListWorkflowTypesPages", _a0, _a1)
}
func (m *SWFAPI) MockOnAny_ListWorkflowTypesPages() *mock.Mock {
	return m.Mock.On("ListWorkflowTypesPages", mock.Anything, mock.Anything)
}
func (m *SWFAPI) ListWorkflowTypesPages(_a0 *swf.ListWorkflowTypesInput, _a1 func(*swf.ListWorkflowTypesOutput, bool) bool) error {
	ret := m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(*swf.ListWorkflowTypesInput, func(*swf.ListWorkflowTypesOutput, bool) bool) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (m *SWFAPI) Name_PollForActivityTaskRequest() string {
	return "PollForActivityTaskRequest"
}
func (m *SWFAPI) MockOn_PollForActivityTaskRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("PollForActivityTaskRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_PollForActivityTaskRequest(_a0 *swf.PollForActivityTaskInput) *mock.Mock {
	return m.Mock.On("PollForActivityTaskRequest", _a0)
}
func (m *SWFAPI) MockOnAny_PollForActivityTaskRequest() *mock.Mock {
	return m.Mock.On("PollForActivityTaskRequest", mock.Anything)
}
func (m *SWFAPI) PollForActivityTaskRequest(_a0 *swf.PollForActivityTaskInput) (*request.Request, *swf.PollForActivityTaskOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.PollForActivityTaskInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.PollForActivityTaskOutput
	if rf, ok := ret.Get(1).(func(*swf.PollForActivityTaskInput) *swf.PollForActivityTaskOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.PollForActivityTaskOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_PollForActivityTask() string {
	return "PollForActivityTask"
}
func (m *SWFAPI) MockOn_PollForActivityTask(_a0 interface{}) *mock.Mock {
	return m.Mock.On("PollForActivityTask", _a0)
}
func (m *SWFAPI) MockOnTyped_PollForActivityTask(_a0 *swf.PollForActivityTaskInput) *mock.Mock {
	return m.Mock.On("PollForActivityTask", _a0)
}
func (m *SWFAPI) MockOnAny_PollForActivityTask() *mock.Mock {
	return m.Mock.On("PollForActivityTask", mock.Anything)
}
func (m *SWFAPI) PollForActivityTask(_a0 *swf.PollForActivityTaskInput) (*swf.PollForActivityTaskOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.PollForActivityTaskOutput
	if rf, ok := ret.Get(0).(func(*swf.PollForActivityTaskInput) *swf.PollForActivityTaskOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.PollForActivityTaskOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.PollForActivityTaskInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_PollForDecisionTaskRequest() string {
	return "PollForDecisionTaskRequest"
}
func (m *SWFAPI) MockOn_PollForDecisionTaskRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("PollForDecisionTaskRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_PollForDecisionTaskRequest(_a0 *swf.PollForDecisionTaskInput) *mock.Mock {
	return m.Mock.On("PollForDecisionTaskRequest", _a0)
}
func (m *SWFAPI) MockOnAny_PollForDecisionTaskRequest() *mock.Mock {
	return m.Mock.On("PollForDecisionTaskRequest", mock.Anything)
}
func (m *SWFAPI) PollForDecisionTaskRequest(_a0 *swf.PollForDecisionTaskInput) (*request.Request, *swf.PollForDecisionTaskOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.PollForDecisionTaskInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.PollForDecisionTaskOutput
	if rf, ok := ret.Get(1).(func(*swf.PollForDecisionTaskInput) *swf.PollForDecisionTaskOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.PollForDecisionTaskOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_PollForDecisionTask() string {
	return "PollForDecisionTask"
}
func (m *SWFAPI) MockOn_PollForDecisionTask(_a0 interface{}) *mock.Mock {
	return m.Mock.On("PollForDecisionTask", _a0)
}
func (m *SWFAPI) MockOnTyped_PollForDecisionTask(_a0 *swf.PollForDecisionTaskInput) *mock.Mock {
	return m.Mock.On("PollForDecisionTask", _a0)
}
func (m *SWFAPI) MockOnAny_PollForDecisionTask() *mock.Mock {
	return m.Mock.On("PollForDecisionTask", mock.Anything)
}
func (m *SWFAPI) PollForDecisionTask(_a0 *swf.PollForDecisionTaskInput) (*swf.PollForDecisionTaskOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.PollForDecisionTaskOutput
	if rf, ok := ret.Get(0).(func(*swf.PollForDecisionTaskInput) *swf.PollForDecisionTaskOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.PollForDecisionTaskOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.PollForDecisionTaskInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_PollForDecisionTaskPages() string {
	return "PollForDecisionTaskPages"
}
func (m *SWFAPI) MockOn_PollForDecisionTaskPages(_a0 interface{}, _a1 interface{}) *mock.Mock {
	return m.Mock.On("PollForDecisionTaskPages", _a0, _a1)
}
func (m *SWFAPI) MockOnTyped_PollForDecisionTaskPages(_a0 *swf.PollForDecisionTaskInput, _a1 func(*swf.PollForDecisionTaskOutput, bool) bool) *mock.Mock {
	return m.Mock.On("PollForDecisionTaskPages", _a0, _a1)
}
func (m *SWFAPI) MockOnAny_PollForDecisionTaskPages() *mock.Mock {
	return m.Mock.On("PollForDecisionTaskPages", mock.Anything, mock.Anything)
}
func (m *SWFAPI) PollForDecisionTaskPages(_a0 *swf.PollForDecisionTaskInput, _a1 func(*swf.PollForDecisionTaskOutput, bool) bool) error {
	ret := m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(*swf.PollForDecisionTaskInput, func(*swf.PollForDecisionTaskOutput, bool) bool) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (m *SWFAPI) Name_RecordActivityTaskHeartbeatRequest() string {
	return "RecordActivityTaskHeartbeatRequest"
}
func (m *SWFAPI) MockOn_RecordActivityTaskHeartbeatRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RecordActivityTaskHeartbeatRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_RecordActivityTaskHeartbeatRequest(_a0 *swf.RecordActivityTaskHeartbeatInput) *mock.Mock {
	return m.Mock.On("RecordActivityTaskHeartbeatRequest", _a0)
}
func (m *SWFAPI) MockOnAny_RecordActivityTaskHeartbeatRequest() *mock.Mock {
	return m.Mock.On("RecordActivityTaskHeartbeatRequest", mock.Anything)
}
func (m *SWFAPI) RecordActivityTaskHeartbeatRequest(_a0 *swf.RecordActivityTaskHeartbeatInput) (*request.Request, *swf.RecordActivityTaskHeartbeatOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.RecordActivityTaskHeartbeatInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.RecordActivityTaskHeartbeatOutput
	if rf, ok := ret.Get(1).(func(*swf.RecordActivityTaskHeartbeatInput) *swf.RecordActivityTaskHeartbeatOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.RecordActivityTaskHeartbeatOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_RecordActivityTaskHeartbeat() string {
	return "RecordActivityTaskHeartbeat"
}
func (m *SWFAPI) MockOn_RecordActivityTaskHeartbeat(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RecordActivityTaskHeartbeat", _a0)
}
func (m *SWFAPI) MockOnTyped_RecordActivityTaskHeartbeat(_a0 *swf.RecordActivityTaskHeartbeatInput) *mock.Mock {
	return m.Mock.On("RecordActivityTaskHeartbeat", _a0)
}
func (m *SWFAPI) MockOnAny_RecordActivityTaskHeartbeat() *mock.Mock {
	return m.Mock.On("RecordActivityTaskHeartbeat", mock.Anything)
}
func (m *SWFAPI) RecordActivityTaskHeartbeat(_a0 *swf.RecordActivityTaskHeartbeatInput) (*swf.RecordActivityTaskHeartbeatOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.RecordActivityTaskHeartbeatOutput
	if rf, ok := ret.Get(0).(func(*swf.RecordActivityTaskHeartbeatInput) *swf.RecordActivityTaskHeartbeatOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.RecordActivityTaskHeartbeatOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.RecordActivityTaskHeartbeatInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_RegisterActivityTypeRequest() string {
	return "RegisterActivityTypeRequest"
}
func (m *SWFAPI) MockOn_RegisterActivityTypeRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RegisterActivityTypeRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_RegisterActivityTypeRequest(_a0 *swf.RegisterActivityTypeInput) *mock.Mock {
	return m.Mock.On("RegisterActivityTypeRequest", _a0)
}
func (m *SWFAPI) MockOnAny_RegisterActivityTypeRequest() *mock.Mock {
	return m.Mock.On("RegisterActivityTypeRequest", mock.Anything)
}
func (m *SWFAPI) RegisterActivityTypeRequest(_a0 *swf.RegisterActivityTypeInput) (*request.Request, *swf.RegisterActivityTypeOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.RegisterActivityTypeInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.RegisterActivityTypeOutput
	if rf, ok := ret.Get(1).(func(*swf.RegisterActivityTypeInput) *swf.RegisterActivityTypeOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.RegisterActivityTypeOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_RegisterActivityType() string {
	return "RegisterActivityType"
}
func (m *SWFAPI) MockOn_RegisterActivityType(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RegisterActivityType", _a0)
}
func (m *SWFAPI) MockOnTyped_RegisterActivityType(_a0 *swf.RegisterActivityTypeInput) *mock.Mock {
	return m.Mock.On("RegisterActivityType", _a0)
}
func (m *SWFAPI) MockOnAny_RegisterActivityType() *mock.Mock {
	return m.Mock.On("RegisterActivityType", mock.Anything)
}
func (m *SWFAPI) RegisterActivityType(_a0 *swf.RegisterActivityTypeInput) (*swf.RegisterActivityTypeOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.RegisterActivityTypeOutput
	if rf, ok := ret.Get(0).(func(*swf.RegisterActivityTypeInput) *swf.RegisterActivityTypeOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.RegisterActivityTypeOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.RegisterActivityTypeInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_RegisterDomainRequest() string {
	return "RegisterDomainRequest"
}
func (m *SWFAPI) MockOn_RegisterDomainRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RegisterDomainRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_RegisterDomainRequest(_a0 *swf.RegisterDomainInput) *mock.Mock {
	return m.Mock.On("RegisterDomainRequest", _a0)
}
func (m *SWFAPI) MockOnAny_RegisterDomainRequest() *mock.Mock {
	return m.Mock.On("RegisterDomainRequest", mock.Anything)
}
func (m *SWFAPI) RegisterDomainRequest(_a0 *swf.RegisterDomainInput) (*request.Request, *swf.RegisterDomainOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.RegisterDomainInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.RegisterDomainOutput
	if rf, ok := ret.Get(1).(func(*swf.RegisterDomainInput) *swf.RegisterDomainOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.RegisterDomainOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_RegisterDomain() string {
	return "RegisterDomain"
}
func (m *SWFAPI) MockOn_RegisterDomain(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RegisterDomain", _a0)
}
func (m *SWFAPI) MockOnTyped_RegisterDomain(_a0 *swf.RegisterDomainInput) *mock.Mock {
	return m.Mock.On("RegisterDomain", _a0)
}
func (m *SWFAPI) MockOnAny_RegisterDomain() *mock.Mock {
	return m.Mock.On("RegisterDomain", mock.Anything)
}
func (m *SWFAPI) RegisterDomain(_a0 *swf.RegisterDomainInput) (*swf.RegisterDomainOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.RegisterDomainOutput
	if rf, ok := ret.Get(0).(func(*swf.RegisterDomainInput) *swf.RegisterDomainOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.RegisterDomainOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.RegisterDomainInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_RegisterWorkflowTypeRequest() string {
	return "RegisterWorkflowTypeRequest"
}
func (m *SWFAPI) MockOn_RegisterWorkflowTypeRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RegisterWorkflowTypeRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_RegisterWorkflowTypeRequest(_a0 *swf.RegisterWorkflowTypeInput) *mock.Mock {
	return m.Mock.On("RegisterWorkflowTypeRequest", _a0)
}
func (m *SWFAPI) MockOnAny_RegisterWorkflowTypeRequest() *mock.Mock {
	return m.Mock.On("RegisterWorkflowTypeRequest", mock.Anything)
}
func (m *SWFAPI) RegisterWorkflowTypeRequest(_a0 *swf.RegisterWorkflowTypeInput) (*request.Request, *swf.RegisterWorkflowTypeOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.RegisterWorkflowTypeInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.RegisterWorkflowTypeOutput
	if rf, ok := ret.Get(1).(func(*swf.RegisterWorkflowTypeInput) *swf.RegisterWorkflowTypeOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.RegisterWorkflowTypeOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_RegisterWorkflowType() string {
	return "RegisterWorkflowType"
}
func (m *SWFAPI) MockOn_RegisterWorkflowType(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RegisterWorkflowType", _a0)
}
func (m *SWFAPI) MockOnTyped_RegisterWorkflowType(_a0 *swf.RegisterWorkflowTypeInput) *mock.Mock {
	return m.Mock.On("RegisterWorkflowType", _a0)
}
func (m *SWFAPI) MockOnAny_RegisterWorkflowType() *mock.Mock {
	return m.Mock.On("RegisterWorkflowType", mock.Anything)
}
func (m *SWFAPI) RegisterWorkflowType(_a0 *swf.RegisterWorkflowTypeInput) (*swf.RegisterWorkflowTypeOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.RegisterWorkflowTypeOutput
	if rf, ok := ret.Get(0).(func(*swf.RegisterWorkflowTypeInput) *swf.RegisterWorkflowTypeOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.RegisterWorkflowTypeOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.RegisterWorkflowTypeInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_RequestCancelWorkflowExecutionRequest() string {
	return "RequestCancelWorkflowExecutionRequest"
}
func (m *SWFAPI) MockOn_RequestCancelWorkflowExecutionRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RequestCancelWorkflowExecutionRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_RequestCancelWorkflowExecutionRequest(_a0 *swf.RequestCancelWorkflowExecutionInput) *mock.Mock {
	return m.Mock.On("RequestCancelWorkflowExecutionRequest", _a0)
}
func (m *SWFAPI) MockOnAny_RequestCancelWorkflowExecutionRequest() *mock.Mock {
	return m.Mock.On("RequestCancelWorkflowExecutionRequest", mock.Anything)
}
func (m *SWFAPI) RequestCancelWorkflowExecutionRequest(_a0 *swf.RequestCancelWorkflowExecutionInput) (*request.Request, *swf.RequestCancelWorkflowExecutionOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.RequestCancelWorkflowExecutionInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.RequestCancelWorkflowExecutionOutput
	if rf, ok := ret.Get(1).(func(*swf.RequestCancelWorkflowExecutionInput) *swf.RequestCancelWorkflowExecutionOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.RequestCancelWorkflowExecutionOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_RequestCancelWorkflowExecution() string {
	return "RequestCancelWorkflowExecution"
}
func (m *SWFAPI) MockOn_RequestCancelWorkflowExecution(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RequestCancelWorkflowExecution", _a0)
}
func (m *SWFAPI) MockOnTyped_RequestCancelWorkflowExecution(_a0 *swf.RequestCancelWorkflowExecutionInput) *mock.Mock {
	return m.Mock.On("RequestCancelWorkflowExecution", _a0)
}
func (m *SWFAPI) MockOnAny_RequestCancelWorkflowExecution() *mock.Mock {
	return m.Mock.On("RequestCancelWorkflowExecution", mock.Anything)
}
func (m *SWFAPI) RequestCancelWorkflowExecution(_a0 *swf.RequestCancelWorkflowExecutionInput) (*swf.RequestCancelWorkflowExecutionOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.RequestCancelWorkflowExecutionOutput
	if rf, ok := ret.Get(0).(func(*swf.RequestCancelWorkflowExecutionInput) *swf.RequestCancelWorkflowExecutionOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.RequestCancelWorkflowExecutionOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.RequestCancelWorkflowExecutionInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_RespondActivityTaskCanceledRequest() string {
	return "RespondActivityTaskCanceledRequest"
}
func (m *SWFAPI) MockOn_RespondActivityTaskCanceledRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RespondActivityTaskCanceledRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_RespondActivityTaskCanceledRequest(_a0 *swf.RespondActivityTaskCanceledInput) *mock.Mock {
	return m.Mock.On("RespondActivityTaskCanceledRequest", _a0)
}
func (m *SWFAPI) MockOnAny_RespondActivityTaskCanceledRequest() *mock.Mock {
	return m.Mock.On("RespondActivityTaskCanceledRequest", mock.Anything)
}
func (m *SWFAPI) RespondActivityTaskCanceledRequest(_a0 *swf.RespondActivityTaskCanceledInput) (*request.Request, *swf.RespondActivityTaskCanceledOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.RespondActivityTaskCanceledInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.RespondActivityTaskCanceledOutput
	if rf, ok := ret.Get(1).(func(*swf.RespondActivityTaskCanceledInput) *swf.RespondActivityTaskCanceledOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.RespondActivityTaskCanceledOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_RespondActivityTaskCanceled() string {
	return "RespondActivityTaskCanceled"
}
func (m *SWFAPI) MockOn_RespondActivityTaskCanceled(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RespondActivityTaskCanceled", _a0)
}
func (m *SWFAPI) MockOnTyped_RespondActivityTaskCanceled(_a0 *swf.RespondActivityTaskCanceledInput) *mock.Mock {
	return m.Mock.On("RespondActivityTaskCanceled", _a0)
}
func (m *SWFAPI) MockOnAny_RespondActivityTaskCanceled() *mock.Mock {
	return m.Mock.On("RespondActivityTaskCanceled", mock.Anything)
}
func (m *SWFAPI) RespondActivityTaskCanceled(_a0 *swf.RespondActivityTaskCanceledInput) (*swf.RespondActivityTaskCanceledOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.RespondActivityTaskCanceledOutput
	if rf, ok := ret.Get(0).(func(*swf.RespondActivityTaskCanceledInput) *swf.RespondActivityTaskCanceledOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.RespondActivityTaskCanceledOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.RespondActivityTaskCanceledInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_RespondActivityTaskCompletedRequest() string {
	return "RespondActivityTaskCompletedRequest"
}
func (m *SWFAPI) MockOn_RespondActivityTaskCompletedRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RespondActivityTaskCompletedRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_RespondActivityTaskCompletedRequest(_a0 *swf.RespondActivityTaskCompletedInput) *mock.Mock {
	return m.Mock.On("RespondActivityTaskCompletedRequest", _a0)
}
func (m *SWFAPI) MockOnAny_RespondActivityTaskCompletedRequest() *mock.Mock {
	return m.Mock.On("RespondActivityTaskCompletedRequest", mock.Anything)
}
func (m *SWFAPI) RespondActivityTaskCompletedRequest(_a0 *swf.RespondActivityTaskCompletedInput) (*request.Request, *swf.RespondActivityTaskCompletedOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.RespondActivityTaskCompletedInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.RespondActivityTaskCompletedOutput
	if rf, ok := ret.Get(1).(func(*swf.RespondActivityTaskCompletedInput) *swf.RespondActivityTaskCompletedOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.RespondActivityTaskCompletedOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_RespondActivityTaskCompleted() string {
	return "RespondActivityTaskCompleted"
}
func (m *SWFAPI) MockOn_RespondActivityTaskCompleted(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RespondActivityTaskCompleted", _a0)
}
func (m *SWFAPI) MockOnTyped_RespondActivityTaskCompleted(_a0 *swf.RespondActivityTaskCompletedInput) *mock.Mock {
	return m.Mock.On("RespondActivityTaskCompleted", _a0)
}
func (m *SWFAPI) MockOnAny_RespondActivityTaskCompleted() *mock.Mock {
	return m.Mock.On("RespondActivityTaskCompleted", mock.Anything)
}
func (m *SWFAPI) RespondActivityTaskCompleted(_a0 *swf.RespondActivityTaskCompletedInput) (*swf.RespondActivityTaskCompletedOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.RespondActivityTaskCompletedOutput
	if rf, ok := ret.Get(0).(func(*swf.RespondActivityTaskCompletedInput) *swf.RespondActivityTaskCompletedOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.RespondActivityTaskCompletedOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.RespondActivityTaskCompletedInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_RespondActivityTaskFailedRequest() string {
	return "RespondActivityTaskFailedRequest"
}
func (m *SWFAPI) MockOn_RespondActivityTaskFailedRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RespondActivityTaskFailedRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_RespondActivityTaskFailedRequest(_a0 *swf.RespondActivityTaskFailedInput) *mock.Mock {
	return m.Mock.On("RespondActivityTaskFailedRequest", _a0)
}
func (m *SWFAPI) MockOnAny_RespondActivityTaskFailedRequest() *mock.Mock {
	return m.Mock.On("RespondActivityTaskFailedRequest", mock.Anything)
}
func (m *SWFAPI) RespondActivityTaskFailedRequest(_a0 *swf.RespondActivityTaskFailedInput) (*request.Request, *swf.RespondActivityTaskFailedOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.RespondActivityTaskFailedInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.RespondActivityTaskFailedOutput
	if rf, ok := ret.Get(1).(func(*swf.RespondActivityTaskFailedInput) *swf.RespondActivityTaskFailedOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.RespondActivityTaskFailedOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_RespondActivityTaskFailed() string {
	return "RespondActivityTaskFailed"
}
func (m *SWFAPI) MockOn_RespondActivityTaskFailed(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RespondActivityTaskFailed", _a0)
}
func (m *SWFAPI) MockOnTyped_RespondActivityTaskFailed(_a0 *swf.RespondActivityTaskFailedInput) *mock.Mock {
	return m.Mock.On("RespondActivityTaskFailed", _a0)
}
func (m *SWFAPI) MockOnAny_RespondActivityTaskFailed() *mock.Mock {
	return m.Mock.On("RespondActivityTaskFailed", mock.Anything)
}
func (m *SWFAPI) RespondActivityTaskFailed(_a0 *swf.RespondActivityTaskFailedInput) (*swf.RespondActivityTaskFailedOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.RespondActivityTaskFailedOutput
	if rf, ok := ret.Get(0).(func(*swf.RespondActivityTaskFailedInput) *swf.RespondActivityTaskFailedOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.RespondActivityTaskFailedOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.RespondActivityTaskFailedInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_RespondDecisionTaskCompletedRequest() string {
	return "RespondDecisionTaskCompletedRequest"
}
func (m *SWFAPI) MockOn_RespondDecisionTaskCompletedRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RespondDecisionTaskCompletedRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_RespondDecisionTaskCompletedRequest(_a0 *swf.RespondDecisionTaskCompletedInput) *mock.Mock {
	return m.Mock.On("RespondDecisionTaskCompletedRequest", _a0)
}
func (m *SWFAPI) MockOnAny_RespondDecisionTaskCompletedRequest() *mock.Mock {
	return m.Mock.On("RespondDecisionTaskCompletedRequest", mock.Anything)
}
func (m *SWFAPI) RespondDecisionTaskCompletedRequest(_a0 *swf.RespondDecisionTaskCompletedInput) (*request.Request, *swf.RespondDecisionTaskCompletedOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.RespondDecisionTaskCompletedInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.RespondDecisionTaskCompletedOutput
	if rf, ok := ret.Get(1).(func(*swf.RespondDecisionTaskCompletedInput) *swf.RespondDecisionTaskCompletedOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.RespondDecisionTaskCompletedOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_RespondDecisionTaskCompleted() string {
	return "RespondDecisionTaskCompleted"
}
func (m *SWFAPI) MockOn_RespondDecisionTaskCompleted(_a0 interface{}) *mock.Mock {
	return m.Mock.On("RespondDecisionTaskCompleted", _a0)
}
func (m *SWFAPI) MockOnTyped_RespondDecisionTaskCompleted(_a0 *swf.RespondDecisionTaskCompletedInput) *mock.Mock {
	return m.Mock.On("RespondDecisionTaskCompleted", _a0)
}
func (m *SWFAPI) MockOnAny_RespondDecisionTaskCompleted() *mock.Mock {
	return m.Mock.On("RespondDecisionTaskCompleted", mock.Anything)
}
func (m *SWFAPI) RespondDecisionTaskCompleted(_a0 *swf.RespondDecisionTaskCompletedInput) (*swf.RespondDecisionTaskCompletedOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.RespondDecisionTaskCompletedOutput
	if rf, ok := ret.Get(0).(func(*swf.RespondDecisionTaskCompletedInput) *swf.RespondDecisionTaskCompletedOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.RespondDecisionTaskCompletedOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.RespondDecisionTaskCompletedInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_SignalWorkflowExecutionRequest() string {
	return "SignalWorkflowExecutionRequest"
}
func (m *SWFAPI) MockOn_SignalWorkflowExecutionRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("SignalWorkflowExecutionRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_SignalWorkflowExecutionRequest(_a0 *swf.SignalWorkflowExecutionInput) *mock.Mock {
	return m.Mock.On("SignalWorkflowExecutionRequest", _a0)
}
func (m *SWFAPI) MockOnAny_SignalWorkflowExecutionRequest() *mock.Mock {
	return m.Mock.On("SignalWorkflowExecutionRequest", mock.Anything)
}
func (m *SWFAPI) SignalWorkflowExecutionRequest(_a0 *swf.SignalWorkflowExecutionInput) (*request.Request, *swf.SignalWorkflowExecutionOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.SignalWorkflowExecutionInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.SignalWorkflowExecutionOutput
	if rf, ok := ret.Get(1).(func(*swf.SignalWorkflowExecutionInput) *swf.SignalWorkflowExecutionOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.SignalWorkflowExecutionOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_SignalWorkflowExecution() string {
	return "SignalWorkflowExecution"
}
func (m *SWFAPI) MockOn_SignalWorkflowExecution(_a0 interface{}) *mock.Mock {
	return m.Mock.On("SignalWorkflowExecution", _a0)
}
func (m *SWFAPI) MockOnTyped_SignalWorkflowExecution(_a0 *swf.SignalWorkflowExecutionInput) *mock.Mock {
	return m.Mock.On("SignalWorkflowExecution", _a0)
}
func (m *SWFAPI) MockOnAny_SignalWorkflowExecution() *mock.Mock {
	return m.Mock.On("SignalWorkflowExecution", mock.Anything)
}
func (m *SWFAPI) SignalWorkflowExecution(_a0 *swf.SignalWorkflowExecutionInput) (*swf.SignalWorkflowExecutionOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.SignalWorkflowExecutionOutput
	if rf, ok := ret.Get(0).(func(*swf.SignalWorkflowExecutionInput) *swf.SignalWorkflowExecutionOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.SignalWorkflowExecutionOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.SignalWorkflowExecutionInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_StartWorkflowExecutionRequest() string {
	return "StartWorkflowExecutionRequest"
}
func (m *SWFAPI) MockOn_StartWorkflowExecutionRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("StartWorkflowExecutionRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_StartWorkflowExecutionRequest(_a0 *swf.StartWorkflowExecutionInput) *mock.Mock {
	return m.Mock.On("StartWorkflowExecutionRequest", _a0)
}
func (m *SWFAPI) MockOnAny_StartWorkflowExecutionRequest() *mock.Mock {
	return m.Mock.On("StartWorkflowExecutionRequest", mock.Anything)
}
func (m *SWFAPI) StartWorkflowExecutionRequest(_a0 *swf.StartWorkflowExecutionInput) (*request.Request, *swf.StartWorkflowExecutionOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.StartWorkflowExecutionInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.StartWorkflowExecutionOutput
	if rf, ok := ret.Get(1).(func(*swf.StartWorkflowExecutionInput) *swf.StartWorkflowExecutionOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.StartWorkflowExecutionOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_StartWorkflowExecution() string {
	return "StartWorkflowExecution"
}
func (m *SWFAPI) MockOn_StartWorkflowExecution(_a0 interface{}) *mock.Mock {
	return m.Mock.On("StartWorkflowExecution", _a0)
}
func (m *SWFAPI) MockOnTyped_StartWorkflowExecution(_a0 *swf.StartWorkflowExecutionInput) *mock.Mock {
	return m.Mock.On("StartWorkflowExecution", _a0)
}
func (m *SWFAPI) MockOnAny_StartWorkflowExecution() *mock.Mock {
	return m.Mock.On("StartWorkflowExecution", mock.Anything)
}
func (m *SWFAPI) StartWorkflowExecution(_a0 *swf.StartWorkflowExecutionInput) (*swf.StartWorkflowExecutionOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.StartWorkflowExecutionOutput
	if rf, ok := ret.Get(0).(func(*swf.StartWorkflowExecutionInput) *swf.StartWorkflowExecutionOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.StartWorkflowExecutionOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.StartWorkflowExecutionInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (m *SWFAPI) Name_TerminateWorkflowExecutionRequest() string {
	return "TerminateWorkflowExecutionRequest"
}
func (m *SWFAPI) MockOn_TerminateWorkflowExecutionRequest(_a0 interface{}) *mock.Mock {
	return m.Mock.On("TerminateWorkflowExecutionRequest", _a0)
}
func (m *SWFAPI) MockOnTyped_TerminateWorkflowExecutionRequest(_a0 *swf.TerminateWorkflowExecutionInput) *mock.Mock {
	return m.Mock.On("TerminateWorkflowExecutionRequest", _a0)
}
func (m *SWFAPI) MockOnAny_TerminateWorkflowExecutionRequest() *mock.Mock {
	return m.Mock.On("TerminateWorkflowExecutionRequest", mock.Anything)
}
func (m *SWFAPI) TerminateWorkflowExecutionRequest(_a0 *swf.TerminateWorkflowExecutionInput) (*request.Request, *swf.TerminateWorkflowExecutionOutput) {
	ret := m.Called(_a0)

	var r0 *request.Request
	if rf, ok := ret.Get(0).(func(*swf.TerminateWorkflowExecutionInput) *request.Request); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*request.Request)
		}
	}

	var r1 *swf.TerminateWorkflowExecutionOutput
	if rf, ok := ret.Get(1).(func(*swf.TerminateWorkflowExecutionInput) *swf.TerminateWorkflowExecutionOutput); ok {
		r1 = rf(_a0)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(*swf.TerminateWorkflowExecutionOutput)
		}
	}

	return r0, r1
}
func (m *SWFAPI) Name_TerminateWorkflowExecution() string {
	return "TerminateWorkflowExecution"
}
func (m *SWFAPI) MockOn_TerminateWorkflowExecution(_a0 interface{}) *mock.Mock {
	return m.Mock.On("TerminateWorkflowExecution", _a0)
}
func (m *SWFAPI) MockOnTyped_TerminateWorkflowExecution(_a0 *swf.TerminateWorkflowExecutionInput) *mock.Mock {
	return m.Mock.On("TerminateWorkflowExecution", _a0)
}
func (m *SWFAPI) MockOnAny_TerminateWorkflowExecution() *mock.Mock {
	return m.Mock.On("TerminateWorkflowExecution", mock.Anything)
}
func (m *SWFAPI) TerminateWorkflowExecution(_a0 *swf.TerminateWorkflowExecutionInput) (*swf.TerminateWorkflowExecutionOutput, error) {
	ret := m.Called(_a0)

	var r0 *swf.TerminateWorkflowExecutionOutput
	if rf, ok := ret.Get(0).(func(*swf.TerminateWorkflowExecutionInput) *swf.TerminateWorkflowExecutionOutput); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*swf.TerminateWorkflowExecutionOutput)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*swf.TerminateWorkflowExecutionInput) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
