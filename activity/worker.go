package activity

import (
	"fmt"
	"log"

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
}

type ActivityWorker struct {
	Serializer fsm.StateSerializer
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

func (a *ActivityWorker) dispatchTask(activityTask *swf.ActivityTask) {
	a.ActivityTaskDispatcher.DispatchTask(activityTask, a.handleWithRecovery(a.handleActivityTask))
}

func (a *ActivityWorker) handleActivityTask(activityTask *swf.ActivityTask) {
	a.ActivityInterceptor.BeforeTask(activityTask)
	handler := a.handlers[*activityTask.ActivityType.Name]
	if handler != nil {
		var deserialized interface{}
		if activityTask.Input != nil {
			deserialized = handler.ZeroInput()
			err := a.Serializer.Deserialize(*activityTask.Input, deserialized)
			if err != nil {
				a.ActivityInterceptor.AfterTaskFailed(activityTask, err)
				a.fail(activityTask, err)
				return
			}
		} else {
			deserialized = nil
		}

		result, err := handler.HandlerFunc(activityTask, deserialized)
		if err != nil {
			a.ActivityInterceptor.AfterTaskFailed(activityTask, err)
			a.fail(activityTask, err)
		} else {
			if result == nil {
				a.ActivityInterceptor.AfterTaskComplete(activityTask, "")
			} else {
				a.ActivityInterceptor.AfterTaskComplete(activityTask, result)
				switch t := result.(type) {
				case string:
					a.done(activityTask, t)
				default:
					serialized, err := a.Serializer.Serialize(result)
					if err != nil {
						a.fail(activityTask, err)
					} else {
						a.done(activityTask, serialized)
					}
				}
			}
		}
	} else {
		//fail
		err := errors.NewErr("no handler for activity: %s", activityTask.ActivityType.Name)
		a.ActivityInterceptor.AfterTaskFailed(activityTask, &err)
		a.fail(activityTask, &err)
	}
}

func (h *ActivityWorker) fail(resp *swf.ActivityTask, err error) {
	log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=fail error=%s ", *resp.WorkflowExecution.WorkflowID, *resp.ActivityType.Name, *resp.ActivityID, err.Error())
	failErr := h.SWF.RespondActivityTaskFailed(&swf.RespondActivityTaskFailedInput{
		TaskToken: resp.TaskToken,
		Reason:    S(err.Error()),
		Details:   S(err.Error()),
	})
	if failErr != nil {
		log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=failed-response-fail error=%s ", *resp.WorkflowExecution.WorkflowID, *resp.ActivityType.Name, *resp.ActivityID, failErr.Error())
	}
}

func (h *ActivityWorker) done(resp *swf.ActivityTask, result string) {
	log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=done", *resp.WorkflowExecution.WorkflowID, *resp.ActivityType.Name, *resp.ActivityID)

	completeErr := h.SWF.RespondActivityTaskCompleted(&swf.RespondActivityTaskCompletedInput{
		TaskToken: resp.TaskToken,
		Result:    S(result),
	})
	if completeErr != nil {
		log.Printf("workflow-id=%s  activity-id=%s activity-id=%s at=completed-response-fail error=%s ", *resp.WorkflowExecution.WorkflowID, *resp.ActivityType.Name, *resp.ActivityID, completeErr.Error())
	}
}

func (h *ActivityWorker) handleWithRecovery(handler func(*swf.ActivityTask)) func(*swf.ActivityTask) {
	return func(resp *swf.ActivityTask) {
		defer func() {
			var anErr error
			if r := recover(); r != nil {
				if err, ok := r.(error); ok && err != nil {
					anErr = err
				} else {
					anErr = errors.New("panic in activity with nil error")
				}
				log.Printf("component=activity at=error error=activity-panic-recovery msg=%s", r)
				h.fail(resp, anErr)
			}
		}()
		handler(resp)

	}
}
