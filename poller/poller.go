package poller

import (
	"log"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/swf"
	"github.com/juju/errors"
	. "github.com/sclasen/swfsm/sugar"
)

// SWFOps is the subset of the swf.SWF api used bu pollers
type DecisionOps interface {
	PollForDecisionTask(req *swf.PollForDecisionTaskInput) (resp *swf.DecisionTask, err error)
}

type ActivityOps interface {
	PollForActivityTask(req *swf.PollForActivityTaskInput) (resp *swf.ActivityTask, err error)
}

// NewDecisionTaskPoller returns a DecisionTaskPoller whick can be used to poll the given task list.
func NewDecisionTaskPoller(dwc DecisionOps, domain string, identity string, taskList string) *DecisionTaskPoller {
	return &DecisionTaskPoller{
		client:   dwc,
		Domain:   domain,
		Identity: identity,
		TaskList: taskList,
	}
}

// DecisionTaskPoller polls a given task list in a domain for decision tasks.
type DecisionTaskPoller struct {
	client   DecisionOps
	Identity string
	Domain   string
	TaskList string
}

// Poll polls the task list for a task. If there is no task available, nil is
// returned. If an error is encountered, no task is returned.
func (p *DecisionTaskPoller) Poll() (*swf.DecisionTask, error) {
	resp, err := p.client.PollForDecisionTask(&swf.PollForDecisionTaskInput{
		Domain:       aws.String(p.Domain),
		Identity:     aws.String(p.Identity),
		ReverseOrder: aws.True(),
		TaskList:     &swf.TaskList{Name: aws.String(p.TaskList)},
	})
	if err != nil {
		log.Printf("component=DecisionTaskPoller at=error error=%s", err.Error())
		return nil, errors.Trace(err)
	}
	if resp.TaskToken != nil {
		log.Printf("component=DecisionTaskPoller at=decision-task-recieved workflow=%s", LS(resp.WorkflowType.Name))
		p.logTaskLatency(resp)
		return resp, nil
	}
	log.Println("component=DecisionTaskPoller at=decision-task-empty-response")
	return nil, nil
}

// PollUntilShutdownBy will poll until signaled to shutdown by the PollerShutdownManager. this func blocks, so run it in a goroutine if necessary.
// The implementation calls Poll() and invokes the callback whenever a valid PollForDecisionTaskResponse is received.
func (p *DecisionTaskPoller) PollUntilShutdownBy(mgr *ShutdownManager, pollerName string, onTask func(*swf.DecisionTask)) {
	stop := make(chan bool, 1)
	stopAck := make(chan bool, 1)
	mgr.Register(pollerName, stop, stopAck)
	for {
		select {
		case <-stop:
			log.Printf("component=DecisionTaskPoller fn=PollUntilShutdownBy at=recieved-stop action=shutting-down poller=%s", pollerName)
			stopAck <- true
			return
		default:
			task, err := p.Poll()
			if err != nil {
				log.Printf("component=DecisionTaskPoller fn=PollUntilShutdownBy at=poll-err  poller=%s error=%q", pollerName, err)
				continue
			}
			if task == nil {
				log.Printf("component=DecisionTaskPoller fn=PollUntilShutdownBy at=poll-no-task  poller=%s", pollerName)
				continue
			}
			onTask(task)
		}
	}
}

func (p *DecisionTaskPoller) logTaskLatency(resp *swf.DecisionTask) {
	for _, e := range resp.Events {
		if e.EventID == resp.StartedEventID {
			elapsed := time.Since(e.EventTimestamp.Time)
			log.Printf("component=DecisionTaskPoller at=decision-task-latency latency=%s workflow=%s", elapsed, LS(resp.WorkflowType.Name))
		}
	}
}

// NewActivityTaskPoller returns an ActivityTaskPoller.
func NewActivityTaskPoller(awc ActivityOps, domain string, identity string, taskList string) *ActivityTaskPoller {
	return &ActivityTaskPoller{
		client:   awc,
		Domain:   domain,
		Identity: identity,
		TaskList: taskList,
	}
}

