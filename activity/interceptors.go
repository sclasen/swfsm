package activity

import (
	"github.com/awslabs/aws-sdk-go/gen/swf"
)

//ActivityInterceptor allows manipulation of the decision task and the outcome at key points in the task lifecycle.
type ActivityInterceptor interface {
	BeforeTask(*swf.ActivityTask)
	AfterTaskComplete(t *swf.ActivityTask, result interface{})
	AfterTaskFailed(t *swf.ActivityTask, err error)
	AfterTaskCanceled(t *swf.ActivityTask, details string)
}

//FuncInterceptor is a ActivityInterceptor that you can set handler funcs on. if any are unset, they are no-ops.
type FuncInterceptor struct {
	BeforeTaskFn        func(*swf.ActivityTask)
	AfterTaskCompleteFn func(t *swf.ActivityTask, result interface{})
	AfterTaskFailedFn   func(t *swf.ActivityTask, err error)
	AfterTaskCanceledFn func(t *swf.ActivityTask, details string)
}

//BeforeTask runs the BeforeTaskFn if not nil
func (i *FuncInterceptor) BeforeTask(activity *swf.ActivityTask) {
	if i.BeforeTaskFn != nil {
		i.BeforeTaskFn(activity)
	}
}

//AfterTaskComplete runs the AfterTaskCompleteFn if not nil
func (i *FuncInterceptor) AfterTaskComplete(activity *swf.ActivityTask, result interface{}) {
	if i.AfterTaskCompleteFn != nil {
		i.AfterTaskCompleteFn(activity, result)
	}
}

//AfterTaskFailed runs the AfterTaskFailedFn if not nil
func (i *FuncInterceptor) AfterTaskFailed(activity *swf.ActivityTask, err error) {
	if i.AfterTaskFailedFn != nil {
		i.AfterTaskFailedFn(activity, err)
	}
}

//AfterTaskCanceled runs the AfterTaskCanceledFn if not nil
func (i *FuncInterceptor) AfterTaskCanceled(activity *swf.ActivityTask, details string) {
	if i.AfterTaskCanceledFn != nil {
		i.AfterTaskCanceledFn(activity, details)
	}
}
