package activity

import (
	"errors"
	"log"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/swf"
	. "github.com/sclasen/swfsm/sugar"
)

func TestCoordinatedActivityHandler_Complete(t *testing.T) {
	hc := &TestCoordinatedTaskHandler{
		t: t,
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
	worker.AddCoordinatedHandler(5*time.Millisecond, 1*time.Millisecond, handler)
	worker.Init()
	input, _ := worker.Serializer.Serialize(&TestInput{Name: "Foo"})

	go worker.HandleActivityTask(&swf.PollForActivityTaskOutput{
		TaskToken:         S("token"),
		WorkflowExecution: &swf.WorkflowExecution{},
		ActivityType: &swf.ActivityType{
			Name: S("activity"),
		},
		ActivityId: S("id"),
		Input:      S(input),
	})

	time.Sleep(50 * time.Millisecond)

	hc.stop = true

	time.Sleep(100 * time.Millisecond)

	if !mockSwf.CompletedSet {
		t.Fatal("Not Completed")
	}
}

func TestCoordinatedActivityHandler_Cancel(t *testing.T) {
	hc := &TestCoordinatedTaskHandler{
		t: t,
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

	t.Log("test cancel")
	worker.AddCoordinatedHandler(5*time.Millisecond, 1*time.Millisecond, handler)
	worker.Init()
	input, _ := worker.Serializer.Serialize(&TestInput{Name: "Foo"})

	go worker.HandleActivityTask(&swf.PollForActivityTaskOutput{
		TaskToken:         S("token"),
		WorkflowExecution: &swf.WorkflowExecution{},
		ActivityType: &swf.ActivityType{
			Name: S("activity"),
		},
		ActivityId: S("id"),
		Input:      S(input),
	})

	time.Sleep(30 * time.Millisecond)

	mockSwf.Canceled = true

	time.Sleep(500 * time.Millisecond)

	if !hc.canceled {
		t.Fatal("Not Canceled")
	}
}

func TestCoordinatedActivityHandler_StartError(t *testing.T) {
	hc := &TestCoordinatedTaskHandler{
		t:        t,
		startErr: errors.New("start failed"),
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

	worker.AddCoordinatedHandler(5*time.Millisecond, 1*time.Millisecond, handler)
	worker.Init()
	input, _ := worker.Serializer.Serialize(&TestInput{Name: "Foo"})

	go worker.HandleActivityTask(&swf.PollForActivityTaskOutput{
		TaskToken:         S("token"),
		WorkflowExecution: &swf.WorkflowExecution{},
		ActivityType: &swf.ActivityType{
			Name: S("activity"),
		},
		ActivityId: S("id"),
		Input:      S(input),
	})

	time.Sleep(100 * time.Millisecond)

	if !mockSwf.Failed {
		t.Fatal("did not fail")
	}
}

func TestTypedCoordinatedActivityHandler_Complete(t *testing.T) {
	hc := &TypedCoordinatedTaskHandler{
		t: t,
	}

	handler := NewCoordinatedActivityHandler("activity", hc.Begin, hc.Work, hc.Stop)

	mockSwf := &MockSWF{}
	worker := ActivityWorker{
		SWF: mockSwf,
	}

	t.Log("test complete")
	worker.AddCoordinatedHandler(5*time.Millisecond, 1*time.Millisecond, handler)
	worker.Init()
	input, _ := worker.Serializer.Serialize(&TestInput{Name: "Foo"})

	go worker.HandleActivityTask(&swf.PollForActivityTaskOutput{
		TaskToken:         S("token"),
		WorkflowExecution: &swf.WorkflowExecution{},
		ActivityType: &swf.ActivityType{
			Name: S("activity"),
		},
		ActivityId: S("id"),
		Input:      S(input),
	})

	time.Sleep(50 * time.Millisecond)

	hc.stop = true

	time.Sleep(100 * time.Millisecond)

	if !mockSwf.CompletedSet {
		t.Fatal("Not Completed")
	}
}

func TestTypedCoordinatedActivityHandler_Cancel(t *testing.T) {
	hc := &TypedCoordinatedTaskHandler{
		t: t,
	}

	handler := NewCoordinatedActivityHandler("activity", hc.Begin, hc.Work, hc.Stop)

	mockSwf := &MockSWF{}
	worker := ActivityWorker{
		SWF: mockSwf,
	}

	t.Log("test cancel")
	worker.AddCoordinatedHandler(5*time.Millisecond, 1*time.Millisecond, handler)
	worker.Init()
	input, _ := worker.Serializer.Serialize(&TestInput{Name: "Foo"})

	go worker.HandleActivityTask(&swf.PollForActivityTaskOutput{
		TaskToken:         S("token"),
		WorkflowExecution: &swf.WorkflowExecution{},
		ActivityType: &swf.ActivityType{
			Name: S("activity"),
		},
		ActivityId: S("id"),
		Input:      S(input),
	})

	time.Sleep(30 * time.Millisecond)

	mockSwf.Canceled = true

	time.Sleep(500 * time.Millisecond)

	if !hc.canceled {
		t.Fatal("was not canceled")
	}
}

func TestTypedCoordinatedActivityHandler_StartError(t *testing.T) {
	hc := &TypedCoordinatedTaskHandler{
		t:        t,
		startErr: errors.New("start failed"),
	}

	handler := NewCoordinatedActivityHandler("activity", hc.Begin, hc.Work, hc.Stop)

	mockSwf := &MockSWF{}
	worker := ActivityWorker{
		SWF: mockSwf,
	}

	worker.AddCoordinatedHandler(5*time.Millisecond, 1*time.Millisecond, handler)
	worker.Init()
	input, _ := worker.Serializer.Serialize(&TestInput{Name: "Foo"})

	go worker.HandleActivityTask(&swf.PollForActivityTaskOutput{
		TaskToken:         S("token"),
		WorkflowExecution: &swf.WorkflowExecution{},
		ActivityType: &swf.ActivityType{
			Name: S("activity"),
		},
		ActivityId: S("id"),
		Input:      S(input),
	})

	time.Sleep(500 * time.Millisecond)

	if !mockSwf.Failed {
		t.Fatal("did not fail")
	}
}

func TestTickRateLimit(t *testing.T) {
	hc := &TestCoordinatedTaskHandler{
		t: t,
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
	go worker.HandleActivityTask(&swf.PollForActivityTaskOutput{
		TaskToken:         S("token"),
		WorkflowExecution: &swf.WorkflowExecution{},
		ActivityType: &swf.ActivityType{
			Name: S("activity"),
		},
		ActivityId: S("id"),
		Input:      S(input),
	})

	// let it run for 500ms
	time.Sleep(500 * time.Millisecond)
	hc.stop = true

	time.Sleep(100 * time.Millisecond)
	if !mockSwf.CompletedSet {
		t.Fatal("Not Completed")
	}
	if hc.ticks == 0 || hc.ticks > 5 {
		t.Fatalf("must run no more than 5x in 500ms with a tickInterval of 100ms. Ticks: %d")
	}
}

type TestCoordinatedTaskHandler struct {
	t        *testing.T
	startErr error
	stop     bool
	canceled bool
	ticks    int
}

func (c *TestCoordinatedTaskHandler) Start(a *swf.PollForActivityTaskOutput, d interface{}) (interface{}, error) {
	log.Println("START")
	return nil, c.startErr
}

func (c *TestCoordinatedTaskHandler) Tick(a *swf.PollForActivityTaskOutput, d interface{}) (bool, interface{}, error) {
	c.ticks++
	c.t.Log("TICK")
	time.Sleep(100 * time.Millisecond)
	if c.stop {
		return false, &TestOutput{Name: "done"}, nil
	}
	return true, nil, nil
}

func (c *TestCoordinatedTaskHandler) Cancel(a *swf.PollForActivityTaskOutput, d interface{}) error {
	c.t.Log("CANCEL")
	c.canceled = true
	return nil
}

type TypedCoordinatedTaskHandler struct {
	t        *testing.T
	startErr error
	stop     bool
	canceled bool
}

func (c *TypedCoordinatedTaskHandler) Begin(a *swf.PollForActivityTaskOutput, d *TestInput) (*TestOutput, error) {
	c.t.Log("START")
	return nil, c.startErr
}

func (c *TypedCoordinatedTaskHandler) Work(a *swf.PollForActivityTaskOutput, d *TestInput) (bool, *TestOutput, error) {
	c.t.Log("TICK")
	time.Sleep(100 * time.Millisecond)
	if c.stop {
		return false, &TestOutput{Name: "done"}, nil
	}
	return true, nil, nil
}

func (c *TypedCoordinatedTaskHandler) Stop(a *swf.PollForActivityTaskOutput, d *TestInput) error {
	c.t.Log("CANCEL")
	c.canceled = true
	return nil
}
