package supervisor

import "context"

// Service that can be managed by a supervisor.
type Service interface {
	Run(context.Context) error
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

// Run the service.
func (f fnService) Run(ctx context.Context) error {
	return f.fn(ctx)
}
