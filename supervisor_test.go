package supervisor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/einride/clock-go/pkg/mockclock"
	"github.com/einride/supervisor-go/internal/gomockextra"
	"github.com/golang/mock/gomock"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

const mockNowNanos = 1234

func TestSupervisor_SingleService(t *testing.T) {
	cfg := &Config{}
	cfg.Services = append(cfg.Services, NewService("service1", func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}))
	_, done := newTestFixture(t, cfg)
	defer done()
}

func TestSupervisor_IgnoreNilService(t *testing.T) {
	cfg := &Config{}
	cfg.Services = append(cfg.Services, nil)
	cfg.Services = append(cfg.Services, NewService("service1", func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}))
	_, done := newTestFixture(t, cfg)
	defer done()
}

func TestSupervisor_RestartOnError(t *testing.T) {
	cfg := &Config{}
	rendezvousChan := make(chan struct{})
	cfg.Services = append(cfg.Services, NewService("service1", func(ctx context.Context) error {
		rendezvousChan <- struct{}{}
		return xerrors.New("boom")
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
	cfg := &Config{}
	rendezvousChan := make(chan struct{})
	cfg.Services = append(cfg.Services, NewService("service1", func(ctx context.Context) error {
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
	cfg := &Config{}
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

type testFixture struct {
	clock           *mockclock.MockClock
	restartTicker   *mockclock.MockTicker
	restartTickChan chan time.Time
}

func newTestFixture(t *testing.T, cfg *Config) (*testFixture, func()) {
	t.Helper()
	mockCtrl := gomock.NewController(gomockextra.GoroutineReporter(t))
	f := &testFixture{
		clock:           mockclock.NewMockClock(mockCtrl),
		restartTicker:   mockclock.NewMockTicker(mockCtrl),
		restartTickChan: make(chan time.Time),
	}
	cfg.RestartInterval = time.Second
	f.clock.EXPECT().NewTicker(cfg.RestartInterval).Return(f.restartTicker)
	f.clock.EXPECT().Now().Return(time.Unix(0, mockNowNanos)).AnyTimes()
	f.restartTicker.EXPECT().C().Return(f.restartTickChan)
	f.restartTicker.EXPECT().Stop()
	cfg.Logger = zap.NewExample()
	cfg.Clock = f.clock
	s := New(cfg)
	var g errgroup.Group
	ctx, cancel := context.WithCancel(context.Background())
	g.Go(func() error {
		return s.Start(ctx)
	})
	done := func() {
		cancel()
		assert.NilError(t, g.Wait())
		mockCtrl.Finish()
	}
	return f, done
}
