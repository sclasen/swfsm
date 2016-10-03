package activity

import (
	"fmt"

	"time"

	"math"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/juju/errors"
	"github.com/sclasen/swfsm/fsm"
	"github.com/sclasen/swfsm/internal/panicinfo"
	. "github.com/sclasen/swfsm/log"
	"github.com/sclasen/swfsm/poller"
	. "github.com/sclasen/swfsm/sugar"
)

type ActivityTaskCanceledError struct {
	details string
}

func (e ActivityTaskCanceledError) Error() string {
	return "AcvitityTask canceled: " + e.details
}

func (e ActivityTaskCanceledError) Details() *string {
	if e.details == "" {
		return nil
	}
	dup := e.details
	return &dup
}

type SWFOps interface {
	RecordActivityTaskHeartbeat(req *swf.RecordActivityTaskHeartbeatInput) (*swf.RecordActivityTaskHeartbeatOutput, error)
	RespondActivityTaskCanceled(req *swf.RespondActivityTaskCanceledInput) (*swf.RespondActivityTaskCanceledOutput, error)
	RespondActivityTaskCompleted(req *swf.RespondActivityTaskCompletedInput) (*swf.RespondActivityTaskCompletedOutput, error)
	RespondActivityTaskFailed(req *swf.RespondActivityTaskFailedInput) (*swf.RespondActivityTaskFailedOutput, error)
	PollForActivityTask(req *swf.PollForActivityTaskInput) (*swf.PollForActivityTaskOutput, error)
	GetWorkflowExecutionHistory(req *swf.GetWorkflowExecutionHistoryInput) (*swf.GetWorkflowExecutionHistoryOutput, error)
	SignalWorkflowExecution(req *swf.SignalWorkflowExecutionInput) (*swf.SignalWorkflowExecutionOutput, error)
}

