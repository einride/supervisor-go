package supervisor_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/einride/supervisor-go"
)

func ExampleSupervisor() {
	// Restart stopped services every 10ms.
	cfg := supervisor.Config{
		RestartInterval: 10 * time.Millisecond,
		// No specified clock returns system clock
		// No specified logger returns a nop-logger
	}
	// Register a listener that prints all updates
	listener := func(updates []supervisor.StatusUpdate) {
		for _, update := range updates {
			fmt.Printf("%v: %v\n", update.ServiceName, update.Status)
		}
	}
	cfg.StatusUpdateListeners = append(cfg.StatusUpdateListeners, listener)
	// Service that sleeps and then crashes.
	svc := supervisor.NewService("example", func(ctx context.Context) error {
		return errors.New("timeout")
	})
	// Add service to the supervised services.
	cfg.Services = append(cfg.Services, svc)
	// Create the supervisor from the config.
	s := supervisor.New(&cfg)
	// Create a context which will timeout immediately, the supervisor will still
	// wait for its services to exit before exiting.
	ctx, cancel := context.WithTimeout(context.Background(), 0*time.Millisecond)
	// Start the supervisor (blocking call).
	err := s.Start(ctx)
	if err != nil {
		// handle error
	}
	defer cancel()
	// Output:
	// example: Idle
	// example: Running
	// example: Error
}

func ExampleNewStdLogger() {
	// Create a supervisor.StdLogger
	logger := supervisor.NewStdLogger(os.Stdout)

	// Log a debug message
	logger.Debugf("this message supports %s", "Printf formatting")

	// Log a warning message
	logger.Warningf("this is a warning")
	// Output:
	// DEBUG: this message supports Printf formatting
	// WARN: this is a warning
}

func ExampleStdLogger() {
	// Create a supervisor.StdLogger that only logs on warning
	// level to os.Stdout and sets log.Flags to the underlying
	// log.Logger.
	logger := supervisor.DefaultLoggerOpts().
		WithWarningOutput(os.Stdout).
		WithFlags(log.Lshortfile).
		WithLogLevel(supervisor.LogLevelWarning).
		New()

	// Log a debug message
	logger.Debugf("this message supports %s", "Printf formatting")

	// Log a warning message
	logger.Warningf("this is a warning")
	// Output:
	// WARN: logger.go:59: this is a warning
}
