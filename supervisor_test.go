package supervisor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	testr "github.com/go-logr/logr/testing"
	"go.einride.tech/clock"
	"go.einride.tech/clock/systemclock"
	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestSupervisor_New(t *testing.T) {
	// given
	var bs bytes.Buffer
	cfg := Config{
		RestartInterval: 100 * time.Millisecond,
		Clock:           systemclock.New(),
		Logger:          testr.NewTestLogger(t),
	}
	supervisor := New(&cfg)
	// when the supervisor is started
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	assert.NilError(t, supervisor.Run(ctx))
	cancel()
	// then nothing is logged since no services are added
	assert.Equal(t, "", bs.String())
}

func TestSupervisor_SingleService(t *testing.T) {
	cfg := &Config{
		Logger: testr.NewTestLogger(t),
	}
	cfg.Services = append(cfg.Services, NewService("service1", func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}))
	_, done := newTestFixture(t, cfg)
	defer done()
}

func TestSupervisor_Logger(t *testing.T) {
	cfg := &Config{
		Logger: testr.NewTestLogger(t),
	}
	cfg.Services = append(cfg.Services, NewService("service1", func(ctx context.Context) error {
		logger, err := logr.FromContext(ctx)
		assert.NilError(t, err)
		logger.Info("running")
		defer logger.Info("returning")
		<-ctx.Done()
		return nil
	}))
	_, done := newTestFixture(t, cfg)
	defer done()
}

func TestSupervisor_IgnoreNilService(t *testing.T) {
	cfg := &Config{
		Logger: testr.NewTestLogger(t),
	}
	cfg.Services = append(cfg.Services, nil)
	cfg.Services = append(cfg.Services, NewService("service1", func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}))
	_, done := newTestFixture(t, cfg)
	defer done()
}

func TestSupervisor_RestartOnError(t *testing.T) {
	cfg := &Config{
		Logger: testr.NewTestLogger(t),
	}
	rendezvousChan := make(chan struct{})
	cfg.Services = append(cfg.Services, NewService("service1", func(_ context.Context) error {
		rendezvousChan <- struct{}{}
		return errors.New("boom")
	}))
	statusUpdateChan := make(chan StatusUpdate, 6)
	cfg.StatusUpdateListeners = append(cfg.StatusUpdateListeners, func(statusUpdates []StatusUpdate) {
		assert.Assert(t, is.Len(statusUpdates, 1))
		assert.Equal(t, "service1", statusUpdates[0].ServiceName)
		statusUpdateChan <- statusUpdates[0]
	})
	f, done := newTestFixture(t, cfg)
	assert.Equal(t, StatusIdle, (<-statusUpdateChan).Status)
	assert.Equal(t, StatusRunning, (<-statusUpdateChan).Status)
	select {
	case <-rendezvousChan:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first run")
	}
	assert.Equal(t, StatusError, (<-statusUpdateChan).Status)
	f.restartTickChan <- time.Unix(0, 0)
	assert.Equal(t, StatusIdle, (<-statusUpdateChan).Status)
	assert.Equal(t, StatusRunning, (<-statusUpdateChan).Status)
	select {
	case <-rendezvousChan:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second run")
	}
	done()
	assert.Equal(t, StatusError, (<-statusUpdateChan).Status)
}

func TestSupervisor_RestartOnPanic(t *testing.T) {
	cfg := &Config{
		Logger: testr.NewTestLogger(t),
	}
	rendezvousChan := make(chan struct{})
	cfg.Services = append(cfg.Services, NewService("service1", func(_ context.Context) error {
		rendezvousChan <- struct{}{}
		panic("boom")
	}))
	statusUpdateChan := make(chan StatusUpdate, 6)
	cfg.StatusUpdateListeners = append(cfg.StatusUpdateListeners, func(statusUpdates []StatusUpdate) {
		assert.Assert(t, is.Len(statusUpdates, 1))
		assert.Equal(t, "service1", statusUpdates[0].ServiceName)
		statusUpdateChan <- statusUpdates[0]
	})
	f, done := newTestFixture(t, cfg)
	assert.Equal(t, StatusIdle, (<-statusUpdateChan).Status)
	assert.Equal(t, StatusRunning, (<-statusUpdateChan).Status)
	select {
	case <-rendezvousChan:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first run")
	}
	assert.Equal(t, StatusPanic, (<-statusUpdateChan).Status)
	f.restartTickChan <- time.Unix(0, 0)
	assert.Equal(t, StatusIdle, (<-statusUpdateChan).Status)
	assert.Equal(t, StatusRunning, (<-statusUpdateChan).Status)
	select {
	case <-rendezvousChan:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second run")
	}
	done()
	assert.Equal(t, StatusPanic, (<-statusUpdateChan).Status)
}

func TestSupervisor_MultipleServices(t *testing.T) {
	cfg := &Config{
		Logger: testr.NewTestLogger(t),
	}
	const numServices = 10
	serviceChan := make(chan struct{})
	for i := 0; i < numServices; i++ {
		cfg.Services = append(cfg.Services, NewService(fmt.Sprintf("service%d", i), func(ctx context.Context) error {
			serviceChan <- struct{}{}
			<-ctx.Done()
			return nil
		}))
	}
	_, done := newTestFixture(t, cfg)
	defer done()
	for i := 0; i < numServices; i++ {
		select {
		case <-serviceChan:
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for service %d to run", i)
		}
	}
}

type mockClock struct {
	clock.Clock
	now      time.Time
	tickChan chan time.Time
}

var _ clock.Clock = &mockClock{}

func (m *mockClock) Now() time.Time {
	return m.now
}

func (m *mockClock) NewTicker(time.Duration) clock.Ticker {
	return &mockTicker{timeChan: m.tickChan}
}

type mockTicker struct {
	timeChan chan time.Time
}

var _ clock.Ticker = &mockTicker{}

func (m *mockTicker) C() <-chan time.Time {
	return m.timeChan
}

func (m *mockTicker) Stop() {}

func (m *mockTicker) Reset(_ time.Duration) {}

type testFixture struct {
	clock           *mockClock
	restartTickChan chan time.Time
}

func newTestFixture(t *testing.T, cfg *Config) (*testFixture, func()) {
	t.Helper()
	restartTickChan := make(chan time.Time)
	f := &testFixture{
		restartTickChan: restartTickChan,
		clock:           &mockClock{tickChan: restartTickChan},
	}
	cfg.RestartInterval = time.Second
	cfg.Clock = f.clock
	s := New(cfg)
	var g errgroup.Group
	ctx, cancel := context.WithCancel(context.Background())
	g.Go(func() error {
		return s.Run(ctx)
	})
	done := func() {
		cancel()
		assert.NilError(t, g.Wait())
	}
	return f, done
}
