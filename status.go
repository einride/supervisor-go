package supervisor

// Status models the current status of a service.
type Status uint8

//go:generate gobin -m -run golang.org/x/tools/cmd/stringer -type Status -trimprefix Status

const (
	// StatusIdle is when a service is waiting to be scheduled by the OS scheduler.
	StatusIdle Status = iota
	// StatusRunning is when a service is running and everything is a-OK.
	StatusRunning
	// StatusStopped is when a service has stopped without without an error.
	StatusStopped
	// StatusError is when a service has stopped with an error.
	StatusError
	// StatusPanic is when a service has stopped with a runtime panic.
	StatusPanic
)

// IsAlive returns true for statuses indicating that the service is currently alive.
func (s Status) IsAlive() bool {
	return s < StatusStopped
}
