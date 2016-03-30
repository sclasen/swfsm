package fsm

import (
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/sclasen/swfsm/sugar"
)

type testData struct {
	someField string
}

func TestCancelWorkflowExpectsCanceledStateAndCancelWorkflowDecision(t *testing.T) {
	// arrange
	fsmContext := &FSMContext{}
	testData := &testData{"Some data"}
	details := "Some details about the cancellation"

	// act
	outcome := fsmContext.CancelWorkflow(testData, S(details))

	// assert
	assert.Equal(t, CanceledState, outcome.State, "Expected outcome state to be "+CanceledState)
	assert.Equal(t, testData, outcome.Data, "Expected data in the outcome to match what was passed in")
	assert.Equal(t, 1, len(outcome.Decisions), "Expected one decision in the outcome")
	cancelDecision := FindDecision(outcome.Decisions, cancelWorkflowPredicate)
	assert.NotNil(t, cancelDecision, "Expected to find a cancellation decision in the outcome")
	assert.Equal(t, details, *cancelDecision.CancelWorkflowExecutionDecisionAttributes.Details,
		"Expected details in the cancel decision to match what was passed in")
}

func TestFailWorkflowExpectsFailedStateAndFailWorkflowDecision(t *testing.T) {
	// arrange
	fsmContext := &FSMContext{}
	testData := &testData{"Some data"}
	details := "Some details about the failure"

	// act
	outcome := fsmContext.FailWorkflow(testData, S(details))

	// assert
	assert.Equal(t, FailedState, outcome.State, "Expected outcome state to be "+FailedState)
	assert.Equal(t, testData, outcome.Data, "Expected data in the outcome to match what was passed in")
	assert.Equal(t, 1, len(outcome.Decisions), "Expected one decision in the outcome")
	failDecision := FindDecision(outcome.Decisions, failWorkflowPredicate)
	assert.NotNil(t, failDecision, "Expected to find a fail workflow decision in the outcome")
	assert.Equal(t, details, *failDecision.FailWorkflowExecutionDecisionAttributes.Details,
		"Expected details in the fail decision to match what was passed in")
}
