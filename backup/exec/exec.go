package exec

import (
	"io"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type ExecCallback func(error, string) tea.Msg

// Extends the tea.Exec function to also pass the stderr and stdout output into the callback function.
// Useful for interactive commands where you want the error output if they fail.
// If includeStdout is false, only stderr is passed.
func Exec(cmd *exec.Cmd, fn ExecCallback, includeStdout bool) tea.Cmd {
	errOut := &strings.Builder{}
	var out io.Writer
	if includeStdout {
		out = errOut
	}
	c := newExecCommand(cmd, out, errOut)
	return tea.Exec(c, func(err error) tea.Msg {
		if fn != nil {
			return fn(err, errOut.String())
		}
		return nil
	})
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
