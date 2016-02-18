package fsm

import (
	"sync"

	"github.com/aws/aws-sdk-go/service/swf"
)

//DecisionTaskDispatcher is used by the FSM machinery to
type DecisionTaskDispatcher interface {
	DispatchTask(*swf.PollForDecisionTaskOutput, func(*swf.PollForDecisionTaskOutput))
}

//CallingGoroutineDispatcher is a DecisionTaskDispatcher that runs the decision handler in the polling goroutine
type CallingGoroutineDispatcher struct{}

//DispatchTask calls the handler in the same goroutine.
func (*CallingGoroutineDispatcher) DispatchTask(task *swf.PollForDecisionTaskOutput, handler func(*swf.PollForDecisionTaskOutput)) {
	handler(task)
}

//NewGoroutineDispatcher is a DecisionTaskDispatcher that runs the decision handler in a new goroutine.
type NewGoroutineDispatcher struct {
}

//DispatchTask calls the handler in a new  goroutine.
func (*NewGoroutineDispatcher) DispatchTask(task *swf.PollForDecisionTaskOutput, handler func(*swf.PollForDecisionTaskOutput)) {
	go handler(task)
}

//BoundedGoroutineDispatcher is a DecisionTaskDispatcher that uses a bounded number of goroutines to run decision handlers.
type BoundedGoroutineDispatcher struct {
	NumGoroutines int
	started       bool
	tasks         chan *swf.PollForDecisionTaskOutput
}

//DispatchTask calls sends the task on a channel that NumGoroutines goroutines are selecting on.
//Goroutines recieving a task run it in the same goroutine.
//note that this is unsynchronized as DispatchTask will only be called by the single poller goroutine.
func (b *BoundedGoroutineDispatcher) DispatchTask(task *swf.PollForDecisionTaskOutput, handler func(*swf.PollForDecisionTaskOutput)) {

	if !b.started {
		if b.NumGoroutines == 0 {
			//use at least 1
			b.NumGoroutines = 1
		}
		b.tasks = make(chan *swf.PollForDecisionTaskOutput)
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

//GoroutinePerWorkflowDispatcher allows a single goroutine per workflow execution (RunID) to run at a time.
//Tasks are queued for each workflow execution.
//Any workflow execution with maxPendingTasks can cause DispatchTask to block until at least one of them gets handled.
func GoroutinePerWorkflowDispatcher(maxPendingTasks int) DecisionTaskDispatcher {
	return &goroutinePerWorkflowDispatcher{
		tasks:           make(map[string]chan *swf.PollForDecisionTaskOutput),
		tasksBufferSize: maxPendingTasks,
	}
}

type goroutinePerWorkflowDispatcher struct {
	tasks           map[string]chan *swf.PollForDecisionTaskOutput
	tasksBufferSize int
	tasksMux        sync.Mutex
}

func (b *goroutinePerWorkflowDispatcher) DispatchTask(task *swf.PollForDecisionTaskOutput, handler func(*swf.PollForDecisionTaskOutput)) {
	tasks := b.tasksFor(*task.WorkflowExecution.RunId)
	go func(queue chan *swf.PollForDecisionTaskOutput) {
		handler(<-queue)
	}(tasks)
	tasks <- task // can block if there are maxPendingTasks for this workflow execution
}

func (b *goroutinePerWorkflowDispatcher) tasksFor(workflowRunID string) chan *swf.PollForDecisionTaskOutput {
	b.tasksMux.Lock()
	defer b.tasksMux.Unlock()

	tasks, ok := b.tasks[workflowRunID]
	if !ok {
		tasks = make(chan *swf.PollForDecisionTaskOutput, b.tasksBufferSize)
		b.tasks[workflowRunID] = tasks
	}
	return tasks
}
