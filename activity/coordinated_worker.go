package activity

import (
	"log"
	"time"

	"strings"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/swf"
	. "github.com/sclasen/swfsm/sugar"
)

const (
	TaskGone = "Unknown activity"
)

func (w *ActivityWorker) AddCoordinatedHandler(heartbeatInterval time.Duration, heartbeatErrorThreshold int, handler *CoordinatedActivityHandler) {

	ticks := make(chan CoordinatedActivityHandlerTick)
	startErr := make(chan error)

	handlerTickFunc := func(activityTask *swf.ActivityTask, input interface{}) {
		go func() {
			cont, res, err := handler.Tick(activityTask, input)
			ticks <- CoordinatedActivityHandlerTick{
				Continue: cont,
				Result:   res,
				Error:    err,
			}
		}()
	}

	handlerStartFunc := func(activityTask *swf.ActivityTask, input interface{}) {
		go func() {
			err := handler.Start(activityTask, input)
			if err != nil {
				startErr <- err
			} else {
				err = w.signalStart(activityTask)
				if err != nil {
					startErr <- err
				} else {
					handlerTickFunc(activityTask, input)
				}
			}
		}()
	}

	heartbeats := time.Tick(heartbeatInterval)

	handlerFunc := func(activityTask *swf.ActivityTask, input interface{}) {
		handlerStartFunc(activityTask, input)
		for {
			select {
			case e := <-startErr:
				w.fail(activityTask, e)
				return
			case tick := <-ticks:
				if tick.Continue {
					handlerTickFunc(activityTask, input)
				} else {
					if tick.Error != nil {
						w.fail(activityTask, tick.Error)
					} else {
						w.result(activityTask, tick.Result)
					}
					return
				}
			case <-heartbeats:
				status, err := w.SWF.RecordActivityTaskHeartbeat(&swf.RecordActivityTaskHeartbeatInput{
					TaskToken: activityTask.TaskToken,
				})
				if err != nil {
					if ae, ok := err.(aws.APIError); ok && ae.Code == ErrorTypeUnknownResourceFault && strings.Contains(ae.Message, TaskGone) {
						log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-gone", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID))
						//wait for the tick to complete before exiting
						<-ticks
						return
					}
					log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=heartbeat-error error=%s ", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID), err.Error())
				} else {
					log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=heartbeat-recorded", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID))
					if *status.CancelRequested {
						log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-cancel-requested", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID))
						//Wait for any tick to finish
						<-ticks
						handler.Cancel(activityTask, input)
						w.canceled(activityTask)
						log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-canceld", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID))
						return
					}
				}
			}
		}
	}

	l := &LongRunningActivityHandler{
		Activity:    handler.Activity,
		HandlerFunc: handlerFunc,
		Input:       handler.Input,
	}

	w.AddLongRunningHandler(l)
}

type CoordinatedActivityHandlerTick struct {
	Continue bool
	Result   interface{}
	Error    error
}
