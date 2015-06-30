package activity

import (
	"testing"

	"time"

	"github.com/aws/aws-sdk-go/service/swf"
	. "github.com/sclasen/swfsm/sugar"
)

func TestCoordinatedActivityHandler(t *testing.T) {
	hc := &TestCoordinatedTaskHandler{
		t:        t,
		canceled: false,
	}

	handler := &CoordinatedActivityHandler{
		Input:    TestInput{},
		Activity: "activity",
		Start:    hc.Start,
		Tick:     hc.Tick,
		Cancel:   hc.Cancel,
	}

	mockSwf := &MockSWF{}
	worker := ActivityWorker{
		SWF: mockSwf,
	}

	t.Log("test complete")
	worker.AddCoordinatedHandler(1*time.Second, 1*time.Millisecond, handler)
	worker.Init()
	input, _ := worker.Serializer.Serialize(&TestInput{Name: "Foo"})
	worker.handleActivityTask(&swf.PollForActivityTaskOutput{
		TaskToken:         S("token"),
		WorkflowExecution: &swf.WorkflowExecution{},
		ActivityType: &swf.ActivityType{
			Name: S("activity"),
		},
		ActivityID: S("id"),
		Input:      S(input),
	})

	hc.cont = false
	time.Sleep(100 * time.Millisecond)
	if !mockSwf.CompletedSet {
		t.Fatal("Not Completed")
	}

	t.Log("test cancel")
	hc.cont = true
	mockSwf.CompletedSet = false
	mockSwf.Completed = nil
	mockSwf.Canceled = true

	worker.handleActivityTask(&swf.PollForActivityTaskOutput{
		TaskToken:         S("token"),
		WorkflowExecution: &swf.WorkflowExecution{},
		ActivityType: &swf.ActivityType{
			Name: S("activity"),
		},
		ActivityID: S("id"),
		Input:      S(input),
	})

	time.Sleep(100 * time.Millisecond)
	if !hc.canceled {
		t.Fatal("Not Canceled")
	}

}

func TestTypedCoordinatedActivityHandler(t *testing.T) {
	hc := &TypedCoordinatedTaskHandler{
		t:        t,
		canceled: false,
	}

	handler := NewCoordinatedActivityHandler("activity", hc.Begin, hc.Work, hc.Stop)

	mockSwf := &MockSWF{}
	worker := ActivityWorker{
		SWF: mockSwf,
	}

	t.Log("test complete")
	worker.AddCoordinatedHandler(1*time.Second, 1*time.Millisecond, handler)
	worker.Init()
	input, _ := worker.Serializer.Serialize(&TestInput{Name: "Foo"})
	worker.handleActivityTask(&swf.PollForActivityTaskOutput{
		TaskToken:         S("token"),
		WorkflowExecution: &swf.WorkflowExecution{},
		ActivityType: &swf.ActivityType{
			Name: S("activity"),
		},
		ActivityID: S("id"),
		Input:      S(input),
	})

	hc.cont = false
	time.Sleep(100 * time.Millisecond)
	if !mockSwf.CompletedSet {
		t.Fatal("Not Completed")
	}

	t.Log("test cancel")
	hc.cont = true
	mockSwf.CompletedSet = false
	mockSwf.Completed = nil
	mockSwf.Canceled = true

	worker.handleActivityTask(&swf.PollForActivityTaskOutput{
		TaskToken:         S("token"),
		WorkflowExecution: &swf.WorkflowExecution{},
		ActivityType: &swf.ActivityType{
			Name: S("activity"),
		},
		ActivityID: S("id"),
		Input:      S(input),
	})

	time.Sleep(200 * time.Millisecond)

	if !hc.canceled {
		t.Fatal("Not Canceled")
	}

}

