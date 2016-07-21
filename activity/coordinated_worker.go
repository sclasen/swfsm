package activity

import (
	"time"

	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/swf"
	. "github.com/sclasen/swfsm/log"
	. "github.com/sclasen/swfsm/sugar"
)

const (
	TaskGone      = "Unknown activity"
	ExecutionGone = "Unknown execution"
)

// AddCoordinatedHandler automatically takes care of sending back heartbeats and
// updating state on workflows for an activity task. tickMinInterval determines
// the max rate at which the CoordinatedActivityHandler.Tick function will be
// called.
//
// For example, when the Tick function returns quickly (e.g.: noop), and
// tickMinInterval is 1 * time.Second, Tick is guaranteed to be called at most
// once per second. The rate can be slower if Tick takes more than
// tickMinInterval to complete.
func (w *ActivityWorker) AddCoordinatedHandler(heartbeatInterval, tickMinInterval time.Duration, handler *CoordinatedActivityHandler) {
	adapter := &coordinatedActivityAdapter{
		heartbeatInterval: heartbeatInterval,
		tickMinInterval:   tickMinInterval,
		worker:            w,
		handler:           handler,
	}
	w.AddHandler(&ActivityHandler{
		Activity:    handler.Activity,
		HandlerFunc: adapter.coordinate,
		Input:       handler.Input,
	})
}

type coordinatedActivityAdapter struct {
	heartbeatInterval time.Duration
	tickMinInterval   time.Duration
	worker            *ActivityWorker
	handler           *CoordinatedActivityHandler
}

func (c *coordinatedActivityAdapter) heartbeat(activityTask *swf.PollForActivityTaskOutput, stop <-chan struct{}, cancelActivity chan error) {
	heartbeats := time.NewTicker(c.heartbeatInterval)
	defer heartbeats.Stop()
	for {
		select {
		case <-heartbeats.C:
			if status, err := c.worker.SWF.RecordActivityTaskHeartbeat(&swf.RecordActivityTaskHeartbeatInput{
				TaskToken: activityTask.TaskToken,
			}); err != nil {
				if ae, ok := err.(awserr.Error); ok && isGoneError(ae) {
					Log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-gone", LS(activityTask.WorkflowExecution.WorkflowId), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityId))
					cancelActivity <- nil
					return
				}
				Log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=heartbeat-error error=%q", LS(activityTask.WorkflowExecution.WorkflowId), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityId), err.Error())
			} else {
				Log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=heartbeat-recorded", LS(activityTask.WorkflowExecution.WorkflowId), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityId))
				if *status.CancelRequested {
					Log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-cancel-requested", LS(activityTask.WorkflowExecution.WorkflowId), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityId))
					cancelActivity <- ActivityTaskCanceledError{}
					return
				}
			}
		case <-stop:
			return
		}
	}
}

func (c *coordinatedActivityAdapter) coordinate(activityTask *swf.PollForActivityTaskOutput, input interface{}) (interface{}, error) {
	defer c.finish(activityTask, input)

	update, err := c.handler.Start(activityTask, input)
	if err != nil {
		return nil, err
	}
	if err := c.worker.signalStart(activityTask, update); err != nil {
		return nil, err
	}

	cancel := make(chan error, 2)
	stopHeartbeating := make(chan struct{})

	go c.heartbeat(activityTask, stopHeartbeating, cancel)
	defer close(stopHeartbeating)

	ticks := time.NewTicker(c.tickMinInterval)
	defer ticks.Stop()
	for {
		select {
		case cause := <-cancel:
			c.cancel(activityTask, input)
			return nil, cause
		case <-ticks.C:
			cont, res, err := c.handler.Tick(activityTask, input)
			if !cont {
				return res, err
			}
			if res != nil {
				//send an activity update when the result is not null, but we are continuing
				if err := c.worker.signalUpdate(activityTask, res); err != nil {
					Log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=signal-update-error error=%q", LS(activityTask.WorkflowExecution.WorkflowId), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityId), err)
					cancel <- err
					continue // go pick up the cancel message
				}
				Log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=signal-update", LS(activityTask.WorkflowExecution.WorkflowId), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityId))
			}
		}
	}
}

func (c *coordinatedActivityAdapter) cancel(activityTask *swf.PollForActivityTaskOutput, input interface{}) {
	if err := c.handler.Cancel(activityTask, input); err != nil {
		Log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-cancel-err error=%q", LS(activityTask.WorkflowExecution.WorkflowId), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityId), err)
	}
}

func (c *coordinatedActivityAdapter) finish(activityTask *swf.PollForActivityTaskOutput, input interface{}) {
	if err := c.handler.Finish(activityTask, input); err != nil {
		Log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-finish-err error=%q", LS(activityTask.WorkflowExecution.WorkflowId), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityId), err)
	}
}

func isGoneError(err awserr.Error) bool {
	return err.Code() == ErrorTypeUnknownResourceFault &&
		(strings.Contains(err.Message(), TaskGone) || strings.Contains(err.Message(), ExecutionGone))
}