type ActivityWorker struct {
	Serializer       fsm.StateSerializer
	SystemSerializer fsm.StateSerializer
	// Domain of the workflow associated with the FSM.
	Domain string
	// TaskList that the underlying poller will poll for decision tasks.
	TaskList string
	// Identity used in PollForActivityTaskRequests, can be empty.
	Identity string
	// Client used to make SWF api requests.
	SWF SWFOps
	// Type Info for handled activities
	handlers map[string]*ActivityHandler
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

func (a *ActivityWorker) Init() {
	if a.Serializer == nil {
		a.Serializer = fsm.JSONStateSerializer{}
	}

	if a.SystemSerializer == nil {
		a.SystemSerializer = fsm.JSONStateSerializer{}
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
	go poller.PollUntilShutdownBy(a.ShutdownManager, fmt.Sprintf("%s-poller", a.Identity), a.dispatchTask)
}

func (a *ActivityWorker) dispatchTask(activityTask *swf.PollForActivityTaskOutput) {
	if a.AllowPanics {
		a.ActivityTaskDispatcher.DispatchTask(activityTask, a.HandleActivityTask)
	} else {
		a.ActivityTaskDispatcher.DispatchTask(activityTask, a.HandleWithRecovery(a.HandleActivityTask))
	}
}

// HandleActivityTask is the callback passed into the registered ActivityTaskDispatcher.
// It is exposed so that users can handle polling themselves and call DispatchTask directly
// with this as the callback.
//
// e.g.  activityWorker.ActivityTaskDispatcher.DispatchTask(activityTask, a.HandleWithRecovery(a.HandleActivityTask))
//
// Note: You will need to handle recovering from panics if you call this directly without wrapping
// with HandleWithRecovery.
func (a *ActivityWorker) HandleActivityTask(activityTask *swf.PollForActivityTaskOutput) {
	a.ActivityInterceptor.BeforeTask(activityTask)
	handler := a.handlers[*activityTask.ActivityType.Name]

	if handler == nil {
		err := errors.NewErr("no handler for activity: %s", LS(activityTask.ActivityType.Name))
		a.ActivityInterceptor.AfterTaskFailed(activityTask, &err)
		a.fail(activityTask, &err)
		return
	}

	var deserialized interface{}
	if activityTask.Input != nil {
		switch handler.Input.(type) {
		case string:
			deserialized = *activityTask.Input
		default:
			deserialized = handler.ZeroInput()
			err := a.Serializer.Deserialize(*activityTask.Input, deserialized)
			if err != nil {
				a.ActivityInterceptor.AfterTaskFailed(activityTask, err)
				a.fail(activityTask, errors.Annotate(err, "deserialize"))
				return
			}
		}
	} else {
		deserialized = nil
	}

	result, err := handler.HandlerFunc(activityTask, deserialized)
	result, err = a.ActivityInterceptor.AfterTask(activityTask, result, err)
	if err != nil {
		if e, ok := err.(ActivityTaskCanceledError); ok {
			a.ActivityInterceptor.AfterTaskCanceled(activityTask, e.details)
			a.canceled(activityTask, e.Details())
		} else {
			a.ActivityInterceptor.AfterTaskFailed(activityTask, err)
			a.fail(activityTask, errors.Annotate(err, "handler"))
		}
	} else {
		a.ActivityInterceptor.AfterTaskComplete(activityTask, result)
		a.result(activityTask, result)
	}
}

func (a *ActivityWorker) result(activityTask *swf.PollForActivityTaskOutput, result interface{}) {
	switch t := result.(type) {
	case string:
		a.done(activityTask, &t)
	case nil:
		a.done(activityTask, nil)
	default:
		serialized, err := a.Serializer.Serialize(result)
		if err != nil {
			a.fail(activityTask, errors.Annotate(err, "serialize"))
		} else {
			a.done(activityTask, &serialized)
		}
	}
}

func (h *ActivityWorker) fail(task *swf.PollForActivityTaskOutput, err error) {
	if h.BackoffOnFailure {
		hist, err := h.SWF.GetWorkflowExecutionHistory(&swf.GetWorkflowExecutionHistoryInput{
			Domain:       S(h.Domain),
			Execution:    task.WorkflowExecution,
			ReverseOrder: aws.Bool(true),
		})
		if err == nil {
			for _, e := range hist.Events {
				if *e.EventType == swf.EventTypeMarkerRecorded && *e.MarkerRecordedEventAttributes.MarkerName == fsm.CorrelatorMarker {
					correlator := new(fsm.EventCorrelator)
					err := h.Serializer.Deserialize(*e.MarkerRecordedEventAttributes.Details, correlator)
					if err == nil {
						attempts := correlator.ActivityAttempts[*task.ActivityId]
						backoff := h.backoff(attempts)
						Log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=retry-backoff attempts=%d sleep=%ds ", LS(task.WorkflowExecution.WorkflowId), LS(task.ActivityType.Name), LS(task.ActivityId), attempts, backoff)
						time.Sleep(time.Duration(backoff) * time.Second)
					}
					break
				}
			}
		}
	}
	Log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=fail error=%q", LS(task.WorkflowExecution.WorkflowId), LS(task.ActivityType.Name), LS(task.ActivityId), err.Error())
	if len(err.Error()) > FailureReasonMaxChars {
		Log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=truncating-failure-reason error=%q", LS(task.WorkflowExecution.WorkflowId), LS(task.ActivityType.Name), LS(task.ActivityId), err.Error())
	}
	_, failErr := h.SWF.RespondActivityTaskFailed(&swf.RespondActivityTaskFailedInput{
		TaskToken: task.TaskToken,
		Reason:    S(truncate(err.Error(), FailureReasonMaxChars)),
		Details:   S(err.Error()),
	})
	if failErr != nil {
		Log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=failed-response-fail error=%q", LS(task.WorkflowExecution.WorkflowId), LS(task.ActivityType.Name), LS(task.ActivityId), failErr.Error())
	}
}

func (h *ActivityWorker) signalStart(activityTask *swf.PollForActivityTaskOutput, data interface{}) error {
	return h.signal(activityTask, fsm.ActivityStartedSignal, data)
}

func (h *ActivityWorker) signalUpdate(activityTask *swf.PollForActivityTaskOutput, data interface{}) error {
	return h.signal(activityTask, fsm.ActivityUpdatedSignal, data)
}

func (h *ActivityWorker) signal(activityTask *swf.PollForActivityTaskOutput, signal string, data interface{}) error {
	state := new(fsm.SerializedActivityState)
	state.ActivityId = *activityTask.ActivityId
	if data != nil {
		ser, err := h.Serializer.Serialize(data)
		if err != nil {
			return err
		}
		state.Input = &ser
	}

	serializedState, err := h.SystemSerializer.Serialize(state)
	if err != nil {
		return err
	}

	_, rerr := h.SWF.SignalWorkflowExecution(&swf.SignalWorkflowExecutionInput{
		Domain:     S(h.Domain),
		WorkflowId: activityTask.WorkflowExecution.WorkflowId,
		SignalName: S(signal),
		Input:      S(serializedState),
	})

	return rerr
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

func (h *ActivityWorker) done(resp *swf.PollForActivityTaskOutput, result *string) {
	Log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=done", LS(resp.WorkflowExecution.WorkflowId), LS(resp.ActivityType.Name), LS(resp.ActivityId))

	_, completeErr := h.SWF.RespondActivityTaskCompleted(&swf.RespondActivityTaskCompletedInput{
		TaskToken: resp.TaskToken,
		Result:    result,
	})
	if completeErr != nil {
		Log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=completed-response-fail error=%q", LS(resp.WorkflowExecution.WorkflowId), LS(resp.ActivityType.Name), LS(resp.ActivityId), completeErr.Error())
	}
}

func (h *ActivityWorker) canceled(resp *swf.PollForActivityTaskOutput, details *string) {
	Log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=canceled", LS(resp.WorkflowExecution.WorkflowId), LS(resp.ActivityType.Name), LS(resp.ActivityId))

	_, canceledErr := h.SWF.RespondActivityTaskCanceled(&swf.RespondActivityTaskCanceledInput{
		TaskToken: resp.TaskToken,
		Details:   details,
	})
	if canceledErr != nil {
		Log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=canceled-response-fail error=%q", LS(resp.WorkflowExecution.WorkflowId), LS(resp.ActivityType.Name), LS(resp.ActivityId), canceledErr.Error())
	}
}

func truncate(s string, i int) string {
	runes := []rune(s)
	if len(runes) > i {
		return string(runes[:i])
	}
	return s
}

// HandleWithRecovery is used to wrap handler functions (such as HandleActivityTask)
// so they gracefully recover from panics.
func (h *ActivityWorker) HandleWithRecovery(handler func(*swf.PollForActivityTaskOutput)) func(*swf.PollForActivityTaskOutput) {
	return func(resp *swf.PollForActivityTaskOutput) {
		defer func() {
			var anErr error
			if r := recover(); r != nil {
				file, line, name := panicinfo.LocatePanic(r)
				if err, ok := r.(error); ok && err != nil {
					anErr = err
				} else {
					anErr = errors.New("panic in activity with nil error")
				}
				Log.Printf("component=activity at=activity-panic-recovery-error func=%q file=\"%s:%d\" error=%q", name, file, line, r)
				h.fail(resp, anErr)
			}
		}()
		handler(resp)

	}
}
