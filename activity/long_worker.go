package activity

import (
	"log"
	"time"

	"github.com/awslabs/aws-sdk-go/gen/swf"
	. "github.com/sclasen/swfsm/sugar"
)

//LongRunningActivityFunc that creates all the coordination channels, starts heartbeating, and calls into
type LongRunningActivityCoordinator struct {
	HeartbeatInterval     time.Duration
	ToCancelActivity      chan struct{}
	ToAckCancelActivity   chan struct{}
	ToStopHeartbeating    chan struct{}
	ToAckStopHeartbeating chan struct{}
	HeartbeatErrors       chan error
}

type HandleCoordinatedActivity func(*LongRunningActivityCoordinator, *swf.ActivityTask, interface{})

//NewCoordinatedActivityHandler creates a LongRunningActivityFunc that will build a LongRunningActivityCoordinator and execute your HandleCoordinatedActivity
//
// * heartbeats the activity at the given interval
// * sends any errors heartbeating into the HeartbeatErrors channel, which is buffered by heartbeatErrorThreshold
// * if the send to HeartbeatErrors blocks because the buffer is full and not being consumed the task will eventually timeout.
// * if the heartbeat indicates the task was canceled, send on the ToCancelActivity channel and wait on ToAckCancelActivity channel
// * your HandleCoordinatedActivityHandler is responsible for responding to messages on ToCancelActivity, by stopping, acking the cancel to swf, and sending on ToAckCancel
// * if your HandleCoordinatedActivityHandler wishes to stop heartbeats, send on ToStopHeartbeating and recieve on ToAckStopHeartbeating.

func NewCoordinatedActivityHandler(swfOps SWFOps, heartbeatInterval time.Duration, heartbeatErrorThreshold int, handler HandleCoordinatedActivity) LongRunningActivityHandlerFunc {
	coordinator := &LongRunningActivityCoordinator{
		HeartbeatInterval:     heartbeatInterval,
		HeartbeatErrors:       make(chan error, heartbeatErrorThreshold),
		ToCancelActivity:      make(chan struct{}),
		ToAckCancelActivity:   make(chan struct{}),
		ToStopHeartbeating:    make(chan struct{}),
		ToAckStopHeartbeating: make(chan struct{}),
	}

	return func(activityTask *swf.ActivityTask, input interface{}) {
		go handler(coordinator, activityTask, input)
		for {
			select {
			case <-time.After(heartbeatInterval):
				status, err := swfOps.RecordActivityTaskHeartbeat(&swf.RecordActivityTaskHeartbeatInput{
					TaskToken: activityTask.TaskToken,
				})
				if err != nil {
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
}
