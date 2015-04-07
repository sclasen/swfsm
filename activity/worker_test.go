package activity

import (
    "github.com/awslabs/aws-sdk-go/gen/swf"
    "github.com/sclasen/swfsm/fsm"
)

type MockSWF struct {
}

func (MockSWF) RecordActivityTaskHeartbeat(req *swf.RecordActivityTaskHeartbeatInput) (resp *swf.ActivityTaskStatus, err error) {
	return nil, nil
}
func (MockSWF) RespondActivityTaskCanceled(req *swf.RespondActivityTaskCanceledInput) (err error) {
	return nil
}
func (MockSWF) RespondActivityTaskCompleted(req *swf.RespondActivityTaskCompletedInput) (err error) {
	return nil
}
func (MockSWF) RespondActivityTaskFailed(req *swf.RespondActivityTaskFailedInput) (err error) {
	return nil
}
func (MockSWF) PollForActivityTask(req *swf.PollForActivityTaskInput) (resp *swf.ActivityTask, err error) {
	return nil, nil
}
func (MockSWF) PollForDecisionTask(req *swf.PollForDecisionTaskInput) (resp *swf.DecisionTask, err error) {
	return nil, nil
}


func ExampleActivityWorker(){

    var swfOps SWFOps

    taskList := "aTaskListSharedBetweenTaskOneAndTwo"

    handleTask1 := func(task *swf.ActivityTask, input interface{}) (interface{}, error){
        return input, nil
    }

    handleTask2 := func(task *swf.ActivityTask, input interface{}) (interface{}, error){
        return input, nil
    }

    handler1 := &ActivityHandler{Activity: "one", HandlerFunc: handleTask1}

    handler2 := &ActivityHandler{Activity: "two", HandlerFunc: handleTask2}

    worker := &ActivityWorker{
        Domain: "swf-domain",
        Serializer: fsm.JSONStateSerializer{},
        TaskList: taskList,
        SWF: swfOps,
        Identity: "test-activity-worker",
    }

    worker.AddHandler(handler1)

    worker.AddHandler(handler2)

    go worker.Start()

}