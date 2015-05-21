package enums

// Possible values for SWF Timeouts.
const (
	ActivityTaskTimeoutTypeHeartbeat       = "HEARTBEAT"
	ActivityTaskTimeoutTypeScheduleToClose = "SCHEDULE_TO_CLOSE"
	ActivityTaskTimeoutTypeScheduleToStart = "SCHEDULE_TO_START"
	ActivityTaskTimeoutTypeStartToClose    = "START_TO_CLOSE"
)

// Possible values for SWF CancelTimerFailed.
const (
	CancelTimerFailedCauseOperationNotPermitted = "OPERATION_NOT_PERMITTED"
	CancelTimerFailedCauseTimerIDUnknown        = "TIMER_ID_UNKNOWN"
)

// Possible values for SWF.
const (
	CancelWorkflowExecutionFailedCauseOperationNotPermitted = "OPERATION_NOT_PERMITTED"
	CancelWorkflowExecutionFailedCauseUnhandledDecision     = "UNHANDLED_DECISION"
)

// Possible values for SWF.
const (
	ChildPolicyAbandon       = "ABANDON"
	ChildPolicyRequestCancel = "REQUEST_CANCEL"
	ChildPolicyTerminate     = "TERMINATE"
)

// Possible values for SWF.
const (
	CloseStatusCanceled       = "CANCELED"
	CloseStatusCompleted      = "COMPLETED"
	CloseStatusContinuedAsNew = "CONTINUED_AS_NEW"
	CloseStatusFailed         = "FAILED"
	CloseStatusTerminated     = "TERMINATED"
	CloseStatusTimedOut       = "TIMED_OUT"
)

// Possible values for SWF.
const (
	CompleteWorkflowExecutionFailedCauseOperationNotPermitted = "OPERATION_NOT_PERMITTED"
	CompleteWorkflowExecutionFailedCauseUnhandledDecision     = "UNHANDLED_DECISION"
)

// Possible values for SWF.
const (
	ContinueAsNewWorkflowExecutionFailedCauseContinueAsNewWorkflowExecutionRateExceeded   = "CONTINUE_AS_NEW_WORKFLOW_EXECUTION_RATE_EXCEEDED"
	ContinueAsNewWorkflowExecutionFailedCauseDefaultChildPolicyUndefined                  = "DEFAULT_CHILD_POLICY_UNDEFINED"
	ContinueAsNewWorkflowExecutionFailedCauseDefaultExecutionStartToCloseTimeoutUndefined = "DEFAULT_EXECUTION_START_TO_CLOSE_TIMEOUT_UNDEFINED"
	ContinueAsNewWorkflowExecutionFailedCauseDefaultTaskListUndefined                     = "DEFAULT_TASK_LIST_UNDEFINED"
	ContinueAsNewWorkflowExecutionFailedCauseDefaultTaskStartToCloseTimeoutUndefined      = "DEFAULT_TASK_START_TO_CLOSE_TIMEOUT_UNDEFINED"
	ContinueAsNewWorkflowExecutionFailedCauseOperationNotPermitted                        = "OPERATION_NOT_PERMITTED"
	ContinueAsNewWorkflowExecutionFailedCauseUnhandledDecision                            = "UNHANDLED_DECISION"
	ContinueAsNewWorkflowExecutionFailedCauseWorkflowTypeDeprecated                       = "WORKFLOW_TYPE_DEPRECATED"
	ContinueAsNewWorkflowExecutionFailedCauseWorkflowTypeDoesNotExist                     = "WORKFLOW_TYPE_DOES_NOT_EXIST"
)

// Possible values for SWF.
const (
	DecisionTaskTimeoutTypeStartToClose = "START_TO_CLOSE"
)

