package fsm

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/swf"
	. "github.com/sclasen/swfsm/sugar"
)

type MockClient struct {
	*kinesis.Kinesis
	*swf.SWF
	putRecords []kinesis.PutRecordInput
	seqNumber  int
}

func (c *MockClient) PutRecord(req *kinesis.PutRecordInput) (*kinesis.PutRecordOutput, error) {
	if c.putRecords == nil {
		c.seqNumber = rand.Int()
		c.putRecords = make([]kinesis.PutRecordInput, 0)
	}
	c.putRecords = append(c.putRecords, *req)
	c.seqNumber++
	return &kinesis.PutRecordOutput{
		SequenceNumber: S(strconv.Itoa(c.seqNumber)),
		ShardId:        req.PartitionKey,
	}, nil
}

func (c *MockClient) RespondDecisionTaskCompleted(req *swf.RespondDecisionTaskCompletedInput) (*swf.RespondDecisionTaskCompletedOutput, error) {
	return nil, nil
}

func TestKinesisReplication(t *testing.T) {
	client := &MockClient{}
	rep := KinesisReplication{
		KinesisStream:     "test-stream",
		KinesisOps:        client,
		KinesisReplicator: defaultKinesisReplicator(),
	}
	fsm := testFSM()
	fsm.SWF = client
	fsm.ReplicationHandler = rep.Handler
	fsm.AddInitialState(&FSMState{
		Name: "initial",
		Decider: func(f *FSMContext, h *swf.HistoryEvent, d interface{}) Outcome {
			if *h.EventType == swf.EventTypeWorkflowExecutionStarted {
				return f.Goto("done", d, f.EmptyDecisions())
			}
			t.Fatal("unexpected")
			return f.Pass()
		},
	})
	fsm.AddState(&FSMState{
		Name: "done",
		Decider: func(f *FSMContext, h *swf.HistoryEvent, d interface{}) Outcome {
			go fsm.ShutdownManager.StopPollers()
			return f.Stay(d, f.EmptyDecisions())
		},
	})
	events := []*swf.HistoryEvent{
		&swf.HistoryEvent{EventType: S("DecisionTaskStarted"), EventId: I(3)},
		&swf.HistoryEvent{EventType: S("DecisionTaskScheduled"), EventId: I(2)},
		&swf.HistoryEvent{
			EventId:   I(1),
			EventType: S("WorkflowExecutionStarted"),
			WorkflowExecutionStartedEventAttributes: &swf.WorkflowExecutionStartedEventAttributes{
				Input: StartFSMWorkflowInput(fsm, new(TestData)),
			},
		},
	}
	decisionTask := testDecisionTask(0, events)

	fsm.handleDecisionTask(decisionTask)

	if client.putRecords == nil || len(client.putRecords) != 1 {
		t.Fatalf("expected only one state to be replicated, got: %v", client.putRecords)
	}
	replication := client.putRecords[0]
	if *replication.StreamName != rep.KinesisStream {
		t.Fatalf("expected Kinesis stream: %q, got %q", rep.KinesisStream, *replication.StreamName)
	}
	var replicatedState SerializedState
	if err := fsm.Serializer.Deserialize(string(replication.Data), &replicatedState); err != nil {
		t.Fatal(err)
	}
	if replicatedState.StateVersion != 1 {
		t.Fatalf("state.StateVersion != 1, got: %d", replicatedState.StateVersion)
	}
	if replicatedState.StateName != "done" {
		t.Fatalf("current state being replicated is not 'done', got %q", replicatedState.StateName)
	}
}
