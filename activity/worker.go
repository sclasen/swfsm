package activity

import (
	"fmt"
	"log"

	"time"

	"math"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/swf"
	"github.com/juju/errors"
	"github.com/sclasen/swfsm/fsm"
	"github.com/sclasen/swfsm/poller"
	. "github.com/sclasen/swfsm/sugar"
)

type SWFOps interface {
	RecordActivityTaskHeartbeat(req *swf.RecordActivityTaskHeartbeatInput) (resp *swf.ActivityTaskStatus, err error)
	RespondActivityTaskCanceled(req *swf.RespondActivityTaskCanceledInput) (err error)
	RespondActivityTaskCompleted(req *swf.RespondActivityTaskCompletedInput) (err error)
	RespondActivityTaskFailed(req *swf.RespondActivityTaskFailedInput) (err error)
	PollForActivityTask(req *swf.PollForActivityTaskInput) (resp *swf.ActivityTask, err error)
	GetWorkflowExecutionHistory(req *swf.GetWorkflowExecutionHistoryInput) (resp *swf.History, err error)
	SignalWorkflowExecution(req *swf.SignalWorkflowExecutionInput) (err error)
}

type ActivityContext struct {
	Task    *swf.ActivityTask
	Resumed bool
}

type ActivityWorker struct {
	Serializer       fsm.StateSerializer
	systemSerializer fsm.StateSerializer
	// Domain of the workflow associated with the FSM.
	Domain string
	// TaskList that the underlying poller will poll for decision tasks.
	TaskList string
	// Identity used in PollForActivityTaskRequests, can be empty.
	Identity string
	// Client used to make SWF api requests.
	SWF SWFOps
	// Type Info for handled activities
	handlers            map[string]*ActivityHandler
	longRunningHandlers map[string]*LongRunningActivityHandler
	// ShutdownManager
	ShutdownManager *poller.ShutdownManager
	// ActivityTaskDispatcher
	ActivityTaskDispatcher ActivityTaskDispatcher
	// ActivityInterceptor
	ActivityInterceptor ActivityInterceptor
	// allow panics in activities rather than recovering and failing the activity, useful for testing
	AllowPanics bool
	// reads the EventCorrelator and backs off based on what retry # the activity is.
	BackoffOnFailure bool
	// maximum backoff sleep on retries that fail.
	MaxBackoffSeconds int
}

func (a *ActivityWorker) AddHandler(handler *ActivityHandler) {
	if a.handlers == nil {
		a.handlers = map[string]*ActivityHandler{}
	}
	a.handlers[handler.Activity] = handler
}

func (a *ActivityWorker) AddLongRunningHandler(handler *LongRunningActivityHandler) {
	if a.longRunningHandlers == nil {
		a.longRunningHandlers = map[string]*LongRunningActivityHandler{}
	}
	a.longRunningHandlers[handler.Activity] = handler
}

func (a *ActivityWorker) Init() {
	if a.Serializer == nil {
		a.Serializer = fsm.JSONStateSerializer{}
	}

	if a.systemSerializer == nil {
		a.systemSerializer = fsm.JSONStateSerializer{}
	}

	if a.ActivityInterceptor == nil {
		a.ActivityInterceptor = &FuncInterceptor{}
	}

	if a.ActivityTaskDispatcher == nil {
		a.ActivityTaskDispatcher = &CallingGoroutineDispatcher{}
	}

	if a.ShutdownManager == nil {
		a.ShutdownManager = poller.NewShutdownManager()
	}
}

func (a *ActivityWorker) Start() {
	a.Init()
	poller := poller.NewActivityTaskPoller(a.SWF, a.Domain, a.Identity, a.TaskList)
	go poller.PollUntilShutdownBy(a.ShutdownManager, fmt.Sprintf("%s-poller", a.Identity), a.Begin)
}

func (a *ActivityWorker) Resume(activityTask *swf.ActivityTask) {
	ctx := &ActivityContext{
		Task:    activityTask,
		Resumed: true,
	}
	a.dispatchTask(ctx)
}

