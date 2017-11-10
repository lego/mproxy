package log

import (
	"fmt"

	"github.com/mgutz/ansi"
)

type Verbosity int

const (
	Warn Verbosity = iota
	Info
	Debug
	Trace
)

type Color string

const (
	Blue           Color = "blue"
	Green                = "green"
	Red                  = "red"
	Magenta              = "magenta"
	RedEmphasized        = "red+b:white"
	BlueEmphasized       = "blue+b:white"
)

// Logger - Interface to pass into Proxy for it to log messages
type Logger interface {
	Trace(f string, args ...interface{})
	Debug(f string, args ...interface{})
	Info(f string, args ...interface{})
	Warn(f string, args ...interface{})
	Log(verbosity Verbosity, f string, args ...interface{})
	LogC(verbosity Verbosity, color Color, f string, args ...interface{})
}

// NullLogger - An empty logger that ignores everything
type NullLogger struct{}

// Trace - no-op
func (l NullLogger) Trace(f string, args ...interface{}) {}

// Debug - no-op
func (l NullLogger) Debug(f string, args ...interface{}) {}

// Info - no-op
func (l NullLogger) Info(f string, args ...interface{}) {}

// Warn - no-op
func (l NullLogger) Warn(f string, args ...interface{}) {}

// Log - no-op
func (l NullLogger) Log(verbosity Verbosity, f string, args ...interface{}) {}

// LogC - no-op
func (l NullLogger) LogC(verbosity Verbosity, color Color, f string, args ...interface{}) {}

// ColorLogger - A Logger that logs to stdout in color
type ColorLogger struct {
	VeryVerbose bool
	Verbose     bool
	Prefix      string
	Color       bool
}

// Trace - Log a very verbose trace message
func (l ColorLogger) Trace(f string, args ...interface{}) {
	if !l.VeryVerbose {
		return
	}
	l.output(Green, f, args...)
}

// Debug - Log a debug message
func (l ColorLogger) Debug(f string, args ...interface{}) {
	if !l.Verbose {
		return
	}
	l.output(Green, f, args...)
}

// Info - Log a general message
func (l ColorLogger) Info(f string, args ...interface{}) {
	l.output(Blue, f, args...)
}

// Warn - Log a warning
func (l ColorLogger) Warn(f string, args ...interface{}) {
	l.output(Red, f, args...)
}

// Log
func (l ColorLogger) Log(verbosity Verbosity, f string, args ...interface{}) {
	if l.VeryVerbose && verbosity > Trace {
		return
	} else if l.Verbose && verbosity > Debug {
		return
	}
	l.output("", f, args...)
}

// LogC
func (l ColorLogger) LogC(verbosity Verbosity, color Color, f string, args ...interface{}) {
	if l.VeryVerbose && verbosity > Trace {
		return
	} else if l.Verbose && verbosity > Debug {
		return
	}
	l.output(color, f, args...)
}

func (l ColorLogger) output(color Color, f string, args ...interface{}) {
	if l.Color && color != "" {
		f = ansi.Color(f, string(color))
	}
	fmt.Printf(fmt.Sprintf("%s%s\n", l.Prefix, f), args...)
}
