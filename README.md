# supervisor-go

A service that manages service lifetimes.

A supervisor is essentially a more capable errgroup. It monitors a set
of running services, and restarts them if they fail.
The supervisor keeps track of the status of each service and reports any
status changes to listeners via a callback.

## Examples

### Supervising a service

```go
import (
	"context"
	"errors"
	"fmt"
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
```