// Possible values for SWF.
const (
	DecisionTypeCancelTimer                            = "CancelTimer"
	DecisionTypeCancelWorkflowExecution                = "CancelWorkflowExecution"
	DecisionTypeCompleteWorkflowExecution              = "CompleteWorkflowExecution"
	DecisionTypeContinueAsNewWorkflowExecution         = "ContinueAsNewWorkflowExecution"
	DecisionTypeFailWorkflowExecution                  = "FailWorkflowExecution"
	DecisionTypeRecordMarker                           = "RecordMarker"
	DecisionTypeRequestCancelActivityTask              = "RequestCancelActivityTask"
	DecisionTypeRequestCancelExternalWorkflowExecution = "RequestCancelExternalWorkflowExecution"
	DecisionTypeScheduleActivityTask                   = "ScheduleActivityTask"
	DecisionTypeSignalExternalWorkflowExecution        = "SignalExternalWorkflowExecution"
	DecisionTypeStartChildWorkflowExecution            = "StartChildWorkflowExecution"
	DecisionTypeStartTimer                             = "StartTimer"
)

// Possible values for SWF.
const (
	EventTypeActivityTaskCancelRequested                     = "ActivityTaskCancelRequested"
	EventTypeActivityTaskCanceled                            = "ActivityTaskCanceled"
	EventTypeActivityTaskCompleted                           = "ActivityTaskCompleted"
	EventTypeActivityTaskFailed                              = "ActivityTaskFailed"
	EventTypeActivityTaskScheduled                           = "ActivityTaskScheduled"
	EventTypeActivityTaskStarted                             = "ActivityTaskStarted"
	EventTypeActivityTaskTimedOut                            = "ActivityTaskTimedOut"
	EventTypeCancelTimerFailed                               = "CancelTimerFailed"
	EventTypeCancelWorkflowExecutionFailed                   = "CancelWorkflowExecutionFailed"
	EventTypeChildWorkflowExecutionCanceled                  = "ChildWorkflowExecutionCanceled"
	EventTypeChildWorkflowExecutionCompleted                 = "ChildWorkflowExecutionCompleted"
	EventTypeChildWorkflowExecutionFailed                    = "ChildWorkflowExecutionFailed"
	EventTypeChildWorkflowExecutionStarted                   = "ChildWorkflowExecutionStarted"
	EventTypeChildWorkflowExecutionTerminated                = "ChildWorkflowExecutionTerminated"
	EventTypeChildWorkflowExecutionTimedOut                  = "ChildWorkflowExecutionTimedOut"
	EventTypeCompleteWorkflowExecutionFailed                 = "CompleteWorkflowExecutionFailed"
	EventTypeContinueAsNewWorkflowExecutionFailed            = "ContinueAsNewWorkflowExecutionFailed"
	EventTypeDecisionTaskCompleted                           = "DecisionTaskCompleted"
	EventTypeDecisionTaskScheduled                           = "DecisionTaskScheduled"
	EventTypeDecisionTaskStarted                             = "DecisionTaskStarted"
	EventTypeDecisionTaskTimedOut                            = "DecisionTaskTimedOut"
	EventTypeExternalWorkflowExecutionCancelRequested        = "ExternalWorkflowExecutionCancelRequested"
	EventTypeExternalWorkflowExecutionSignaled               = "ExternalWorkflowExecutionSignaled"
	EventTypeFailWorkflowExecutionFailed                     = "FailWorkflowExecutionFailed"
	EventTypeMarkerRecorded                                  = "MarkerRecorded"
	EventTypeRecordMarkerFailed                              = "RecordMarkerFailed"
	EventTypeRequestCancelActivityTaskFailed                 = "RequestCancelActivityTaskFailed"
	EventTypeRequestCancelExternalWorkflowExecutionFailed    = "RequestCancelExternalWorkflowExecutionFailed"
	EventTypeRequestCancelExternalWorkflowExecutionInitiated = "RequestCancelExternalWorkflowExecutionInitiated"
	EventTypeScheduleActivityTaskFailed                      = "ScheduleActivityTaskFailed"
	EventTypeSignalExternalWorkflowExecutionFailed           = "SignalExternalWorkflowExecutionFailed"
	EventTypeSignalExternalWorkflowExecutionInitiated        = "SignalExternalWorkflowExecutionInitiated"
	EventTypeStartChildWorkflowExecutionFailed               = "StartChildWorkflowExecutionFailed"
	EventTypeStartChildWorkflowExecutionInitiated            = "StartChildWorkflowExecutionInitiated"
	EventTypeStartTimerFailed                                = "StartTimerFailed"
	EventTypeTimerCanceled                                   = "TimerCanceled"
	EventTypeTimerFired                                      = "TimerFired"
	EventTypeTimerStarted                                    = "TimerStarted"
	EventTypeWorkflowExecutionCancelRequested                = "WorkflowExecutionCancelRequested"
	EventTypeWorkflowExecutionCanceled                       = "WorkflowExecutionCanceled"
	EventTypeWorkflowExecutionCompleted                      = "WorkflowExecutionCompleted"
	EventTypeWorkflowExecutionContinuedAsNew                 = "WorkflowExecutionContinuedAsNew"
	EventTypeWorkflowExecutionFailed                         = "WorkflowExecutionFailed"
	EventTypeWorkflowExecutionSignaled                       = "WorkflowExecutionSignaled"
	EventTypeWorkflowExecutionStarted                        = "WorkflowExecutionStarted"
	EventTypeWorkflowExecutionTerminated                     = "WorkflowExecutionTerminated"
	EventTypeWorkflowExecutionTimedOut                       = "WorkflowExecutionTimedOut"
)

