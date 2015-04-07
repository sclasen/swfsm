package test

import (
	"testing"

	"github.com/awslabs/aws-sdk-go/gen/swf"
	"github.com/sclasen/swfsm/fsm"
	"github.com/sclasen/swfsm/ops"
)

//These tests are passing if they compile basically.

func TestSWFIsAnSWFOperations(t *testing.T) {
	awsSwf := new(swf.SWF)
	AFuncForSWFOperations(awsSwf)
}

func TestSWFOperationsIsAnFSMSWFOps(t *testing.T) {
	awsSwf := new(swf.SWF)
	AFuncForFSMSWFOps(awsSwf)
}

func TestMockSWFIsAnSWFOperations(t *testing.T) {
	mock := new(MockSWF)
	AFuncForSWFOperations(mock)
}

func TestMockSWFIsAnFSMSWFOps(t *testing.T) {
	mock := new(MockSWF)
	AFuncForFSMSWFOps(mock)
}

func AFuncForSWFOperations(ops ops.SWFOperations) {

}

func AFuncForFSMSWFOps(ops fsm.SWFOps) {

}
