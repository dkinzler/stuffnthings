package internal

import (
	"context"
	"io"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Note: we could have also used the functional options pattern, but this works well enough for now.
type execOptions struct {
	returnStdout bool
	returnStderr bool
	timeout      time.Duration
}

func defaultExecOptions() execOptions {
	return execOptions{
		returnStdout: true,
		returnStderr: true,
		timeout:      time.Second * 10,
	}
}

type execResult struct {
	// for convenience the command that was run
	cmd      []string
	exitCode int
	err      error
	stdout   string
	stderr   string
}

type execCallback func(execResult) tea.Msg

// TODO polish comment
// Runs the given command in the foreground by effectively pausing this program and handing over the terminal screen.
// Extends the tea.Exec function to return a more detailed result including e.g. the exit code and output on stdin and stderr.
// Useful for interactive commands where you also want the standard and error output e.g. to provide better error messages.
// Note that not some programs write error messages to stdout.
func execForeground(cmd []string, fn execCallback, options execOptions) tea.Cmd {
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
	c := newExecCommand(exec.Command(name, args...), outBuffer, errBuffer)
	return tea.Exec(c, func(err error) tea.Msg {
		if fn == nil {
			return nil
		}
		var result execResult
		result.cmd = cmd
		if err != nil {
			result.exitCode = -1
			e, ok := err.(*exec.ExitError)
			if ok {
				if e.Exited() {
					result.exitCode = e.ExitCode()
				} else {
					result.err = err
				}
			} else {
				result.err = err
			}
		}
		if options.returnStdout {
			result.stdout = outBuffer.String()
		}
		if options.returnStderr {
			result.stderr = errBuffer.String()
		}
		return fn(result)
	})
}

func execBackground(cmd []string, fn execCallback, options execOptions) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), options.timeout)
		defer cancel()

		var name string
		var args []string
		if len(cmd) > 0 {
			name = cmd[0]
			args = cmd[1:]
		}

		c := exec.CommandContext(ctx, name, args...)
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

		err := c.Run()

		if fn == nil {
			return nil
		}
		var result execResult
		result.cmd = cmd
		if err != nil {
			result.exitCode = -1
			e, ok := err.(*exec.ExitError)
			if ok {
				if e.Exited() {
					result.exitCode = e.ExitCode()
				} else {
					result.err = err
				}
			} else {
				result.err = err
			}
		}
		if options.returnStdout {
			result.stdout = outBuffer.String()
		}
		if options.returnStderr {
			result.stderr = errBuffer.String()
		}
		return fn(result)
	}
}

func newExecCommand(c *exec.Cmd, out, err io.Writer) *execCommand {
	return &execCommand{Cmd: c, out: out, err: err}
}

type execCommand struct {
	*exec.Cmd
	out io.Writer
	err io.Writer
}

func (c *execCommand) SetStdin(r io.Reader) {
	if c.Stdin == nil {
		c.Stdin = r
	}
}

func (c *execCommand) SetStdout(w io.Writer) {
	if c.Stdout == nil {
		if c.out != nil {
			c.Stdout = io.MultiWriter(w, c.out)
		} else {
			c.Stdout = w
		}
	}
}

func (c *execCommand) SetStderr(w io.Writer) {
	if c.Stderr == nil {
		if c.err != nil {
			c.Stderr = io.MultiWriter(w, c.err)
		} else {
			c.Stderr = w
		}
	}
}
