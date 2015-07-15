package activity

import (
	"github.com/aws/aws-sdk-go/service/swf"
)

//ActivityInterceptor allows manipulation of the decision task and the outcome at key points in the task lifecycle.
type ActivityInterceptor interface {
	BeforeTask(*swf.PollForActivityTaskOutput)
	AfterTask(t *swf.PollForActivityTaskOutput, result interface{}, err error) (interface{}, error)
	AfterTaskComplete(t *swf.PollForActivityTaskOutput, result interface{})
	AfterTaskFailed(t *swf.PollForActivityTaskOutput, err error)
	AfterTaskCanceled(t *swf.PollForActivityTaskOutput, details string)
}

//FuncInterceptor is a ActivityInterceptor that you can set handler funcs on. if any are unset, they are no-ops.
type FuncInterceptor struct {
	BeforeTaskFn        func(*swf.PollForActivityTaskOutput)
	AfterTaskFn         func(t *swf.PollForActivityTaskOutput, result interface{}, err error) (interface{}, error)
	AfterTaskCompleteFn func(t *swf.PollForActivityTaskOutput, result interface{})
	AfterTaskFailedFn   func(t *swf.PollForActivityTaskOutput, err error)
	AfterTaskCanceledFn func(t *swf.PollForActivityTaskOutput, details string)
}

//BeforeTask runs the BeforeTaskFn if not nil
func (i *FuncInterceptor) BeforeTask(activity *swf.PollForActivityTaskOutput) {
	if i.BeforeTaskFn != nil {
		i.BeforeTaskFn(activity)
	}
}

func (i *FuncInterceptor) AfterTask(activity *swf.PollForActivityTaskOutput, result interface{}, err error) (interface{}, error) {
	if i.AfterTaskFn != nil {
		return i.AfterTaskFn(activity, result, err)
	}
	return result, err
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

//AfterTaskCanceled runs the AfterTaskCanceledFn if not nil
func (i *FuncInterceptor) AfterTaskCanceled(activity *swf.PollForActivityTaskOutput, details string) {
	if i.AfterTaskCanceledFn != nil {
		i.AfterTaskCanceledFn(activity, details)
	}
}

type ComposedDecisionInterceptor struct {
	interceptors []ActivityInterceptor
}

func NewComposedDecisionInterceptor(interceptors ...ActivityInterceptor) ActivityInterceptor {
	c := &ComposedDecisionInterceptor{}
	for _, i := range interceptors {
		if i != nil {
			c.interceptors = append(c.interceptors, i)
		}
	}
	return c
}

func (c *ComposedDecisionInterceptor) BeforeTask(t *swf.PollForActivityTaskOutput) {
	for _, i := range c.interceptors {
		i.BeforeTask(t)
	}
}

func (c *ComposedDecisionInterceptor) AfterTask(t *swf.PollForActivityTaskOutput, result interface{}, err error) (interface{}, error) {
	for _, i := range c.interceptors {
		result, err = i.AfterTask(t, result, err)
	}
	return result, err
}

func (c *ComposedDecisionInterceptor) AfterTaskComplete(t *swf.PollForActivityTaskOutput, result interface{}) {
	for _, i := range c.interceptors {
		i.AfterTaskComplete(t, result)
	}
}

func (c *ComposedDecisionInterceptor) AfterTaskFailed(t *swf.PollForActivityTaskOutput, err error) {
	for _, i := range c.interceptors {
		i.AfterTaskFailed(t, err)
	}
}

func (c *ComposedDecisionInterceptor) AfterTaskCanceled(t *swf.PollForActivityTaskOutput, details string) {
	for _, i := range c.interceptors {
		i.AfterTaskCanceled(t, details)
	}
}
