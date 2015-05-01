package activity

import (
	"testing"

	"github.com/awslabs/aws-sdk-go/gen/swf"
)

func TestNewLongRunningActivityHandler(t *testing.T) {
	handler := func(c *LongRunningActivityCoordinator, t *swf.ActivityTask, d *TestInput) {}
	NewHandleCoordinatedActivity(handler)
}
