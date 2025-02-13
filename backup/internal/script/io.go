package script

import (
	"bufio"
	"fmt"
	"os"
)

// Why an interface?
// We might want to use another output implementation later.
// Package fmt works but is not safe for concurrent use.
// Package log is safe for concurrent use but always adds a newline to the end of each call
// which doesn't allow you to have inline prompts.
type output interface {
	Printf(format string, a ...any) (int, error)
	Println(a ...any) (int, error)
}

type fmtOutput struct{}

func (f fmtOutput) Printf(format string, a ...any) (int, error) {
	return fmt.Printf(format, a...)
}

func (f fmtOutput) Println(a ...any) (int, error) {
	return fmt.Println(a...)
}

var out output = fmtOutput{}

type input interface {
	ReadLine() (string, error)
}

type defaultInput struct {
	reader *bufio.Reader
}

func newDefaultInput() defaultInput {
	return defaultInput{
		reader: bufio.NewReader(os.Stdin),
	}
}

func (i defaultInput) ReadLine() (string, error) {
	return i.reader.ReadString('\n')
}

var in input = newDefaultInput()
