package enum

type Status string

const (
	TaskStatusScheduled = "scheduled"
	TaskStatusClaimed   = "claimed"
	TaskStatusRunning   = "running"
	TaskStatusSucceeded = "succeeded"
	TaskStatusFailed    = "failed"
	TaskStatusDead      = "dead"
)

func (s Status) String() string {
	return string(s)
}