func (a *ActivityWorker) Begin(activityTask *swf.ActivityTask) {
	ctx := &ActivityContext{
		Task:    activityTask,
		Resumed: false,
	}
	a.dispatchTask(ctx)
}

func (a *ActivityWorker) dispatchTask(ctx *ActivityContext) {
	if a.AllowPanics {
		a.ActivityTaskDispatcher.DispatchTask(ctx, a.handleActivityTask)
	} else {
		a.ActivityTaskDispatcher.DispatchTask(ctx, a.handleWithRecovery(a.handleActivityTask))
	}
}

func (a *ActivityWorker) handleActivityTask(ctx *ActivityContext) {
	activityTask := ctx.Task
	a.ActivityInterceptor.BeforeTask(ctx)
	handler := a.handlers[*activityTask.ActivityType.Name]
	longHandler := a.longRunningHandlers[*activityTask.ActivityType.Name]

	if handler != nil {
		var deserialized interface{}
		if activityTask.Input != nil {
			switch handler.Input.(type) {
			case string:
				deserialized = *activityTask.Input
			default:
				deserialized = handler.ZeroInput()
				err := a.Serializer.Deserialize(*activityTask.Input, deserialized)
				if err != nil {
					a.ActivityInterceptor.AfterTaskFailed(ctx, err)
					a.fail(ctx, errors.Annotate(err, "deserialize"))
					return
				}
			}

		} else {
			deserialized = nil
		}

		result, err := handler.HandlerFunc(ctx, deserialized)
		if err != nil {
			a.ActivityInterceptor.AfterTaskFailed(ctx, err)
			a.fail(ctx, errors.Annotate(err, "handler"))
		} else {
			a.result(ctx, result)
		}
	} else if longHandler != nil {
		var deserialized interface{}
		if activityTask.Input != nil {
			switch longHandler.Input.(type) {
			case string:
				deserialized = *activityTask.Input
			default:
				deserialized = longHandler.ZeroInput()
				err := a.Serializer.Deserialize(*activityTask.Input, deserialized)
				if err != nil {
					a.ActivityInterceptor.AfterTaskFailed(ctx, err)
					a.fail(ctx, errors.Annotate(err, "deserialize"))
					return
				}
			}

		} else {
			deserialized = nil
		}

		longHandler.HandlerFunc(ctx, deserialized)

	} else {
		//fail
		err := errors.NewErr("no handler for activity: %s", LS(activityTask.ActivityType.Name))
		a.ActivityInterceptor.AfterTaskFailed(ctx, &err)
		a.fail(ctx, &err)
	}
}

func (a *ActivityWorker) result(ctx *ActivityContext, result interface{}) {
	a.ActivityInterceptor.AfterTaskComplete(ctx, result)
	switch t := result.(type) {
	case string:
		a.done(ctx, &t)
	case nil:
		a.done(ctx, nil)
	default:
		serialized, err := a.Serializer.Serialize(result)
		if err != nil {
			a.fail(ctx, errors.Annotate(err, "serialize"))
		} else {
			a.done(ctx, &serialized)
		}
	}
}

func (h *ActivityWorker) fail(ctx *ActivityContext, err error) {
	task := ctx.Task
	if h.BackoffOnFailure {
		hist, err := h.SWF.GetWorkflowExecutionHistory(&swf.GetWorkflowExecutionHistoryInput{
			Domain:       S(h.Domain),
			Execution:    task.WorkflowExecution,
			ReverseOrder: aws.True(),
		})
		if err == nil {
			for _, e := range hist.Events {
				if *e.EventType == swf.EventTypeMarkerRecorded && *e.MarkerRecordedEventAttributes.MarkerName == fsm.CorrelatorMarker {
					correlator := new(fsm.EventCorrelator)
					err := h.Serializer.Deserialize(*e.MarkerRecordedEventAttributes.Details, correlator)
					if err == nil {
						attempts := correlator.ActivityAttempts[*task.ActivityID]
						backoff := h.backoff(attempts)
						log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=retry-backoff attempts=%d sleep=%ds ", LS(task.WorkflowExecution.WorkflowID), LS(task.ActivityType.Name), LS(task.ActivityID), attempts, backoff)
						time.Sleep(time.Duration(backoff) * time.Second)
					}
					break
				}
			}
		}
	}
	log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=fail error=%s ", LS(task.WorkflowExecution.WorkflowID), LS(task.ActivityType.Name), LS(task.ActivityID), err.Error())
	failErr := h.SWF.RespondActivityTaskFailed(&swf.RespondActivityTaskFailedInput{
		TaskToken: task.TaskToken,
		Reason:    S(err.Error()),
		Details:   S(err.Error()),
	})
	if failErr != nil {
		log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=failed-response-fail error=%s ", LS(task.WorkflowExecution.WorkflowID), LS(task.ActivityType.Name), LS(task.ActivityID), failErr.Error())
	}
}

