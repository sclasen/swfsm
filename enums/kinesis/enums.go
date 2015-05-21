package kinesis

// Possible values for Kinesis.
const (
	ShardIteratorTypeAfterSequenceNumber = "AFTER_SEQUENCE_NUMBER"
	ShardIteratorTypeAtSequenceNumber    = "AT_SEQUENCE_NUMBER"
	ShardIteratorTypeLatest              = "LATEST"
	ShardIteratorTypeTrimHorizon         = "TRIM_HORIZON"
)

// Possible values for Kinesis.
const (
	StreamStatusActive   = "ACTIVE"
	StreamStatusCreating = "CREATING"
	StreamStatusDeleting = "DELETING"
	StreamStatusUpdating = "UPDATING"
)
