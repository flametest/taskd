package enum

type Status string

const (
	TaskStatusScheduled = "scheduled"
	TaskStatusClaimed   = "claimed"
	TaskStatusRunning   = "running"
	TaskStatusSucceeded = "succeeded"
	TaskStatusFailed    = "failed"
	TaskStatusDead      = "dead"
	TaskStatusCanceled  = "canceled"
)

func (s Status) String() string {
	return string(s)
}
