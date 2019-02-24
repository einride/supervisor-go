// Package supervisor provides a supervisor that monitors, manages the lifetime and reports status on services.
package supervisor

import (
	"context"
	"fmt"
	"path"
	"reflect"
	"time"

	"github.com/einride/clock-go/pkg/clock"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Config contains the full set of dependencies for a supervisor.
type Config struct {
	Services              []Service
	StatusUpdateListeners []func([]StatusUpdate)
	RestartInternal       time.Duration
	Clock                 clock.Clock
	Logger                *zap.Logger
}

type supervisedService struct {
	service Service
	id      int
	name    string
}

type Supervisor struct {
	cfg              *Config
	statusUpdateChan chan StatusUpdate
	// immutable, initialized by constructor
	supervisedServices []*supervisedService
	// mutable, only accessible by the supervisor thread
	latestStatusUpdates []StatusUpdate
}

// New creates a new supervisor from a config.
func New(cfg *Config) *Supervisor {
	s := &Supervisor{
		cfg:                 cfg,
		statusUpdateChan:    make(chan StatusUpdate),
		supervisedServices:  make([]*supervisedService, len(cfg.Services)),
		latestStatusUpdates: make([]StatusUpdate, len(cfg.Services)),
	}
	for id, service := range cfg.Services {
		s.supervisedServices[id] = &supervisedService{
			service: service,
			id:      id,
			name:    serviceName(service),
		}
	}
	return s
}

// Start the supervisor and all its services.
func (s *Supervisor) Start(ctx context.Context) error {
	// start all services
	for _, ss := range s.supervisedServices {
		s.start(ctx, ss)
	}
	s.notifyListeners()
	// monitor running services
	restartTicker := s.cfg.Clock.NewTicker(s.cfg.RestartInternal)
	restartTickChan := restartTicker.C()
	ctxDone := ctx.Done()
	for {
		select {
		case <-restartTickChan:
			for id, update := range s.latestStatusUpdates {
				if !update.Status.IsAlive() {
					s.cfg.Logger.Debug(
						"Restarting service",
						zap.String("serviceName", update.ServiceName),
						zap.Int("serviceID", update.ServiceID),
					)
					s.start(ctx, s.supervisedServices[id])
					s.notifyListeners()
				}
			}
		case update := <-s.statusUpdateChan:
			s.handleStatusUpdate(update)
		case <-ctxDone:
			restartTicker.Stop()
			for isAnyAlive(s.latestStatusUpdates) {
				s.handleStatusUpdate(<-s.statusUpdateChan) // TODO: add a timeout
			}
			return nil // TODO: error if any service failed
		}
	}
}

type contextKey struct{}

type contextValue struct {
	supervisedService *supervisedService
	clock             clock.Clock
	statusUpdateChan  chan StatusUpdate
}

// ReportTransientError is called by a service managed by a supervisor to flag that the functionality is degraded.
//
// Calling this function with nil as the error resolves a previously reported error.
func ReportTransientError(ctx context.Context, err error) error {
	valueOrNil := ctx.Value(contextKey{})
	if valueOrNil == nil {
		return errors.New("non-supervisor context")
	}
	value := valueOrNil.(contextValue)
	status := StatusRunning
	if err != nil {
		status = StatusTransientError
	}
	update := StatusUpdate{
		ServiceID:   value.supervisedService.id,
		ServiceName: value.supervisedService.name,
		Time:        value.clock.Now(),
		Status:      status,
		Err:         err,
	}
	select {
	case value.statusUpdateChan <- update:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Supervisor) handleStatusUpdate(update StatusUpdate) {
	s.cfg.Logger.Debug("Status update", zap.Object("update", update))
	s.latestStatusUpdates[update.ServiceID] = update
	s.notifyListeners()
}

func (s *Supervisor) start(ctx context.Context, ss *supervisedService) {
	s.latestStatusUpdates[ss.id] = StatusUpdate{
		ServiceID:   ss.id,
		ServiceName: ss.name,
		Time:        s.cfg.Clock.Now(),
		Status:      StatusIdle,
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				var err error
				if errPanic, ok := r.(error); ok {
					err = errors.Wrap(errPanic, "panic")
				} else {
					err = errors.Errorf("panic: %v", r)
				}
				s.statusUpdateChan <- StatusUpdate{
					ServiceID:   ss.id,
					ServiceName: ss.name,
					Time:        s.cfg.Clock.Now(),
					Status:      StatusPanic,
					Err:         err,
				}
			}
		}()
		ctx = context.WithValue(ctx, contextKey{}, contextValue{
			supervisedService: ss,
			clock:             s.cfg.Clock,
			statusUpdateChan:  s.statusUpdateChan,
		})
		if initializer, ok := ss.service.(Initializer); ok {
			s.statusUpdateChan <- StatusUpdate{
				ServiceID:   ss.id,
				ServiceName: ss.name,
				Time:        s.cfg.Clock.Now(),
				Status:      StatusInitializing,
			}
			if err := initializer.Initialize(ctx); err != nil {
				s.statusUpdateChan <- StatusUpdate{
					ServiceID:   ss.id,
					ServiceName: ss.name,
					Time:        s.cfg.Clock.Now(),
					Status:      StatusError,
					Err:         err,
				}
				return // fast-fail and wait to be restarted
			}
		}
		s.statusUpdateChan <- StatusUpdate{
			ServiceID:   ss.id,
			ServiceName: ss.name,
			Time:        s.cfg.Clock.Now(),
			Status:      StatusRunning,
		}
		err := ss.service.Start(ctx)
		status := StatusStopped
		if err != nil {
			status = StatusError
		}
		s.statusUpdateChan <- StatusUpdate{
			ServiceID:   ss.id,
			ServiceName: ss.name,
			Time:        s.cfg.Clock.Now(),
			Status:      status,
			Err:         err,
		}
	}()
}

func (s *Supervisor) notifyListeners() {
	if len(s.cfg.StatusUpdateListeners) == 0 {
		return
	}
	result := make([]StatusUpdate, len(s.latestStatusUpdates))
	copy(result, s.latestStatusUpdates)
	for _, listener := range s.cfg.StatusUpdateListeners {
		listener(result)
	}
}

func isAnyAlive(statusUpdates []StatusUpdate) bool {
	for _, statusUpdate := range statusUpdates {
		if statusUpdate.Status.IsAlive() {
			return true
		}
	}
	return false
}

func serviceName(service Service) string {
	if stringer, ok := service.(fmt.Stringer); ok {
		return stringer.String()
	}
	t := reflect.Indirect(reflect.ValueOf(service)).Type()
	return fmt.Sprintf("%s.%s", path.Base(t.PkgPath()), t.Name())
}
