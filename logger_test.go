package supervisor

import (
	"bytes"
	"log"
	"testing"

	"gotest.tools/v3/assert"
)

func TestNewStdLogger(t *testing.T) {
	// given an output buffer and a logger
	var buf bytes.Buffer
	logger := NewStdLogger(&buf)
	// when
	logger.Debugf("Hello, %s!", "debug")
	logger.Warningf("Hello, %s!", "warning")
	// then
	assert.Equal(t, "DEBUG: Hello, debug!\nWARN: Hello, warning!\n", buf.String())
}

func TestStdLoggerOpts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		opts   *StdLoggerOpts
		wanted string
	}{
		{
			name:   "Default",
			opts:   DefaultLoggerOpts(),
			wanted: "DEBUG: hello, debug\nWARN: hello, warn\n",
		},
		{
			name:   "Set log level to warning",
			opts:   DefaultLoggerOpts().WithLogLevel(LogLevelWarning),
			wanted: "WARN: hello, warn\n",
		},
		{
			name:   "Set log flags",
			opts:   DefaultLoggerOpts().WithFlags(log.Lshortfile),
			wanted: "DEBUG: logger.go:52: hello, debug\nWARN: logger.go:59: hello, warn\n",
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// given
			var buf bytes.Buffer
			logger := tc.opts.WithDebugOutput(&buf).WithWarningOutput(&buf).New()
			// when
			logger.Debugf("hello, %v", "debug")
			logger.Warningf("hello, %v", "warn")
			// then
			assert.Equal(t, tc.wanted, buf.String())
		})
	}
}
