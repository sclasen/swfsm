package activity

import (
	"fmt"
	"sync/atomic"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/sclasen/swfsm/poller"
)

//ActivityTaskDispatcher is used by the ActivityWorker machinery to dispatch the handling of ActivityTasks.
//Different implementations can provide different concurrency models.
type ActivityTaskDispatcher interface {
	DispatchTask(context.Context, *swf.PollForActivityTaskOutput, func(context.Context, *swf.PollForActivityTaskOutput))
}

//CallingGoroutineDispatcher is a DecisionTaskDispatcher that runs the decision handler in the polling goroutine
type CallingGoroutineDispatcher struct{}

//DispatchTask calls the handler in the same goroutine.
func (*CallingGoroutineDispatcher) DispatchTask(ctx context.Context, task *swf.PollForActivityTaskOutput, handler func(context.Context, *swf.PollForActivityTaskOutput)) {
	handler(ctx, task)
}

//NewGoroutineDispatcher is a DecisionTaskDispatcher that runs the decision handler in a new goroutine.
type NewGoroutineDispatcher struct {
}

//DispatchTask calls the handler in a new  goroutine.
func (*NewGoroutineDispatcher) DispatchTask(ctx context.Context, task *swf.PollForActivityTaskOutput, handler func(context.Context, *swf.PollForActivityTaskOutput)) {
	go handler(ctx, task)
}

//BoundedGoroutineDispatcher is a DecisionTaskDispatcher that uses a bounded number of goroutines to run decision handlers.
type BoundedGoroutineDispatcher struct {
	NumGoroutines int
	started       bool
	tasks         chan *swf.PollForActivityTaskOutput
}

//DispatchTask calls sends the task on a channel that NumGoroutines goroutines are selecting on.
//Goroutines recieving a task run it in the same goroutine.
//note that this is unsynchronized as DispatchTask will only be called by the single poller goroutine.
func (b *BoundedGoroutineDispatcher) DispatchTask(ctx context.Context, task *swf.PollForActivityTaskOutput, handler func(context.Context, *swf.PollForActivityTaskOutput)) {

	if !b.started {
		if b.NumGoroutines == 0 {
			//use at least 1
			b.NumGoroutines = 1
		}
		b.tasks = make(chan *swf.PollForActivityTaskOutput)
		for i := 0; i < b.NumGoroutines; i++ {
			go func() {
				for {
					select {
					case t := <-b.tasks:
						handler(ctx, t)
					}
				}
			}()
		}
		b.started = true
	}

	b.tasks <- task
}

// CountdownGoroutineDispatcher is a dispatcher that you can register with a  ShutdownManager.  Used in your
// ActivityWorkers, it will count in-flight activities.  It doesnt ack shutdowns until the number of in-flight activities are zero.
type CountdownGoroutineDispatcher struct {
	Stop     chan bool
	StopAck  chan bool
	inFlight int64
}

var countdownDispatcherNameSeq int64

func countdownDispatcherName() string {
	seq := atomic.AddInt64(&countdownDispatcherNameSeq, 1)
	return fmt.Sprintf("countdown-%d", seq)
}

//RegisterNewCountdownGoroutineDispatcher constructs a new CountdownGoroutineDispatcher, start it and register it with the given ShutdownManager
func RegisterNewCountdownGoroutineDispatcher(mgr poller.ShutdownManager) *CountdownGoroutineDispatcher {
	g := &CountdownGoroutineDispatcher{
		Stop:    make(chan bool, 1),
		StopAck: make(chan bool, 1),
	}
	go g.Start()
	mgr.Register(countdownDispatcherName(), g.Stop, g.StopAck)
	return g
}

func (m *CountdownGoroutineDispatcher) DispatchTask(ctx context.Context, t *swf.PollForActivityTaskOutput, f func(context.Context, *swf.PollForActivityTaskOutput)) {
	//run tasks in a new goroutine
	go func() {
		atomic.AddInt64(&m.inFlight, 1)
		f(ctx, t)
		atomic.AddInt64(&m.inFlight, -1)
	}()
}

func (m *CountdownGoroutineDispatcher) Start() {
	<-m.Stop
	for atomic.LoadInt64(&m.inFlight) > 0 {
		time.Sleep(1 * time.Second)
	}
	m.StopAck <- true
}
