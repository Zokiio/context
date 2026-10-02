package cli

import (
	"io"
	"os"
)

// Environment supplies invocation state without changing process directories or
// consulting home configuration before a command needs it.
type Environment struct {
	WorkingDirectory func() (string, error)
	HomeDirectory    func() (string, error)
	Executable       func() (string, error)
	Input            io.Reader
	IsTerminal       func() bool
}

func (environment Environment) defaults() Environment {
	if environment.WorkingDirectory == nil {
		environment.WorkingDirectory = os.Getwd
	}
	if environment.HomeDirectory == nil {
		environment.HomeDirectory = os.UserHomeDir
	}
	if environment.Input == nil {
		environment.Input = os.Stdin
	}
	if environment.Executable == nil {
		environment.Executable = os.Executable
	}
	return environment
}
