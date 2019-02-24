package supervisor

import "context"

// Service that can be managed by a supervisor.
type Service interface {
	Start(context.Context) error
}

// NewService creates a new service from a function.
func NewService(name string, fn func(context.Context) error) Service {
	return &fnService{name: name, fn: fn}
}

type fnService struct {
	name string
	fn   func(context.Context) error
}

// String returns the name of the service.
func (f fnService) String() string {
	return f.name
}

// Start the service.
func (f fnService) Start(ctx context.Context) error {
	return f.fn(ctx)
}
