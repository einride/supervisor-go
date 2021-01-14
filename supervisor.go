// Package supervisor provides a supervisor that monitors, manages the lifetime and reports status on services.
package supervisor

import (
	"context"
	"fmt"
	"path"
	"reflect"
	"runtime/debug"
	"time"
)

// Config contains the full set of dependencies for a supervisor.
type Config struct {
	Services              []Service
	StatusUpdateListeners []func([]StatusUpdate)
	RestartInterval       time.Duration
	Clock                 Clock
	Logger                Logger
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
		cfg:              cfg,
		statusUpdateChan: make(chan StatusUpdate),
	}
	if cfg.Clock == nil {
		s.cfg.Clock = NewSystemClock()
	}
	if cfg.Logger == nil {
		s.cfg.Logger = &nopLogger{}
	}
	var id int
	for _, service := range cfg.Services {
		if service != nil && !reflect.ValueOf(service).IsNil() {
			s.supervisedServices = append(s.supervisedServices, &supervisedService{
				service: service,
				id:      id,
				name:    serviceName(service),
			})
			id++
		}
	}
	s.latestStatusUpdates = make([]StatusUpdate, len(s.supervisedServices))
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
	restartTicker := s.cfg.Clock.NewTicker(s.cfg.RestartInterval)
	restartTickChan := restartTicker.C()
	ctxDone := ctx.Done()
	for {
		select {
		case <-restartTickChan:
			for id, update := range s.latestStatusUpdates {
				if !update.Status.IsAlive() {
					s.cfg.Logger.Warningf(
						"restarting service %s: %v",
						update.ServiceName,
						update,
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
				for _, u := range s.latestStatusUpdates {
					if u.Status.IsAlive() {
						s.cfg.Logger.Debugf("service alive: %v", u)
					}
				}
				s.handleStatusUpdate(<-s.statusUpdateChan) // TODO: add a timeout
			}
			return nil // TODO: error if any service failed
		}
	}
}

func (s *Supervisor) handleStatusUpdate(update StatusUpdate) {
	s.cfg.Logger.Debugf("received update: %v", update)
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
				s.statusUpdateChan <- StatusUpdate{
					ServiceID:   ss.id,
					ServiceName: ss.name,
					Time:        s.cfg.Clock.Now(),
					Status:      StatusPanic,
					Err:         fmt.Errorf("%v: %s", r, string(debug.Stack())),
				}
			}
		}()
		s.statusUpdateChan <- StatusUpdate{
			ServiceID:   ss.id,
			ServiceName: ss.name,
			Time:        s.cfg.Clock.Now(),
			Status:      StatusRunning,
		}
		err := ss.service.Run(ctx)
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
