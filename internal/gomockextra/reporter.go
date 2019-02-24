package gomockextra

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
)

// GoroutineReporter returns a reporter that works with multiple goroutines.
//
// Specifically, the reporter invokes panic on Fatalf to crash the test.
//
// Without this reporter, if a test failure happens on a separate goroutine,
// the test failure is swallowed and the test potentially runs forever.
func GoroutineReporter(t *testing.T) gomock.TestReporter {
	return &goroutineReporter{T: t}
}

type goroutineReporter struct {
	T *testing.T
}

func (r goroutineReporter) Errorf(format string, args ...interface{}) {
	r.T.Errorf(format, args...)
}

func (r goroutineReporter) Fatalf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}