// Possible values for SWF.
const (
	ExecutionStatusClosed = "CLOSED"
	ExecutionStatusOpen   = "OPEN"
)

// Possible values for SWF.
const (
	FailWorkflowExecutionFailedCauseOperationNotPermitted = "OPERATION_NOT_PERMITTED"
	FailWorkflowExecutionFailedCauseUnhandledDecision     = "UNHANDLED_DECISION"
)

// Possible values for SWF.
const (
	RecordMarkerFailedCauseOperationNotPermitted = "OPERATION_NOT_PERMITTED"
)

// Possible values for SWF.
const (
	RegistrationStatusDeprecated = "DEPRECATED"
	RegistrationStatusRegistered = "REGISTERED"
)

// Possible values for SWF.
const (
	RequestCancelActivityTaskFailedCauseActivityIDUnknown     = "ACTIVITY_ID_UNKNOWN"
	RequestCancelActivityTaskFailedCauseOperationNotPermitted = "OPERATION_NOT_PERMITTED"
)

// Possible values for SWF.
const (
	RequestCancelExternalWorkflowExecutionFailedCauseOperationNotPermitted                              = "OPERATION_NOT_PERMITTED"
	RequestCancelExternalWorkflowExecutionFailedCauseRequestCancelExternalWorkflowExecutionRateExceeded = "REQUEST_CANCEL_EXTERNAL_WORKFLOW_EXECUTION_RATE_EXCEEDED"
	RequestCancelExternalWorkflowExecutionFailedCauseUnknownExternalWorkflowExecution                   = "UNKNOWN_EXTERNAL_WORKFLOW_EXECUTION"
)

// Possible values for SWF.
const (
	ScheduleActivityTaskFailedCauseActivityCreationRateExceeded           = "ACTIVITY_CREATION_RATE_EXCEEDED"
	ScheduleActivityTaskFailedCauseActivityIDAlreadyInUse                 = "ACTIVITY_ID_ALREADY_IN_USE"
	ScheduleActivityTaskFailedCauseActivityTypeDeprecated                 = "ACTIVITY_TYPE_DEPRECATED"
	ScheduleActivityTaskFailedCauseActivityTypeDoesNotExist               = "ACTIVITY_TYPE_DOES_NOT_EXIST"
	ScheduleActivityTaskFailedCauseDefaultHeartbeatTimeoutUndefined       = "DEFAULT_HEARTBEAT_TIMEOUT_UNDEFINED"
	ScheduleActivityTaskFailedCauseDefaultScheduleToCloseTimeoutUndefined = "DEFAULT_SCHEDULE_TO_CLOSE_TIMEOUT_UNDEFINED"
	ScheduleActivityTaskFailedCauseDefaultScheduleToStartTimeoutUndefined = "DEFAULT_SCHEDULE_TO_START_TIMEOUT_UNDEFINED"
	ScheduleActivityTaskFailedCauseDefaultStartToCloseTimeoutUndefined    = "DEFAULT_START_TO_CLOSE_TIMEOUT_UNDEFINED"
	ScheduleActivityTaskFailedCauseDefaultTaskListUndefined               = "DEFAULT_TASK_LIST_UNDEFINED"
	ScheduleActivityTaskFailedCauseOpenActivitiesLimitExceeded            = "OPEN_ACTIVITIES_LIMIT_EXCEEDED"
	ScheduleActivityTaskFailedCauseOperationNotPermitted                  = "OPERATION_NOT_PERMITTED"
)

