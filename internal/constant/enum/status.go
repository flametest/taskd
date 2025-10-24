package enum

type Status string

const (
	TaskStatusScheduled = "scheduled"
	TaskStatusRunning   = "running"
	TaskStatusSucceeded = "succeeded"
	TaskStatusFailed    = "failed"
)

func (s Status) String() string {
	return string(s)
}