func TestTickRateLimit(t *testing.T) {
	hc := &NoopCounterTaskHandler{
		t:        t,
		cont:     true,
		canceled: false,
	}

	handler := &CoordinatedActivityHandler{
		Input:    TestInput{},
		Activity: "activity",
		Start:    hc.Start,
		Tick:     hc.Tick,
		Cancel:   hc.Cancel,
	}

	mockSwf := &MockSWF{}
	worker := ActivityWorker{
		SWF: mockSwf,
	}

	worker.AddCoordinatedHandler(1*time.Second, 100*time.Millisecond, handler)
	worker.Init()
	input, _ := worker.Serializer.Serialize(&TestInput{Name: "Foo"})
	go worker.handleActivityTask(&swf.PollForActivityTaskOutput{
		TaskToken:         S("token"),
		WorkflowExecution: &swf.WorkflowExecution{},
		ActivityType: &swf.ActivityType{
			Name: S("activity"),
		},
		ActivityID: S("id"),
		Input:      S(input),
	})

	// let it run for 500ms
	time.Sleep(500 * time.Millisecond)
	hc.cont = false

	time.Sleep(100 * time.Millisecond)
	if !mockSwf.CompletedSet {
		t.Fatal("Not Completed")
	}
	if hc.ticks == 0 || hc.ticks > 5 {
		t.Fatalf("must run no more than 5x in 500ms with a tickInterval of 100ms. Ticks: %d")
	}
}

type TypedCoordinatedTaskHandler struct {
	t        *testing.T
	cont     bool
	canceled bool
}

func (c *TypedCoordinatedTaskHandler) Begin(a *swf.PollForActivityTaskOutput, d *TestInput) (*TestOutput, error) {
	c.t.Log("START")
	return nil, nil
}

func (c *TypedCoordinatedTaskHandler) Work(a *swf.PollForActivityTaskOutput, d *TestInput) (bool, *TestOutput, error) {
	c.t.Log("TICK")
	time.Sleep(100 * time.Millisecond)
	if c.cont {
		return true, nil, nil
	}
	return false, &TestOutput{Name: "done"}, nil
}

func (c *TypedCoordinatedTaskHandler) Stop(a *swf.PollForActivityTaskOutput, d *TestInput) error {
	c.t.Log("CANCEL")
	c.canceled = true
	return nil
}

type TestCoordinatedTaskHandler struct {
	t        *testing.T
	cont     bool
	canceled bool
}

func (c *TestCoordinatedTaskHandler) Start(a *swf.PollForActivityTaskOutput, d interface{}) (interface{}, error) {
	c.t.Log("START")
	return nil, nil
}

func (c *TestCoordinatedTaskHandler) Tick(a *swf.PollForActivityTaskOutput, d interface{}) (bool, interface{}, error) {
	c.t.Log("TICK")
	time.Sleep(100 * time.Millisecond)
	if c.cont {
		return true, nil, nil
	}
	return false, &TestOutput{Name: "done"}, nil
}

func (c *TestCoordinatedTaskHandler) Cancel(a *swf.PollForActivityTaskOutput, d interface{}) error {
	c.t.Log("CANCEL")
	c.canceled = true
	return nil
}

type NoopCounterTaskHandler struct {
	t        *testing.T
	ticks    int
	cont     bool
	canceled bool
}

func (c *NoopCounterTaskHandler) Start(a *swf.PollForActivityTaskOutput, d interface{}) (interface{}, error) {
	c.t.Log("START")
	return nil, nil
}

func (c *NoopCounterTaskHandler) Tick(a *swf.PollForActivityTaskOutput, d interface{}) (bool, interface{}, error) {
	c.t.Log("TICK")
	c.ticks++
	if c.cont {
		return true, nil, nil
	}
	return false, &TestOutput{Name: "done"}, nil
}

func (c *NoopCounterTaskHandler) Cancel(a *swf.PollForActivityTaskOutput, d interface{}) error {
	c.t.Log("CANCEL")
	c.canceled = true
	return nil
}
