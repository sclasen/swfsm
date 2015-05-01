package activity

import (
	"testing"

	"time"

	"github.com/awslabs/aws-sdk-go/gen/swf"
	. "github.com/sclasen/swfsm/sugar"
)

func TestNewLongRunningActivityHandler(t *testing.T) {
	hc := func(c *LongRunningActivityCoordinator, ta *swf.ActivityTask, d *TestInput) {
		t.Log("HANDLE")
		c.ToStopHeartbeating <- struct{}{}
		<-c.ToAckStopHeartbeating
	}
	hca := NewHandleCoordinatedActivity(hc)
	lrahf := NewCoordinatedActivityHandler(nil, 10*time.Second, 100, hca)
	handler := NewLongRunningActivityHandler("activity", lrahf)
	handler.HandlerFunc(&swf.ActivityTask{WorkflowExecution: &swf.WorkflowExecution{}, ActivityType: &swf.ActivityType{Name: S("name"), Version: S("ver")}}, &TestInput{})
}