// ActivityTaskPoller polls a given task list in a domain for activity tasks, and sends tasks on its Tasks channel.
type ActivityTaskPoller struct {
	client   ActivityOps
	Identity string
	Domain   string
	TaskList string
}

// Poll polls the task list for a task. If there is no task, nil is returned.
// If an error is encountered, no task is returned.
func (p *ActivityTaskPoller) Poll() (*swf.ActivityTask, error) {
	resp, err := p.client.PollForActivityTask(&swf.PollForActivityTaskInput{
		Domain:   aws.String(p.Domain),
		Identity: aws.String(p.Identity),
		TaskList: &swf.TaskList{Name: aws.String(p.TaskList)},
	})
	if err != nil {
		log.Printf("component=ActivityTaskPoller at=error error=%s", err.Error())
		return nil, errors.Trace(err)
	}
	if resp.TaskToken != nil {
		log.Printf("component=ActivityTaskPoller at=activity-task-recieved activity=%s", LS(resp.ActivityType.Name))
		return resp, nil
	}
	log.Println("component=ActivityTaskPoller at=activity-task-empty-response")
	return nil, nil
}

// PollUntilShutdownBy will poll until signaled to shutdown by the ShutdownManager. this func blocks, so run it in a goroutine if necessary.
// The implementation calls Poll() and invokes the callback whenever a valid PollForActivityTaskResponse is received.
func (p *ActivityTaskPoller) PollUntilShutdownBy(mgr *ShutdownManager, pollerName string, onTask func(*swf.ActivityTask)) {
	stop := make(chan bool, 1)
	stopAck := make(chan bool, 1)
	mgr.Register(pollerName, stop, stopAck)
	for {
		select {
		case <-stop:
			log.Printf("component=ActivityTaskPoller fn=PollUntilShutdownBy at=recieved-stop action=shutting-down poller=%s", pollerName)
			stopAck <- true
			return
		default:
			task, err := p.Poll()
			if err != nil {
				log.Printf("component=ActivityTaskPoller fn=PollUntilShutdownBy at=poll-err  poller=%s error=%q", pollerName, err)
				continue
			}
			if task == nil {
				log.Printf("component=ActivityTaskPoller fn=PollUntilShutdownBy at=poll-no-task  poller=%s", pollerName)
				continue
			}
			onTask(task)
		}
	}
}

// ShutdownManager facilitates cleanly shutting down pollers when the application decides to exit. When StopPollers() is called it will
// send to each of the stopChan that have been registered, then recieve from each of the ackChan that have been registered. At this point StopPollers() returns.
type ShutdownManager struct {
	registeredPollers map[string]*registeredPoller
}

type registeredPoller struct {
	name           string
	stopChannel    chan bool
	stopAckChannel chan bool
}

// NewShutdownManager creates a ShutdownManager
func NewShutdownManager() *ShutdownManager {

	mgr := &ShutdownManager{
		registeredPollers: make(map[string]*registeredPoller),
	}

	return mgr

}

//StopPollers blocks until it is able to stop all the registered pollers, which can take up to 60 seconds.
func (p *ShutdownManager) StopPollers() {
	log.Printf("component=PollerShutdownManager at=stop-pollers")
	for _, r := range p.registeredPollers {
		log.Printf("component=PollerShutdownManager at=sending-stop name=%s", r.name)
		r.stopChannel <- true
	}
	for _, r := range p.registeredPollers {
		log.Printf("component=PollerShutdownManager at=awaiting-stop-ack name=%s", r.name)
		<-r.stopAckChannel
		log.Printf("component=PollerShutdownManager at=stop-ack name=%s", r.name)
	}
}

// Register registers a named pair of channels to the shutdown manager. Buffered channels please!
func (p *ShutdownManager) Register(name string, stopChan chan bool, ackChan chan bool) {
	p.registeredPollers[name] = &registeredPoller{name, stopChan, ackChan}
}

// Deregister removes a registered pair of channels from the shutdown manager.
func (p *ShutdownManager) Deregister(name string) {
	delete(p.registeredPollers, name)
}
