package activity

//ActivityInterceptor allows manipulation of the decision task and the outcome at key points in the task lifecycle.
type ActivityInterceptor interface {
	BeforeTask(decision *ActivityContext)
	AfterTaskComplete(decision *ActivityContext, result interface{})
	AfterTaskFailed(decision *ActivityContext, err error)
}

//FuncInterceptor is a ActivityInterceptor that you can set handler funcs on. if any are unset, they are no-ops.
type FuncInterceptor struct {
	BeforeTaskFn        func(decision *ActivityContext)
	AfterTaskCompleteFn func(decision *ActivityContext, result interface{})
	AfterTaskFailedFn   func(decision *ActivityContext, err error)
}

//BeforeTask runs the BeforeTaskFn if not nil
func (i *FuncInterceptor) BeforeTask(activity *ActivityContext) {
	if i.BeforeTaskFn != nil {
		i.BeforeTaskFn(activity)
	}
}

//AfterTaskComplete runs the AfterTaskCompleteFn if not nil
func (i *FuncInterceptor) AfterTaskComplete(activity *ActivityContext, result interface{}) {
	if i.AfterTaskCompleteFn != nil {
		i.AfterTaskCompleteFn(activity, result)
	}
}

//AfterTaskFailed runs the AfterTaskFailedFn if not nil
func (i *FuncInterceptor) AfterTaskFailed(activity *ActivityContext, err error) {
	if i.AfterTaskFailedFn != nil {
		i.AfterTaskFailedFn(activity, err)
	}
}
