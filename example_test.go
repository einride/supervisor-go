package supervisor_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-logr/stdr"
	"go.einride.tech/supervisor"
)

func ExampleSupervisor() {
	// Restart stopped services every 10ms.
	cfg := supervisor.Config{
		RestartInterval: 10 * time.Millisecond,
		// No specified clock returns system clock
		// No specified logger returns a nop-logger
	}
	// Create a context which can be canceled.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// Create pingpong table
	table := make(chan int)
	roundsToPlay := 2
	// Create player services.
	pingService := supervisor.NewService("ping", func(ctx context.Context) error {
		i := roundsToPlay
		for {
			select {
			case <-ctx.Done():
				return fmt.Errorf("timeout")
			case table <- i:
				fmt.Println("ping")
				i = <-table
				if i == 0 {
					close(table)
					cancel()
					return nil
				}
			}
		}
	})
	pongService := supervisor.NewService("pong", func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return fmt.Errorf("timeout")
			case i := <-table:
				if i == 0 {
					return nil
				}
				table <- i - 1
				fmt.Println("pong")
			}
		}
	})
	// Add service to the supervised services.
	cfg.Services = append(cfg.Services, pingService, pongService)
	// Create the supervisor from the config.
	s := supervisor.New(&cfg)
	// Run the supervisor (blocking call).
	if err := s.Run(ctx); err != nil {
		// handle error
		panic(err)
	}
	defer cancel()
	// Output:
	// ping
	// pong
	// ping
	// pong
}

func ExampleNew() {
	// Restart stopped services every 10ms.
	cfg := supervisor.Config{
		RestartInterval: 10 * time.Millisecond,
		Logger:          stdr.New(log.New(os.Stderr, "", log.LstdFlags)),
	}
	// Create a context that can be canceled inside the service.
	ctx, cancel := context.WithCancel(context.Background())
	starts := 0
	svc := supervisor.NewService("example", func(_ context.Context) error {
		if starts == 3 {
			cancel()
			return nil
		}
		starts++
		return fmt.Errorf("oops")
	})
	// Add service to set of supervised services.
	cfg.Services = append(cfg.Services, svc)
	// Create supervisor from config.
	s := supervisor.New(&cfg)
	// Run supervisor (blocking).
	_ = s.Run(ctx) // no error currently reported
	fmt.Println("service restarted", starts, "times")
	// Output:
	// service restarted 3 times
}

func ExampleConfig_StatusUpdateListeners() {
	// Restart stopped services every 10ms.
	cfg := supervisor.Config{
		RestartInterval: 10 * time.Millisecond,
		Services: []supervisor.Service{
			// Create a crashing service.
			supervisor.NewService("example", func(_ context.Context) error {
				return fmt.Errorf("oops")
			}),
		},
		Logger: stdr.New(log.New(os.Stderr, "", log.LstdFlags)),
	}
	// Create a context that can be canceled.
	ctx, cancel := context.WithCancel(context.Background())
	stops := 0
	// Create a statusupdate listener that cancels the context
	// after the example service crashes 3 times.
	cfg.StatusUpdateListeners = append(cfg.StatusUpdateListeners, func(updates []supervisor.StatusUpdate) {
		for _, update := range updates {
			if update.ServiceName == "example" &&
				update.Status == supervisor.StatusError ||
				update.Status == supervisor.StatusStopped {
				stops++
			}
		}
		if stops == 3 {
			cancel()
		}
	})
	s := supervisor.New(&cfg)
	_ = s.Run(ctx) // no error currently reported
	fmt.Println("service stopped", stops, "times")
	// Output:
	// service stopped 3 times
}
