package activity

import (
	"sync/atomic"
	"testing"

	"github.com/awslabs/aws-sdk-go/gen/swf"

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

func testDispatcher(dispatcher ActivityTaskDispatcher, t *testing.T) {
	task := &swf.ActivityTask{}
	tasksHandled := int32(0)
	totalTasks := int32(1000)
	done := make(chan struct{}, 1)
	handler := func(d *swf.ActivityTask) {
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
