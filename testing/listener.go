package testing

import (
	"sync"
	"time"

	"fmt"

	"reflect"

	"strings"

	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/pborman/uuid"
	"github.com/sclasen/swfsm/activity"
	"github.com/sclasen/swfsm/fsm"
)

const interestChannelSize = 1000

type TestAdapter interface {
	TestName() string
	Logf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	FailNow()
}

type TestConfig struct {
	Testing               TestAdapter
	FSM                   *fsm.FSM
	StubFSM               *fsm.FSM
	Workers               []*activity.ActivityWorker
	StubbedWorkflows      []string
	ShortStubbedWorkflows []string
	DefaultWaitTimeout    int
	FailActivitiesOnce    bool
	ThrottleSignalsOnce   bool
	ThrottleCancelsOnce   bool
	ThrottleChildrenOnce  bool
	ThrottleTimersOnce    bool
}

func NewTestListener(t TestConfig) *TestListener {
	if t.DefaultWaitTimeout == 0 {
		t.DefaultWaitTimeout = 10
	}

	tl := &TestListener{
		decisionOutcomes: make(chan DecisionOutcome),
		historyInterest:  make(map[string]chan *swf.HistoryEvent),
		decisionInterest: make(map[string]chan *swf.Decision),
		stateInterest:    make(map[string]chan string),
		dataInterest:     make(map[string]chan *StateData),
		DefaultWait:      time.Duration(t.DefaultWaitTimeout) * time.Second,
		TestId:           strings.Join([]string{t.Testing.TestName(), uuid.New()}, "-"),
		testAdapter:      t.Testing,
		dataType:         t.FSM.DataType,
		serializer:       t.FSM.Serializer,
	}

	t.FSM.ReplicationHandler = TestReplicator(tl.decisionOutcomes)

	interceptors := []fsm.DecisionInterceptor{
		t.FSM.DecisionInterceptor,
		TestDecisionInterceptor(tl.TestId, t.StubbedWorkflows, t.ShortStubbedWorkflows),
	}

	if t.ThrottleCancelsOnce {
		interceptors = append(interceptors, TestThrotteCancelsOnceInterceptor())
	}

	if t.ThrottleChildrenOnce {
		interceptors = append(interceptors, TestThrotteChildrenOnceInterceptor())
	}

	if t.ThrottleSignalsOnce {
		interceptors = append(interceptors, TestThrotteSignalsOnceInterceptor())
	}

	if t.ThrottleTimersOnce {
		interceptors = append(interceptors, TestThrotteTimersOnceInterceptor(10))
	}

	t.FSM.DecisionInterceptor = fsm.NewComposedDecisionInterceptor(interceptors...)

	t.FSM.TaskList = tl.TestFsmTaskList()

	if t.StubFSM != nil {
		t.StubFSM.ReplicationHandler = TestReplicator(tl.decisionOutcomes)
	}

	for _, w := range t.Workers {
		if t.FailActivitiesOnce {
			w.ActivityInterceptor = activity.NewComposedDecisionInterceptor(
				w.ActivityInterceptor,
				TestFailOnceActivityInterceptor(),
			)
		}

		w.TaskList = tl.TestWorkerTaskList(w)
		w.AllowPanics = true
	}

	tl.Start()
	return tl
}

type TestListener struct {
	decisionOutcomes chan DecisionOutcome
	historyInterest  map[string]chan *swf.HistoryEvent
	historyLock      sync.Mutex
	decisionInterest map[string]chan *swf.Decision
	decisionLock     sync.Mutex
	stateInterest    map[string]chan string
	stateLock        sync.Mutex
	dataInterest     map[string]chan *StateData
	dataLock         sync.Mutex
	DefaultWait      time.Duration
	testAdapter      TestAdapter
	TestId           string
	dataType         interface{}
	serializer       fsm.StateSerializer
}

func (tl *TestListener) TestFsmTaskList() string {
	return tl.TestId
}

func (tl *TestListener) TestWorkerTaskList(w *activity.ActivityWorker) string {
	return w.TaskList + tl.TestId
}

func (tl *TestListener) getHistoryInterest(workflowId string) chan *swf.HistoryEvent {
	defer tl.historyLock.Unlock()
	tl.historyLock.Lock()
	historyChan, ok := tl.historyInterest[workflowId]
	if !ok {
		historyChan = make(chan *swf.HistoryEvent, interestChannelSize)
		tl.historyInterest[workflowId] = historyChan
	}
	return historyChan
}

