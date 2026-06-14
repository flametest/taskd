package timingwheel

import "github.com/flametest/vita/verrors"

var (
	// ErrKeyExists is returned by Add when key is already present (conflict).
	ErrKeyExists = verrors.ConflictError("timingwheel: key already exists")

	// ErrStopped is returned by Add when the wheel has been stopped and no
	// longer accepts timers (internal state unavailable).
	ErrStopped = verrors.InternalServerError("timingwheel: wheel already stopped")
)