// Possible values for SWF.
const (
	SignalExternalWorkflowExecutionFailedCauseOperationNotPermitted                       = "OPERATION_NOT_PERMITTED"
	SignalExternalWorkflowExecutionFailedCauseSignalExternalWorkflowExecutionRateExceeded = "SIGNAL_EXTERNAL_WORKFLOW_EXECUTION_RATE_EXCEEDED"
	SignalExternalWorkflowExecutionFailedCauseUnknownExternalWorkflowExecution            = "UNKNOWN_EXTERNAL_WORKFLOW_EXECUTION"
)

// Possible values for SWF.
const (
	StartChildWorkflowExecutionFailedCauseChildCreationRateExceeded                    = "CHILD_CREATION_RATE_EXCEEDED"
	StartChildWorkflowExecutionFailedCauseDefaultChildPolicyUndefined                  = "DEFAULT_CHILD_POLICY_UNDEFINED"
	StartChildWorkflowExecutionFailedCauseDefaultExecutionStartToCloseTimeoutUndefined = "DEFAULT_EXECUTION_START_TO_CLOSE_TIMEOUT_UNDEFINED"
	StartChildWorkflowExecutionFailedCauseDefaultTaskListUndefined                     = "DEFAULT_TASK_LIST_UNDEFINED"
	StartChildWorkflowExecutionFailedCauseDefaultTaskStartToCloseTimeoutUndefined      = "DEFAULT_TASK_START_TO_CLOSE_TIMEOUT_UNDEFINED"
	StartChildWorkflowExecutionFailedCauseOpenChildrenLimitExceeded                    = "OPEN_CHILDREN_LIMIT_EXCEEDED"
	StartChildWorkflowExecutionFailedCauseOpenWorkflowsLimitExceeded                   = "OPEN_WORKFLOWS_LIMIT_EXCEEDED"
	StartChildWorkflowExecutionFailedCauseOperationNotPermitted                        = "OPERATION_NOT_PERMITTED"
	StartChildWorkflowExecutionFailedCauseWorkflowAlreadyRunning                       = "WORKFLOW_ALREADY_RUNNING"
	StartChildWorkflowExecutionFailedCauseWorkflowTypeDeprecated                       = "WORKFLOW_TYPE_DEPRECATED"
	StartChildWorkflowExecutionFailedCauseWorkflowTypeDoesNotExist                     = "WORKFLOW_TYPE_DOES_NOT_EXIST"
)

// Possible values for SWF.
const (
	StartTimerFailedCauseOpenTimersLimitExceeded   = "OPEN_TIMERS_LIMIT_EXCEEDED"
	StartTimerFailedCauseOperationNotPermitted     = "OPERATION_NOT_PERMITTED"
	StartTimerFailedCauseTimerCreationRateExceeded = "TIMER_CREATION_RATE_EXCEEDED"
	StartTimerFailedCauseTimerIDAlreadyInUse       = "TIMER_ID_ALREADY_IN_USE"
)

// Possible values for SWF.
const (
	WorkflowExecutionCancelRequestedCauseChildPolicyApplied = "CHILD_POLICY_APPLIED"
)

// Possible values for SWF.
const (
	WorkflowExecutionTerminatedCauseChildPolicyApplied = "CHILD_POLICY_APPLIED"
	WorkflowExecutionTerminatedCauseEventLimitExceeded = "EVENT_LIMIT_EXCEEDED"
	WorkflowExecutionTerminatedCauseOperatorInitiated  = "OPERATOR_INITIATED"
)

// Possible values for SWF.
const (
	WorkflowExecutionTimeoutTypeStartToClose = "START_TO_CLOSE"
)
