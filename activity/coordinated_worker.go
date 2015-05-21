package activity

import (
	"log"
	"time"

	"strings"

	"github.com/awslabs/aws-sdk-go/aws/awserr"
	"github.com/awslabs/aws-sdk-go/service/swf"
	. "github.com/sclasen/swfsm/sugar"
)

const (
	TaskGone = "Unknown activity"
)

func (w *ActivityWorker) AddCoordinatedHandler(heartbeatInterval time.Duration, handler *CoordinatedActivityHandler) {

	ticks := make(chan CoordinatedActivityHandlerTick)
	startErr := make(chan error)

	handlerTickFunc := func(activityTask *swf.PollForActivityTaskOutput, input interface{}) {
		go func() {
			cont, res, err := handler.Tick(activityTask, input)
			ticks <- CoordinatedActivityHandlerTick{
				Continue: cont,
				Result:   res,
				Error:    err,
			}
		}()
	}

	handlerStartFunc := func(activityTask *swf.PollForActivityTaskOutput, input interface{}) {
		go func() {
			update, err := handler.Start(activityTask, input)
			if err != nil {
				startErr <- err
			} else {
				err = w.signalStart(activityTask, update)
				if err != nil {
					startErr <- err
				} else {
					handlerTickFunc(activityTask, input)
				}
			}
		}()
	}

	heartbeats := time.Tick(heartbeatInterval)

	handlerFunc := func(activityTask *swf.PollForActivityTaskOutput, input interface{}) {
		handlerStartFunc(activityTask, input)
		for {
			select {
			case e := <-startErr:
				log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=start-error error=%q", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID), e)
				w.fail(activityTask, e)
				return
			case tick := <-ticks:
				if tick.Continue {
					handlerTickFunc(activityTask, input)
					if tick.Result != nil {
						//send an activity update when the result is not null, but we are continuing
						err := w.signalUpdate(activityTask, tick.Result)
						if err != nil {
							log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=signal-update-error error=%q", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID), err)
							w.fail(activityTask, err)
							log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=signal-update-error-fail-task error=%q", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID), err)
							handler.Cancel(activityTask, tick.Result)
							log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=signal-update-error-cancel-handler error=%q", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID), err)

						} else {
							log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=signal-update", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID))
						}
					}
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
					if ae, ok := err.(awserr.Error); ok && ae.Code() == ErrorTypeUnknownResourceFault && strings.Contains(ae.Message(), TaskGone) {
						log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-gone", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID))
						//wait for the tick to complete before exiting
						<-ticks
						err := handler.Cancel(activityTask, input)
						if err != nil {
							log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-gone-cancel-err err=%q", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID), err)
						}
						return
					}
					log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=heartbeat-error error=%s ", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID), err.Error())
				} else {
					log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=heartbeat-recorded", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID))
					if *status.CancelRequested {
						log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-cancel-requested", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID))
						//Wait for any tick to finish
						<-ticks
						var detail *string
						err := handler.Cancel(activityTask, input)
						if err != nil {
							log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-cancel-err err=%q", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID), err)
							detail = S(err.Error())
						}
						w.canceled(activityTask, detail)
						log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-canceled", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID))
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
