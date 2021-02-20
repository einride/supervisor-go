# go.einride.tech/supervisor

A service that manages service lifetimes.

A supervisor is essentially a more capable errgroup. It monitors a set
of running services, and restarts them if they fail.
The supervisor keeps track of the status of each service and reports any
status changes to listeners via a callback.

## Examples

### Supervising multiple services

Just as with errgroups, a supervisor can manage multiple services.

```go
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
	// Start the supervisor (blocking call).
	err := s.Start(ctx)
	if err != nil {
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
```

### Restarting crashed services

The main difference from errgroups is that a supervisor will restart a crashed service.

```go
func ExampleNew() {
	// Restart stopped services every 10ms.
	cfg := supervisor.Config{
		RestartInterval: 10 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	starts := 0
	svc := supervisor.NewService("example", func(ctx context.Context) error {
		starts++
		if starts > 3 {
			cancel()
			return nil
		}
		return fmt.Errorf("oops")
	})
	cfg.Services = append(cfg.Services, svc)
	s := supervisor.New(&cfg)
	if err := s.Start(ctx); err != nil {
        // no error currently returned
	}
	fmt.Println("service restarted", starts, "times")
	// Output:
	// service restarted 3 times
}
```