func (h *ActivityWorker) signalStart(ctx *ActivityContext, data interface{}) error {
	return h.signal(ctx, fsm.ActivityStartedSignal, data)
}

func (h *ActivityWorker) signalUpdate(ctx *ActivityContext, data interface{}) error {
	return h.signal(ctx, fsm.ActivityUpdatedSignal, data)
}

func (h *ActivityWorker) signal(ctx *ActivityContext, signal string, data interface{}) error {
	state := new(fsm.SerializedActivityState)
	state.ActivityID = *ctx.Task.ActivityID
	if data != nil {
		ser, err := h.Serializer.Serialize(data)
		if err != nil {
			return err
		}
		state.Input = &ser
	}

	serializedState, err := h.systemSerializer.Serialize(state)
	if err != nil {
		return err
	}

	return h.SWF.SignalWorkflowExecution(&swf.SignalWorkflowExecutionInput{
		Domain:     S(h.Domain),
		WorkflowID: ctx.Task.WorkflowExecution.WorkflowID,
		SignalName: S(signal),
		Input:      S(serializedState),
	})
}

func (h *ActivityWorker) backoff(attempts int) int {
	// 0.5, 1, 2, 4, 8...
	exp := attempts - 1
	if exp > 30 {
		//int wraps at 31
		exp = 30
	}
	backoff := int(math.Pow(2, float64(exp)))
	maxBackoff := h.MaxBackoffSeconds
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	return backoff
}

func (h *ActivityWorker) done(ctx *ActivityContext, result *string) {
	log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=done", LS(ctx.Task.WorkflowExecution.WorkflowID), LS(ctx.Task.ActivityType.Name), LS(ctx.Task.ActivityID))

	completeErr := h.SWF.RespondActivityTaskCompleted(&swf.RespondActivityTaskCompletedInput{
		TaskToken: ctx.Task.TaskToken,
		Result:    result,
	})
	if completeErr != nil {
		log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=completed-response-fail error=%s ", LS(ctx.Task.WorkflowExecution.WorkflowID), LS(ctx.Task.ActivityType.Name), LS(ctx.Task.ActivityID), completeErr.Error())
	}
}

func (h *ActivityWorker) canceled(ctx *ActivityContext, details *string) {
	log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=cancled", LS(ctx.Task.WorkflowExecution.WorkflowID), LS(ctx.Task.ActivityType.Name), LS(ctx.Task.ActivityID))

	canceledErr := h.SWF.RespondActivityTaskCanceled(&swf.RespondActivityTaskCanceledInput{
		TaskToken: ctx.Task.TaskToken,
		Details:   details,
	})
	if canceledErr != nil {
		log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=canceled-response-fail error=%s ", LS(ctx.Task.WorkflowExecution.WorkflowID), LS(ctx.Task.ActivityType.Name), LS(ctx.Task.ActivityID), canceledErr.Error())
	}
}

func (h *ActivityWorker) handleWithRecovery(handler func(*ActivityContext)) func(*ActivityContext) {
	return func(ctx *ActivityContext) {
		defer func() {
			var anErr error
			if r := recover(); r != nil {
				if err, ok := r.(error); ok && err != nil {
					anErr = err
				} else {
					anErr = errors.New("panic in activity with nil error")
				}
				log.Printf("component=activity at=error error=activity-panic-recovery msg=%s", r)
				h.fail(ctx, anErr)
			}
		}()
		handler(ctx)

	}
}
