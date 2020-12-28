package supervisor

import (
	"fmt"
	"time"
)

// StatusUpdate represents an update to a supervised service.
type StatusUpdate struct {
	ServiceID   int
	ServiceName string
	Time        time.Time
	Status      Status
	Err         error
}

func (u StatusUpdate) String() string {
	return fmt.Sprintf(
		"{ServiceID: %v, ServiceName: %v, Time: %v, Status: %v, Err: %v}",
		u.ServiceID,
		u.ServiceName,
		u.Time,
		u.Status,
		u.Err,
	)
}
