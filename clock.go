package supervisor

import (
	"time"
)

// Clock represents a pluggable clock service.
type Clock interface {
	// Now returns the current local time.
	Now() time.Time

	// NewTicker returns a new Ticker
	NewTicker(time.Duration) Ticker
}

// Ticker wraps the time.Ticker struct.
type Ticker interface {
	// C returns the channel on which the ticks are delivered.
	C() <-chan time.Time

	// Stop the Ticker.
	Stop()
}

// NewSystem returns a system clock implementing supervisor.Clock.
func NewSystemClock() *SystemClock {
	return &SystemClock{}
}

// System implements supervisor.Clock by wrapping functions in
// standard library time package.
type SystemClock struct{}

// Now wraps time.Now.
func (s *SystemClock) Now() time.Time {
	return time.Now()
}

// NewTicker returns a new time.Ticker.
func (s *SystemClock) NewTicker(duration time.Duration) Ticker {
	return &SystemTicker{time.NewTicker(duration)}
}

type SystemTicker struct {
	*time.Ticker
}

// C returns the channel on which the ticks are delivered.
func (t *SystemTicker) C() <-chan time.Time {
	return t.Ticker.C
}

// Stop stops the ticker.
func (t *SystemTicker) Stop() {
	t.Ticker.Stop()
}
