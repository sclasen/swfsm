package activity

import (
	"testing"


	"github.com/awslabs/aws-sdk-go/gen/swf"
	. "github.com/sclasen/swfsm/sugar"
	"time"
)

func TestNewLongRunningActivityHandler(t *testing.T) {
	hc := func(c *LongRunningActivityCoordinator, ta *swf.ActivityTask, d *TestInput) {
		t.Log("HANDLE")
		c.ToStopHeartbeating <- struct{}{}
		<-c.ToAckStopHeartbeating
	}

	handler := NewCoordinatedActivityHandler("activity", hc)


	worker := ActivityWorker{

	}
	worker.AddCoordinatedHandler(10 * time.Second, 100, handler)
    worker.Init()
	input, _ := worker.Serializer.Serialize(&TestInput{Name:"Foo"})
	worker.handleActivityTask(&swf.ActivityTask{
		TaskToken: S("token"),
        WorkflowExecution: &swf.WorkflowExecution{},
        ActivityType: &swf.ActivityType{
			Name: S("activity"),
		},
		Input: S(input),
	})
}
