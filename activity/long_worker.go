package activity

import "github.com/awslabs/aws-sdk-go/gen/swf"

//LongRunningActivityFunc that creates all the coordination channels, starts heartbeating, and calls into
type LongRunningActivityCoordinator struct {
	ToStopActivity        chan struct{}
	ToAckStopActivity     chan struct{}
	ToStopHeartbeating    chan struct{}
	ToAckStopHeartbeating chan struct{}
}

type HandleCoordinatedActivity func(*LongRunningActivityCoordinator, *swf.ActivityTask, interface{})
