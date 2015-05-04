package activity

import (
	"log"
	"time"

	"strings"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/swf"
	. "github.com/sclasen/swfsm/sugar"
)

//NewCoordinatedActivityHandler creates a LongRunningActivityFunc that will build a LongRunningActivityCoordinator and execute your HandleCoordinatedActivity
//
// * heartbeats the activity at the given interval
// * sends any errors heartbeating into the HeartbeatErrors channel, which is buffered by heartbeatErrorThreshold
// * if the send to HeartbeatErrors blocks because the buffer is full and not being consumed the task will eventually timeout.
// * if the heartbeat indicates the task was canceled, send on the ToCancelActivity channel and wait on ToAckCancelActivity channel
// * your CoordinatedActivityHandler is responsible for responding to messages on ToCancelActivity, by stopping, acking the cancel to swf, and sending on ToAckCancel
// * if your CoordinatedActivityHandler wishes to stop heartbeats, send on ToStopHeartbeating and recieve on ToAckStopHeartbeating.
// * your CoordinatedActivityHandler is responsible for responding to messages on ToStopActivity, by stopping, NOT acking the cancel to swf, and sending on ToAckStopActivity.
// * this happens when heartbeats are attempted after a task is timed out or canceled or completed.

const (
	TaskGone = "Unknown activity"
)

func (w *ActivityWorker) AddCoordinatedHandler(heartbeatInterval time.Duration, heartbeatErrorThreshold int, handler *CoordinatedActivityHandler) {
	coordinator := &LongRunningActivityCoordinator{
		HeartbeatInterval:     heartbeatInterval,
		HeartbeatErrors:       make(chan error, heartbeatErrorThreshold),
		ToCancelActivity:      make(chan struct{}),
		ToAckCancelActivity:   make(chan struct{}),
		ToStopHeartbeating:    make(chan struct{}),
		ToAckStopHeartbeating: make(chan struct{}),
		ToStopActivity:        make(chan struct{}),
		ToAckStopActivity:     make(chan struct{}),
	}

	handlerFunc := func(activityTask *swf.ActivityTask, input interface{}) {
		go handler.HandlerFunc(coordinator, activityTask, input)
		for {
			select {
			case <-time.After(heartbeatInterval):
				status, err := w.SWF.RecordActivityTaskHeartbeat(&swf.RecordActivityTaskHeartbeatInput{
					TaskToken: activityTask.TaskToken,
				})
				if err != nil {
					if ae, ok := err.(aws.APIError); ok && ae.Code == ErrorTypeUnknownResourceFault && strings.Contains(ae.Message, TaskGone) {
						log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-gone", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID))
						coordinator.ToStopActivity <- struct{}{}
						<-coordinator.ToAckStopActivity
						log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=stopped-activity-gone", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID))
						return
					}
					log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=heartbeat-error error=%s ", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID), err.Error())
					coordinator.HeartbeatErrors <- err
				} else {
					if *status.CancelRequested {
						log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-cancel-requested", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID))
						coordinator.ToCancelActivity <- struct{}{}
						<-coordinator.ToAckCancelActivity
						log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=activity-canceled", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID))
						return
					}
				}
			case <-coordinator.ToStopHeartbeating:
				log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=stop-heartbeating", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID))
				coordinator.ToAckStopHeartbeating <- struct{}{}
				log.Printf("workflow-id=%s activity-id=%s activity-id=%s at=ack-stop-heartbeating", LS(activityTask.WorkflowExecution.WorkflowID), LS(activityTask.ActivityType.Name), LS(activityTask.ActivityID))
				return
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
