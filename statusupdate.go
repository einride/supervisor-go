package supervisor

import (
	"time"
)

// StatusUpdate represents an update to a supervised service.
type StatusUpdate struct {
	ServiceID   int       `json:"serviceId"`
	ServiceName string    `json:"serviceName"`
	Time        time.Time `json:"time"`
	Status      Status    `json:"status"`
	Err         error     `json:"-"`
}
