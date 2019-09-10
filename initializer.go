package supervisor

import "context"

// Initializer is any service with a pre-start initialization step.
type Initializer interface {
	Initialize(context.Context) error
}

// NewInitializerService creates a new service from an initialization function and a start function.
func NewInitializerService(
	name string,
	initializeFn func(context.Context) error,
	startFn func(context.Context) error) Service {
	return &fnInitializerService{name: name, initializeFn: initializeFn, startFn: startFn}
}

type fnInitializerService struct {
	name         string
	initializeFn func(context.Context) error
	startFn      func(context.Context) error
}

// String returns the name of the service.
func (f *fnInitializerService) String() string {
	return f.name
}

// Initialize the service.
func (f *fnInitializerService) Initialize(ctx context.Context) error {
	return f.initializeFn(ctx)
}

// Start the service.
func (f *fnInitializerService) Run(ctx context.Context) error {
	return f.startFn(ctx)
}