func (tl *TestListener) getDecisionInterest(workflowId string) chan *swf.Decision {
	defer tl.decisionLock.Unlock()
	tl.decisionLock.Lock()
	decisionChan, ok := tl.decisionInterest[workflowId]
	if !ok {
		decisionChan = make(chan *swf.Decision, interestChannelSize)
		tl.decisionInterest[workflowId] = decisionChan
	}
	return decisionChan
}

func (tl *TestListener) getStateInterest(workflowId string) chan string {
	defer tl.stateLock.Unlock()
	tl.stateLock.Lock()
	stateChan, ok := tl.stateInterest[workflowId]
	if !ok {
		stateChan = make(chan string, interestChannelSize)
		tl.stateInterest[workflowId] = stateChan
	}
	return stateChan
}

func (tl *TestListener) getDataInterest(workflowId string) chan *StateData {
	defer tl.dataLock.Unlock()
	tl.dataLock.Lock()
	dataChan, ok := tl.dataInterest[workflowId]
	if !ok {
		dataChan = make(chan *StateData, interestChannelSize)
		tl.dataInterest[workflowId] = dataChan
	}
	return dataChan
}

func (tl *TestListener) RegisterHistoryInterest(workflowId string) {
	tl.testAdapter.Logf("DEPRECATED: Registration happens automatically now.")
}

func (tl *TestListener) RegisterDecisionInterest(workflowId string) {
	tl.testAdapter.Logf("DEPRECATED: Registration happens automatically now.")
}

func (tl *TestListener) RegisterStateInterest(workflowId string) {
	tl.testAdapter.Logf("DEPRECATED: Registration happens automatically now.")
}

func (tl *TestListener) RegisterDataInterest(workflowId string) {
	tl.testAdapter.Logf("DEPRECATED: Registration happens automatically now.")
}

func (tl *TestListener) AwaitStateFor(workflowId, state string, waitFor time.Duration) {
	ch := tl.getStateInterest(workflowId)
	timer := time.After(waitFor)

	for {
		select {
		case s := <-ch:
			tl.testAdapter.Logf("TestListener: await state for workflow=%s state=%s received-state=%s", workflowId, state, s)
			if s == state {
				return
			}
		case <-timer:
			panic(fmt.Sprintf("TestListener: timed out waiting for workflow=%s state=%s", workflowId, state))
			tl.testAdapter.Fatalf("TestListener: timed out waiting for workflow=%s state=%s", workflowId, state)
			tl.testAdapter.FailNow()
		}
	}
}

func (tl *TestListener) AwaitState(workflowId, state string) {
	tl.AwaitStateFor(workflowId, state, tl.DefaultWait)
}

func (tl *TestListener) AwaitEventFor(workflowId string, waitFor time.Duration, predicate func(*swf.HistoryEvent) bool) {
	ch := tl.getHistoryInterest(workflowId)
	timer := time.After(waitFor)
	for {
		select {
		case h := <-ch:
			if predicate(h) {
				tl.testAdapter.Logf("TestListener: await event for workflow=%s received-event=%s predicate=true", workflowId, *h.EventType)
				return
			} else {
				tl.testAdapter.Logf("TestListener: await event for workflow=%s received-event=%s predicate=false", workflowId, *h.EventType)
			}
		case <-timer:
			panic(fmt.Sprintf("TestListener: timed out waiting for workflow=%s event", workflowId))
			tl.testAdapter.Fatalf("TestListener: timed out waiting for workflow=%s event", workflowId)
			tl.testAdapter.FailNow()
		}
	}
}

func (tl *TestListener) AwaitEvent(workflowId string, predicate func(*swf.HistoryEvent) bool) {
	tl.AwaitEventFor(workflowId, tl.DefaultWait, predicate)
}

