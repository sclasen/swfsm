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

func (w *ActivityWorker) AddCoordinatedHandler(heartbeatInterval time.Duration, handler *CoordinatedActivityHandler) {

	ticks := make(chan CoordinatedActivityHandlerTick)
	startErr := make(chan error)

	handlerTickFunc := func(cxt *ActivityContext, input interface{}) {
		go func() {
			cont, res, err := handler.Tick(cxt, input)
			ticks <- CoordinatedActivityHandlerTick{
				Continue: cont,
				Result:   res,
				Error:    err,
			}
		}()
	}

	handlerStartFunc := func(ctx *ActivityContext, input interface{}) {
		go func() {
			update, err := handler.Start(ctx, input)
			if err != nil {
				startErr <- err
			} else {
				err = w.signalStart(ctx, update)
				if err != nil {
					startErr <- err
				} else {
					handlerTickFunc(ctx, input)
				}
			}
		}()
	}

	heartbeats := time.Tick(heartbeatInterval)

	handlerFunc := func(ctx *ActivityContext, input interface{}) {
		if ctx.Resumed {
			handlerTickFunc(ctx, input)
		} else {
			handlerStartFunc(ctx, input)
		}
		for {
			select {
			case e := <-startErr:
				log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=start-error error=%q", LS(ctx.Task.WorkflowExecution.WorkflowID), LS(ctx.Task.ActivityType.Name), LS(ctx.Task.ActivityID), e)
				w.fail(ctx, e)
				return
			case tick := <-ticks:
				if tick.Continue {
					handlerTickFunc(ctx, input)
					if tick.Result != nil {
						//send an activity update when the result is not null, but we are continuing
						err := w.signalUpdate(ctx, tick.Result)
						if err != nil {
							log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=signal-update-error error=%q", LS(ctx.Task.WorkflowExecution.WorkflowID), LS(ctx.Task.ActivityType.Name), LS(ctx.Task.ActivityID), err)
							w.fail(ctx, err)
							log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=signal-update-error-fail-task error=%q", LS(ctx.Task.WorkflowExecution.WorkflowID), LS(ctx.Task.ActivityType.Name), LS(ctx.Task.ActivityID), err)
							handler.Cancel(ctx, tick.Result)
							log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=signal-update-error-cancel-handler error=%q", LS(ctx.Task.WorkflowExecution.WorkflowID), LS(ctx.Task.ActivityType.Name), LS(ctx.Task.ActivityID), err)

						} else {
							log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=signal-update", LS(ctx.Task.WorkflowExecution.WorkflowID), LS(ctx.Task.ActivityType.Name), LS(ctx.Task.ActivityID))
						}
					}
				} else {
					if tick.Error != nil {
						w.fail(ctx, tick.Error)
					} else {
						w.result(ctx, tick.Result)
					}
					return
				}
			case <-heartbeats:
				status, err := w.SWF.RecordActivityTaskHeartbeat(&swf.RecordActivityTaskHeartbeatInput{
					TaskToken: ctx.Task.TaskToken,
				})
				if err != nil {
					if ae, ok := err.(aws.APIError); ok && ae.Code == ErrorTypeUnknownResourceFault && strings.Contains(ae.Message, TaskGone) {
						log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-gone", LS(ctx.Task.WorkflowExecution.WorkflowID), LS(ctx.Task.ActivityType.Name), LS(ctx.Task.ActivityID))
						//wait for the tick to complete before exiting
						<-ticks
						err := handler.Cancel(ctx, input)
						if err != nil {
							log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-gone-cancel-err err=%q", LS(ctx.Task.WorkflowExecution.WorkflowID), LS(ctx.Task.ActivityType.Name), LS(ctx.Task.ActivityID), err)
						}
						return
					}
					log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=heartbeat-error error=%s ", LS(ctx.Task.WorkflowExecution.WorkflowID), LS(ctx.Task.ActivityType.Name), LS(ctx.Task.ActivityID), err.Error())
				} else {
					log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=heartbeat-recorded", LS(ctx.Task.WorkflowExecution.WorkflowID), LS(ctx.Task.ActivityType.Name), LS(ctx.Task.ActivityID))
					if *status.CancelRequested {
						log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-cancel-requested", LS(ctx.Task.WorkflowExecution.WorkflowID), LS(ctx.Task.ActivityType.Name), LS(ctx.Task.ActivityID))
						//Wait for any tick to finish
						<-ticks
						var detail *string
						err := handler.Cancel(ctx, input)
						if err != nil {
							log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-cancel-err err=%q", LS(ctx.Task.WorkflowExecution.WorkflowID), LS(ctx.Task.ActivityType.Name), LS(ctx.Task.ActivityID), err)
							detail = S(err.Error())
						}
						w.canceled(ctx, detail)
						log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-canceled", LS(ctx.Task.WorkflowExecution.WorkflowID), LS(ctx.Task.ActivityType.Name), LS(ctx.Task.ActivityID))
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
