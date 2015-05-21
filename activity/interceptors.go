package activity

import (
	"github.com/awslabs/aws-sdk-go/service/swf"
)

//ActivityInterceptor allows manipulation of the decision task and the outcome at key points in the task lifecycle.
type ActivityInterceptor interface {
	BeforeTask(decision *swf.PollForActivityTaskOutput)
	AfterTaskComplete(decision *swf.PollForActivityTaskOutput, result interface{})
	AfterTaskFailed(decision *swf.PollForActivityTaskOutput, err error)
}

//FuncInterceptor is a ActivityInterceptor that you can set handler funcs on. if any are unset, they are no-ops.
type FuncInterceptor struct {
	BeforeTaskFn        func(decision *swf.PollForActivityTaskOutput)
	AfterTaskCompleteFn func(decision *swf.PollForActivityTaskOutput, result interface{})
	AfterTaskFailedFn   func(decision *swf.PollForActivityTaskOutput, err error)
}

//BeforeTask runs the BeforeTaskFn if not nil
func (i *FuncInterceptor) BeforeTask(activity *swf.PollForActivityTaskOutput) {
	if i.BeforeTaskFn != nil {
		i.BeforeTaskFn(activity)
	}
}

//AfterTaskComplete runs the AfterTaskCompleteFn if not nil
func (i *FuncInterceptor) AfterTaskComplete(activity *swf.PollForActivityTaskOutput, result interface{}) {
	if i.AfterTaskCompleteFn != nil {
		i.AfterTaskCompleteFn(activity, result)
	}
}

//AfterTaskFailed runs the AfterTaskFailedFn if not nil
func (i *FuncInterceptor) AfterTaskFailed(activity *swf.PollForActivityTaskOutput, err error) {
	if i.AfterTaskFailedFn != nil {
		i.AfterTaskFailedFn(activity, err)
	}
}
