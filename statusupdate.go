package supervisor

import (
	"time"

	"go.uber.org/zap/zapcore"
)

type StatusUpdate struct {
	ServiceID   int
	ServiceName string
	Time        time.Time
	Status      Status
	Err         error
}

func (su StatusUpdate) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt("serviceID", su.ServiceID)
	enc.AddString("serviceName", su.ServiceName)
	enc.AddTime("time", su.Time)
	enc.AddString("status", su.Status.String())
	if su.Err != nil {
		enc.AddString("err", su.Err.Error())
	}
	return nil
}
