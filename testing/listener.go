package testing

import (
	"sync"
	"time"

	"fmt"

	"reflect"

	"code.google.com/p/go-uuid/uuid"
	"github.com/awslabs/aws-sdk-go/service/swf"
	"github.com/sclasen/swfsm/activity"
	"github.com/sclasen/swfsm/fsm"
)

type TestAdapter interface {
	Logf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	FailNow()
}

type TestConfig struct {
	Testing                    TestAdapter
	FSM                        *fsm.FSM
	StubFSM                    *fsm.FSM
	Workers                    []*activity.ActivityWorker
	StubbedWorkflows           []string
	ShortStubbedWorkflows      []string
	DefaultWaitTimeout         int
	DefaultActivityInterceptor activity.ActivityInterceptor
}

func NewTestListener(t TestConfig) *TestListener {
	if t.DefaultWaitTimeout == 0 {
		t.DefaultWaitTimeout = 10
	}

	if t.DefaultActivityInterceptor == nil {
		t.DefaultActivityInterceptor = TestFailOnceActivityInterceptor()
	}

	tl := &TestListener{
		decisionOutcomes: make(chan DecisionOutcome, 1000),
		historyInterest:  make(map[string]chan *swf.HistoryEvent, 1000),
		decisionInterest: make(map[string]chan *swf.Decision, 1000),
		stateInterest:    make(map[string]chan string, 1000),
		dataInterest:     make(map[string]chan *StateData, 1000),
		DefaultWait:      time.Duration(t.DefaultWaitTimeout) * time.Second,
		testAdapter:      t.Testing,
		TestID:           uuid.New(),
		dataType:         t.FSM.DataType,
		serializer:       t.FSM.Serializer,
	}

	t.FSM.ReplicationHandler = TestReplicator(tl.decisionOutcomes)
	t.FSM.DecisionInterceptor = TestDecisionInterceptor(tl.TestID, t.StubbedWorkflows, t.ShortStubbedWorkflows)
	t.FSM.TaskList = tl.TestID

	if t.StubFSM != nil {
		t.StubFSM.ReplicationHandler = TestReplicator(tl.decisionOutcomes)
	}

	for _, w := range t.Workers {
		if w.ActivityInterceptor == nil {
			w.ActivityInterceptor = t.DefaultActivityInterceptor
		}
		w.TaskList = w.TaskList + tl.TestID
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
	TestID           string
	dataType         interface{}
	serializer       fsm.StateSerializer
}

func (tl *TestListener) RegisterHistoryInterest(workflowID string) chan *swf.HistoryEvent {
	defer tl.historyLock.Unlock()
	tl.historyLock.Lock()
	historyChan, ok := tl.historyInterest[workflowID]
	if !ok {
		historyChan = make(chan *swf.HistoryEvent, 1000)
		tl.historyInterest[workflowID] = historyChan
	}
	return historyChan
}

func (tl *TestListener) RegisterDecisionInterest(workflowID string) chan *swf.Decision {
	defer tl.decisionLock.Unlock()
	tl.decisionLock.Lock()
	decisionChan, ok := tl.decisionInterest[workflowID]
	if !ok {
		decisionChan = make(chan *swf.Decision, 1000)
		tl.decisionInterest[workflowID] = decisionChan
	}
	return decisionChan
}

func (tl *TestListener) RegisterStateInterest(workflowID string) chan string {
	defer tl.stateLock.Unlock()
	tl.stateLock.Lock()
	stateChan, ok := tl.stateInterest[workflowID]
	if !ok {
		stateChan = make(chan string, 1000)
		tl.stateInterest[workflowID] = stateChan
	}
	return stateChan
}

func (tl *TestListener) RegisterDataInterest(workflowID string) chan *StateData {
	defer tl.dataLock.Unlock()
	tl.dataLock.Lock()
	dataChan, ok := tl.dataInterest[workflowID]
	if !ok {
		dataChan = make(chan *StateData, 1000)
		tl.dataInterest[workflowID] = dataChan
	}
	return dataChan
}

func (tl *TestListener) AwaitStateFor(workflowID, state string, waitFor time.Duration) {
	ch := tl.RegisterStateInterest(workflowID)
	timer := time.After(waitFor)

	for {
		select {
		case s := <-ch:
			tl.testAdapter.Logf("TestListener: await state for workflow=%s state=%s received-state=%s", workflowID, state, s)
			if s == state {
				return
			}
		case <-timer:
			panic(fmt.Sprintf("TestListener: timed out waiting for workflow=%s state=%s", workflowID, state))
			tl.testAdapter.Fatalf("TestListener: timed out waiting for workflow=%s state=%s", workflowID, state)
			tl.testAdapter.FailNow()
		}
	}
}

func (tl *TestListener) AwaitState(workflowID, state string) {
	tl.AwaitStateFor(workflowID, state, tl.DefaultWait)
}

func (tl *TestListener) AwaitEventFor(workflowID string, waitFor time.Duration, predicate func(*swf.HistoryEvent) bool) {
	ch := tl.RegisterHistoryInterest(workflowID)
	timer := time.After(waitFor)
	for {
		select {
		case h := <-ch:
			if predicate(h) {
				tl.testAdapter.Logf("TestListener: await event for workflow=%s received-event=%s predicate=true", workflowID, *h.EventType)
				return
			} else {
				tl.testAdapter.Logf("TestListener: await event for workflow=%s received-event=%s predicate=false", workflowID, *h.EventType)
			}
		case <-timer:
			panic(fmt.Sprintf("TestListener: timed out waiting for workflow=%s event", workflowID))
			tl.testAdapter.Fatalf("TestListener: timed out waiting for workflow=%s event", workflowID)
			tl.testAdapter.FailNow()
		}
	}
}

func (tl *TestListener) AwaitEvent(workflowID string, predicate func(*swf.HistoryEvent) bool) {
	tl.AwaitEventFor(workflowID, tl.DefaultWait, predicate)
}

func (tl *TestListener) AwaitDecisionFor(workflowID string, waitFor time.Duration, predicate func(*swf.Decision) bool) {
	ch := tl.RegisterDecisionInterest(workflowID)
	timer := time.After(waitFor)
	for {
		select {
		case h := <-ch:
			if predicate(h) {
				tl.testAdapter.Logf("TestListener: await decision for workflow=%s received-decision=%s predicate=true", workflowID, *h.DecisionType)
				return
			} else {
				tl.testAdapter.Logf("TestListener: await decision for workflow=%s received-decision=%s predicate=false", workflowID, *h.DecisionType)
			}
		case <-timer:
			panic(fmt.Sprintf("TestListener: timed out waiting for workflow=%s decision", workflowID))
			tl.testAdapter.Fatalf("TestListener: timed out waiting for workflow=%s decision", workflowID)
			tl.testAdapter.FailNow()
		}
	}
}

func (tl *TestListener) AwaitDecision(workflowID string, predicate func(*swf.Decision) bool) {
	tl.AwaitDecisionFor(workflowID, tl.DefaultWait, predicate)
}

func (tl *TestListener) AwaitDataFor(workflowID string, waitFor time.Duration, predicate func(*StateData) bool) {
	ch := tl.RegisterDataInterest(workflowID)
	timer := time.After(waitFor)
	for {
		select {
		case d := <-ch:
			if predicate(d) {
				tl.testAdapter.Logf("TestListener: await data for workflow=%s received-data=%+v predicate=true", workflowID, d)
				return
			} else {
				tl.testAdapter.Logf("TestListener: await data for workflow=%s received-data=%+v predicate=false", workflowID, d)
			}
		case <-timer:
			panic(fmt.Sprintf("TestListener: timed out waiting for workflow=%s data", workflowID))
			tl.testAdapter.Fatalf("TestListener: timed out waiting for workflow=%s data", workflowID)
			tl.testAdapter.FailNow()
		}
	}
}

func (tl *TestListener) AwaitData(workflowID string, predicate func(*StateData) bool) {
	tl.AwaitDataFor(workflowID, tl.DefaultWait, predicate)
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

			workflow := *do.DecisionTask.WorkflowExecution.WorkflowID
			tl.testAdapter.Logf("TestListener: DecisionOutcome for workflow %s", workflow)
			//send history events
			if c, ok := tl.historyInterest[workflow]; ok {
				tl.testAdapter.Logf("TestListener: yes historyInterest for workflow %s", workflow)
				for i := len(do.DecisionTask.Events) - 1; i >= 0; i-- {
					event := do.DecisionTask.Events[i]
					if *event.EventID > *do.DecisionTask.PreviousStartedEventID {
						tl.testAdapter.Logf("TestListener: yes historyInterest for workflow %s %s", workflow, *do.DecisionTask.Events[i].EventType)
						c <- do.DecisionTask.Events[i]
					}
				}
			} else {
				tl.testAdapter.Logf("TestListener: no historyInterest for workflow %s", workflow)
			}
			//send decisions
			if c, ok := tl.decisionInterest[workflow]; ok {
				for _, d := range do.Decisions {
					tl.testAdapter.Logf("TestListener: yes decisionInterest for workflow %s %s", workflow, *d.DecisionType)
					c <- d
				}
			} else {
				tl.testAdapter.Logf("TestListener: no decisionInterest for workflow %s", workflow)
			}
			//send states
			if c, ok := tl.stateInterest[workflow]; ok {
				tl.testAdapter.Logf("TestListener: yes stateInterest for workflow %s %s", workflow, do.State)
				c <- do.State.StateName
			} else {
				tl.testAdapter.Logf("TestListener: no stateInterest for workflow %s", workflow)
			}
			//send data
			if c, ok := tl.dataInterest[workflow]; ok {
				tl.testAdapter.Logf("TestListener: yes stateInterest for workflow %s %s", workflow, do.State)
				stateData := &StateData{
					State: do.State.StateName,
					Data:  tl.deserialize(do.State.StateData),
				}
				c <- stateData
			} else {
				tl.testAdapter.Logf("TestListener: no stateInterest for workflow %s", workflow)
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
