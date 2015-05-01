package activity

import (
	"testing"

	"time"

	"github.com/awslabs/aws-sdk-go/gen/swf"
)

func TestNewLongRunningActivityHandler(t *testing.T) {
	handler := func(c *LongRunningActivityCoordinator, t *swf.ActivityTask, d *TestInput) {}
	hc := NewHandleCoordinatedActivity(handler)
	lrahf := NewCoordinatedActivityHandler(nil, 10*time.Second, 100, hc)
	NewLongRunningActivityHandler("activity", lrahf)
}