func (tl *TestListener) AwaitDecisionFor(workflowId string, waitFor time.Duration, predicate func(*swf.Decision) bool) {
	ch := tl.getDecisionInterest(workflowId)
	timer := time.After(waitFor)
	for {
		select {
		case h := <-ch:
			if predicate(h) {
				tl.testAdapter.Logf("TestListener: await decision for workflow=%s received-decision=%s predicate=true", workflowId, *h.DecisionType)
				return
			} else {
				tl.testAdapter.Logf("TestListener: await decision for workflow=%s received-decision=%s predicate=false", workflowId, *h.DecisionType)
			}
		case <-timer:
			panic(fmt.Sprintf("TestListener: timed out waiting for workflow=%s decision", workflowId))
			tl.testAdapter.Fatalf("TestListener: timed out waiting for workflow=%s decision", workflowId)
			tl.testAdapter.FailNow()
		}
	}
}

func (tl *TestListener) AwaitDecision(workflowId string, predicate func(*swf.Decision) bool) {
	tl.AwaitDecisionFor(workflowId, tl.DefaultWait, predicate)
}

func (tl *TestListener) AwaitDataFor(workflowId string, waitFor time.Duration, predicate func(*StateData) bool) {
	ch := tl.getDataInterest(workflowId)
	timer := time.After(waitFor)
	for {
		select {
		case d := <-ch:
			if predicate(d) {
				tl.testAdapter.Logf("TestListener: await data for workflow=%s received-data=%+v predicate=true", workflowId, d)
				return
			} else {
				tl.testAdapter.Logf("TestListener: await data for workflow=%s received-data=%+v predicate=false", workflowId, d)
			}
		case <-timer:
			panic(fmt.Sprintf("TestListener: timed out waiting for workflow=%s data", workflowId))
			tl.testAdapter.Fatalf("TestListener: timed out waiting for workflow=%s data", workflowId)
			tl.testAdapter.FailNow()
		}
	}
}

func (tl *TestListener) AwaitData(workflowId string, predicate func(*StateData) bool) {
	tl.AwaitDataFor(workflowId, tl.DefaultWait, predicate)
}

func (tl *TestListener) Start() {
	tl.testAdapter.Logf("TestListener: Starting")
	go tl.forward()
}

func (tl *TestListener) Stop() {
	close(tl.decisionOutcomes)
}

func (tl *TestListener) forward() {
	tl.testAdapter.Logf("TestListener: Forwarding")
	for {
		select {
		case do, ok := <-tl.decisionOutcomes:
			tl.testAdapter.Logf("TestListener: ")
			if !ok {
				tl.testAdapter.Logf("TestListener: decisionOutcomes closed!!!!!!!!")
				return
			}

			workflowId := *do.DecisionTask.WorkflowExecution.WorkflowId
			tl.testAdapter.Logf("TestListener: DecisionOutcome for workflow %s", workflowId)

			//send history events
			historyChan := tl.getHistoryInterest(workflowId)
			for i := len(do.DecisionTask.Events) - 1; i >= 0; i-- {
				event := do.DecisionTask.Events[i]
				if *event.EventId > *do.DecisionTask.PreviousStartedEventId {
					select {
					case historyChan <- do.DecisionTask.Events[i]:
					default:
						tl.testAdapter.Fatalf("historyChan full for %s", workflowId)
					}
				}
			}

			//send decisions
			decisionChan := tl.getDecisionInterest(workflowId)
			for _, d := range do.Decisions {
				select {
				case decisionChan <- d:
				default:
					tl.testAdapter.Fatalf("decisionChan full for %s", workflowId)
				}
			}

			//send states
			stateChan := tl.getStateInterest(workflowId)
			select {
			case stateChan <- do.State.StateName:
			default:
				tl.testAdapter.Fatalf("stateChan full for %s", workflowId)
			}

			//send data
			dataChan := tl.getDataInterest(workflowId)
			stateData := &StateData{
				State: do.State.StateName,
				Data:  tl.deserialize(do.State.StateData),
			}
			select {
			case dataChan <- stateData:
			default:
				tl.testAdapter.Fatalf("dataChan full for %s", workflowId)
			}

		case <-time.After(1 * time.Second):
			tl.testAdapter.Logf("TestListener: warn, no DecisionOutcomes after 1 second")
		}
	}
}

func (tl *TestListener) deserialize(serialized string) interface{} {
	data := reflect.New(reflect.TypeOf(tl.dataType)).Interface()
	err := tl.serializer.Deserialize(serialized, data)
	if err != nil {
		panic("cant deserialize data")
	}
	return data
}
