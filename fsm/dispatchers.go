package fsm

import (
	"github.com/awslabs/aws-sdk-go/gen/swf"
)

//DecisionTaskDispatcher is used by the FSM machinery to
type DecisionTaskDispatcher interface {
	DispatchTask(*swf.DecisionTask, func(*swf.DecisionTask))
}

type CallingGoroutineDispatcher struct{}

func (*CallingGoroutineDispatcher) DispatchTask(task *swf.DecisionTask, handler func(*swf.DecisionTask)) {
	handler(task)
}

type NewGoroutineDispatcher struct {
}

func (*NewGoroutineDispatcher) DispatchTask(task *swf.DecisionTask, handler func(*swf.DecisionTask)) {
	go handler(task)
}

type BoundedGoroutineDispatcher struct {
	NumGoroutines int
	started       bool
	tasks         chan *swf.DecisionTask
}

//note that this is unsynchronized as DispatchTask will only be called by the single poller goroutine.
func (b *BoundedGoroutineDispatcher) DispatchTask(task *swf.DecisionTask, handler func(*swf.DecisionTask)) {

	if !b.started {
		if b.NumGoroutines == 0 {
			//use at least 1
			b.NumGoroutines = 1
		}
		b.tasks = make(chan *swf.DecisionTask)
		for i := 0; i < b.NumGoroutines; i++ {
			go func() {
				for {
					select {
					case t := <-b.tasks:
						handler(t)
					}
				}
			}()
		}
		b.started = true
	}

	b.tasks <- task
}
