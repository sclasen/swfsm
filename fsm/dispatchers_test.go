package fsm

import (
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/swf"

	"time"
)

func TestCallingGoroutineDispatcher(t *testing.T) {
	testDispatcher(&CallingGoroutineDispatcher{}, t)
}

func TestNewGoroutineDispatcher(t *testing.T) {
	testDispatcher(&NewGoroutineDispatcher{}, t)
}
func TestBoundedGoroutineDispatcher(t *testing.T) {
	testDispatcher(&BoundedGoroutineDispatcher{NumGoroutines: 8}, t)
}
func TestGoroutinePerWorkflowDispatcher(t *testing.T) {
	testDispatcher(GoroutinePerWorkflowDispatcher(1000), t)
}
func TestGoroutinePerWorkflowDispatcherUnbuffered(t *testing.T) {
	testDispatcher(GoroutinePerWorkflowDispatcher(0), t)
}

func testDispatcher(dispatcher DecisionTaskDispatcher, t *testing.T) {
	task := &swf.PollForDecisionTaskOutput{
		WorkflowExecution: &swf.WorkflowExecution{
			RunId: aws.String("workflow-dummy"),
		},
	}
	tasksHandled := int32(0)
	totalTasks := int32(1000)
	done := make(chan struct{}, 1)
	handler := func(d *swf.PollForDecisionTaskOutput) {
		handled := atomic.AddInt32(&tasksHandled, 1)
		if handled == totalTasks {
			done <- struct{}{}
		}
	}

	for i := int32(0); i < totalTasks; i++ {
		dispatcher.DispatchTask(task, handler)
	}

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for tasks. Only completed:", tasksHandled)
	}
}
