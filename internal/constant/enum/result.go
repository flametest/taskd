package enum

// Result is the outcome of a single task execution, recorded in task_record. It is
// intentionally distinct from Status (the task lifecycle state machine): an audit
// row's outcome (success/failure of one attempt) must never blur with the task's
// scheduled/claimed/dead state.
type Result string

const (
	// ExecutionSuccess means the executor returned nil for this attempt.
	ExecutionSuccess Result = "success"
	// ExecutionFailure means the executor returned an error (retryable or not)
	// for this attempt.
	ExecutionFailure Result = "failure"
)

func (r Result) String() string {
	return string(r)
}
