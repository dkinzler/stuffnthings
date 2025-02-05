package exec

import (
	"context"
	"io"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type options struct {
	returnStdout bool
	returnStderr bool
	// only for execBackground
	stdin   string
	timeout time.Duration
}

type Option func(*options)

// default is to return stdout
func WithStdout(returnStdout bool) Option {
	return func(o *options) {
		o.returnStdout = returnStdout
	}
}

// default is to return stderr
func WithStderr(returnStderr bool) Option {
	return func(o *options) {
		o.returnStderr = returnStderr
	}
}

func WithStdin(stdin string) Option {
	return func(o *options) {
		o.stdin = stdin
	}
}

// default 10s
func WithTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.timeout = timeout
	}
}

func defaultOptions() *options {
	return &options{
		returnStdout: true,
		returnStderr: true,
		stdin:        "",
		timeout:      time.Second * 10,
	}
}

func (o *options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

type Result struct {
	// for convenience the command that was run
	Cmd      []string
	ExitCode int
	Err      error
	Stdout   string
	Stderr   string
	// time it took the command to run
	Time time.Duration
}

type Callback func(Result) tea.Msg

// Runs a command in the foreground by effectively pausing the program and handing over the terminal window.
// Wraps the tea.Exec function to return a more detailed result including the exit code, outputs on stdin and stderr and more.
// Useful for interactive commands where you also want the standard and error output to e.g. show a detailed error screen.
// Note that some programs write error messages to stdout.
func Foreground(cmd []string, fn Callback, opts ...Option) tea.Cmd {
	options := defaultOptions()
	options.apply(opts...)

	var name string
	var args []string
	if len(cmd) > 0 {
		name = cmd[0]
		args = cmd[1:]
	}

	var outBuffer *strings.Builder
	if options.returnStdout {
		outBuffer = &strings.Builder{}
	}
	var errBuffer *strings.Builder
	if options.returnStderr {
		errBuffer = &strings.Builder{}
	}
	c := newCommand(exec.Command(name, args...), outBuffer, errBuffer)
	startTime := time.Now()
	return tea.Exec(c, func(err error) tea.Msg {
		if fn == nil {
			return nil
		}

		var result Result
		result.Cmd = cmd
		result.Time = time.Now().Sub(startTime)
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
		if options.returnStdout {
			result.Stdout = outBuffer.String()
		}
		if options.returnStderr {
			result.Stderr = errBuffer.String()
		}
		return fn(result)
	})
}

func Background(cmd []string, fn Callback, opts ...Option) tea.Cmd {
	return func() tea.Msg {
		options := defaultOptions()
		options.apply(opts...)

		ctx, cancel := context.WithTimeout(context.Background(), options.timeout)
		defer cancel()

		var name string
		var args []string
		if len(cmd) > 0 {
			name = cmd[0]
			args = cmd[1:]
		}

		c := exec.CommandContext(ctx, name, args...)
		if options.stdin != "" {
			c.Stdin = strings.NewReader(options.stdin)
		}
		var outBuffer *strings.Builder
		if options.returnStdout {
			outBuffer = &strings.Builder{}
		}
		c.Stdout = outBuffer
		var errBuffer *strings.Builder
		if options.returnStderr {
			errBuffer = &strings.Builder{}
		}
		c.Stderr = errBuffer
		if fn == nil {
			return nil
		}

		startTime := time.Now()
		err := c.Run()

		if fn == nil {
			return nil
		}
		var result Result
		result.Cmd = cmd
		result.Time = time.Now().Sub(startTime)
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
		if options.returnStdout {
			result.Stdout = outBuffer.String()
		}
		if options.returnStderr {
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
