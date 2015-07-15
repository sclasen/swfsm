package fsm

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/aws/aws-sdk-go/service/swf"
	"github.com/juju/errors"
)

//ReplicationHandler can be configured on an FSM and will be called when a DecisionTask is successfully completed.
type ReplicationHandler func(*FSMContext, *swf.PollForDecisionTaskOutput, *swf.RespondDecisionTaskCompletedInput, *SerializedState) error

//KinesisOps is the subset of kinesis.Kinesis ops required by KinesisReplication
type KinesisOps interface {
	PutRecord(*kinesis.PutRecordInput) (*kinesis.PutRecordOutput, error)
}

func defaultKinesisReplicator() KinesisReplicator {
	return func(fsm, workflowID string, put func() (*kinesis.PutRecordOutput, error)) (*kinesis.PutRecordOutput, error) {
		return put()
	}
}

//KinesisReplicator lets you customize the retry logic around Replicating State to Kinesis.
type KinesisReplicator func(fsm, workflowID string, put func() (*kinesis.PutRecordOutput, error)) (*kinesis.PutRecordOutput, error)

//KinesisReplication can be used as a ReplicationHandler by setting its Handler func as the FSM ReplicationHandler
type KinesisReplication struct {
	KinesisStream     string
	KinesisReplicator KinesisReplicator
	KinesisOps        KinesisOps
}

//Handler is a ReplicationHandler. to configure it on your FSM, do fsm.ReplicationHandler = &KinesisReplication{...).Handler
func (f *KinesisReplication) Handler(ctx *FSMContext, decisionTask *swf.PollForDecisionTaskOutput, completedDecision *swf.RespondDecisionTaskCompletedInput, state *SerializedState) error {
	if state == nil || f.KinesisStream == "" {
		return nil
	}
	stateToReplicate, err := ctx.Serializer().Serialize(state)
	if err != nil {
		log.Printf("component=kinesis-replication at=serialize-state-failed error=%q", err.Error())
		return errors.Trace(err)
	}

	put := func() (*kinesis.PutRecordOutput, error) {
		return f.KinesisOps.PutRecord(&kinesis.PutRecordInput{
			StreamName: aws.String(f.KinesisStream),
			//partition by workflow
			PartitionKey: decisionTask.WorkflowExecution.WorkflowID,
			Data:         []byte(stateToReplicate),
		})
	}

	resp, err := f.KinesisReplicator(*ctx.WorkflowType.Name, *decisionTask.WorkflowExecution.WorkflowID, put)

	if err != nil {
		log.Printf("component=kinesis-replication  at=replicate-state-failed error=%q", err.Error())
	} else {
		log.Printf("component=kinesis-replication at=replicated-state shard=%s sequence=%s", *resp.ShardID, *resp.SequenceNumber)
	}
	return errors.Trace(err)
}
