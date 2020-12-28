package supervisor

import (
	"io"
	"log"
	"os"
)

//go:generate stringer -type=LogLevel -trimprefix=LogLevel

// Logger represents a pluggable logger service.
//
// Both method should expect format string to be compatible with fmt
// library.
type Logger interface {
	Debugf(format string, values ...interface{})
	Warningf(format string, values ...interface{})
}

// nopLogger is a supervisor.Logger that does not log anything.
type nopLogger struct{}

func (l *nopLogger) Debugf(format string, values ...interface{})   {}
func (l *nopLogger) Warningf(format string, values ...interface{}) {}

// StdLogger is a supervisor.Logger that is backed by log.Logger and
// has a customizable log level.
type StdLogger struct {
	debug   *log.Logger
	warning *log.Logger
	level   LogLevel
}

type LogLevel uint

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
)

// NewStdLogger returns a Logger which logs to out with LogLevelDebug.
func NewStdLogger(out io.Writer) *StdLogger {
	return DefaultLoggerOpts().WithDebugOutput(out).WithWarningOutput(out).New()
}

func (l *StdLogger) Debugf(message string, values ...interface{}) {
	if l.level > LogLevelDebug || l.debug == nil {
		return
	}
	l.debug.Printf(message, values...)
}

func (l *StdLogger) Warningf(message string, values ...interface{}) {
	if l.level > LogLevelWarning || l.warning == nil {
		return
	}
	l.warning.Printf(message, values...)
}

// StdLoggerOpts represents the tunable knobs for creating a
// customized StdLogger.
type StdLoggerOpts struct {
	Flags         int
	LogLevel      LogLevel
	DebugOutput   io.Writer
	WarningOutput io.Writer
}

// DefaultLoggerOpts returns options for creating a StdLogger that
// logs to os.Stderr and logs with Debug level.
func DefaultLoggerOpts() *StdLoggerOpts {
	return &StdLoggerOpts{DebugOutput: os.Stderr, WarningOutput: os.Stderr}
}

// WithDebugOutput sets the logging output for debug prints.
func (o *StdLoggerOpts) WithDebugOutput(out io.Writer) *StdLoggerOpts {
	o.DebugOutput = out
	return o
}

// WithWarningOutput sets the logging output for warning prints.
func (o *StdLoggerOpts) WithWarningOutput(out io.Writer) *StdLoggerOpts {
	o.WarningOutput = out
	return o
}

// WithLevel sets the log level of the logger.
func (o *StdLoggerOpts) WithLogLevel(level LogLevel) *StdLoggerOpts {
	o.LogLevel = level
	return o
}

// WithFlags sets the logging flags of the logger.
func (o *StdLoggerOpts) WithFlags(flags int) *StdLoggerOpts {
	o.Flags = flags
	return o
}

// New creates a StdLogger from the options.
func (o *StdLoggerOpts) New() *StdLogger {
	return &StdLogger{
		debug:   log.New(o.DebugOutput, "DEBUG: ", o.Flags),
		warning: log.New(o.WarningOutput, "WARN: ", o.Flags),
		level:   o.LogLevel,
	}
}
