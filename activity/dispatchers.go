package activity

import "github.com/awslabs/aws-sdk-go/gen/swf"

//ActivityTaskDispatcher is used by the FSM machinery to
type ActivityTaskDispatcher interface {
	DispatchTask(*swf.ActivityTask, func(*swf.ActivityTask))
}

//CallingGoroutineDispatcher is a DecisionTaskDispatcher that runs the decision handler in the polling goroutine
type CallingGoroutineDispatcher struct{}

//DispatchTask calls the handler in the same goroutine.
func (*CallingGoroutineDispatcher) DispatchTask(task *swf.ActivityTask, handler func(*swf.ActivityTask)) {
	handler(task)
}

//NewGoroutineDispatcher is a DecisionTaskDispatcher that runs the decision handler in a new goroutine.
type NewGoroutineDispatcher struct {
}

//DispatchTask calls the handler in a new  goroutine.
func (*NewGoroutineDispatcher) DispatchTask(task *swf.ActivityTask, handler func(*swf.ActivityTask)) {
	go handler(task)
}

//BoundedGoroutineDispatcher is a DecisionTaskDispatcher that uses a bounded number of goroutines to run decision handlers.
type BoundedGoroutineDispatcher struct {
	NumGoroutines int
	started       bool
	tasks         chan *swf.ActivityTask
}

//DispatchTask calls sends the task on a channel that NumGoroutines goroutines are selecting on.
//Goroutines recieving a task run it in the same goroutine.
//note that this is unsynchronized as DispatchTask will only be called by the single poller goroutine.
func (b *BoundedGoroutineDispatcher) DispatchTask(task *swf.ActivityTask, handler func(*swf.ActivityTask)) {

	if !b.started {
		if b.NumGoroutines == 0 {
			//use at least 1
			b.NumGoroutines = 1
		}
		b.tasks = make(chan *swf.ActivityTask)
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
