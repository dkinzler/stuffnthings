package exec

import (
	"context"
	"io"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TODO maybe still use it?
// Note: we could have also used the functional options pattern, but this works well enough for now.
type Options struct {
	ReturnStdout bool
	ReturnStderr bool
	// only for execBackground
	Stdin   string
	Timeout time.Duration
}

func DefaultOptions() Options {
	return Options{
		ReturnStdout: true,
		ReturnStderr: true,
		Stdin:        "",
		Timeout:      time.Second * 10,
	}
}

type Result struct {
	// for convenience the command that was run
	Cmd      []string
	ExitCode int
	Err      error
	Stdout   string
	Stderr   string
}

type Callback func(Result) tea.Msg

// TODO polish comment
// Runs the given command in the foreground by effectively pausing this program and handing over the terminal screen.
// Extends the tea.Exec function to return a more detailed result including e.g. the exit code and output on stdin and stderr.
// Useful for interactive commands where you also want the standard and error output e.g. to provide better error messages.
// Note that not some programs write error messages to stdout.
func Foreground(cmd []string, fn Callback, options Options) tea.Cmd {
	var name string
	var args []string
	if len(cmd) > 0 {
		name = cmd[0]
		args = cmd[1:]
	}

	var outBuffer *strings.Builder
	if options.ReturnStdout {
		outBuffer = &strings.Builder{}
	}
	var errBuffer *strings.Builder
	if options.ReturnStderr {
		errBuffer = &strings.Builder{}
	}
	c := newCommand(exec.Command(name, args...), outBuffer, errBuffer)
	return tea.Exec(c, func(err error) tea.Msg {
		if fn == nil {
			return nil
		}
		var result Result
		result.Cmd = cmd
		if err != nil {
			result.ExitCode = -1
			e, ok := err.(*exec.ExitError)
			if ok {
				if e.Exited() {
					result.ExitCode = e.ExitCode()
				} else {
					result.Err = err
				}
			} else {
				result.Err = err
			}
		}
		if options.ReturnStdout {
			result.Stdout = outBuffer.String()
		}
		if options.ReturnStderr {
			result.Stderr = errBuffer.String()
		}
		return fn(result)
	})
}

func Background(cmd []string, fn Callback, options Options) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
		defer cancel()

		var name string
		var args []string
		if len(cmd) > 0 {
			name = cmd[0]
			args = cmd[1:]
		}

		c := exec.CommandContext(ctx, name, args...)
		if options.Stdin != "" {
			c.Stdin = strings.NewReader(options.Stdin)
		}
		var outBuffer *strings.Builder
		if options.ReturnStdout {
			outBuffer = &strings.Builder{}
		}
		c.Stdout = outBuffer
		var errBuffer *strings.Builder
		if options.ReturnStderr {
			errBuffer = &strings.Builder{}
		}
		c.Stderr = errBuffer
		if fn == nil {
			return nil
		}

		err := c.Run()

		if fn == nil {
			return nil
		}
		var result Result
		result.Cmd = cmd
		if err != nil {
			result.ExitCode = -1
			e, ok := err.(*exec.ExitError)
			if ok {
				if e.Exited() {
					result.ExitCode = e.ExitCode()
				} else {
					result.Err = err
				}
			} else {
				result.Err = err
			}
		}
		if options.ReturnStdout {
			result.Stdout = outBuffer.String()
		}
		if options.ReturnStderr {
			result.Stderr = errBuffer.String()
		}
		return fn(result)
	}
}

func newCommand(c *exec.Cmd, out, err io.Writer) *command {
	return &command{Cmd: c, out: out, err: err}
}

type command struct {
	*exec.Cmd
	out io.Writer
	err io.Writer
}

func (c *command) SetStdin(r io.Reader) {
	if c.Stdin == nil {
		c.Stdin = r
	}
}

func (c *command) SetStdout(w io.Writer) {
	if c.Stdout == nil {
		if c.out != nil {
			c.Stdout = io.MultiWriter(w, c.out)
		} else {
			c.Stdout = w
		}
	}
}

func (c *command) SetStderr(w io.Writer) {
	if c.Stderr == nil {
		if c.err != nil {
			c.Stderr = io.MultiWriter(w, c.err)
		} else {
			c.Stderr = w
		}
	}
}
