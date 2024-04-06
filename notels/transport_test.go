package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRead(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		Input        string
		Output       []byte
		ReturnsError bool
	}{
		{
			Input:        "Content-Length: 8\r\n\r\nabcdefgh",
			Output:       []byte("abcdefgh"),
			ReturnsError: false,
		},
		{
			Input:        "Content-Length: 4\r\n\r\nabcdefgh",
			Output:       []byte("abcd"),
			ReturnsError: false,
		},
		{
			Input:        "Content-Length: 14\r\n\r\nabcdefgh\r\n1234",
			Output:       []byte("abcdefgh\r\n1234"),
			ReturnsError: false,
		},
		{
			Input:        "randomstuff\r\n\r\n",
			Output:       nil,
			ReturnsError: true,
		},
		{
			Input:        "Cntent-Length: 8\r\n\r\nabcdefgh",
			Output:       nil,
			ReturnsError: true,
		},
		{
			Input:        "Content-Type: abc\r\n\r\nabcfdefg",
			Output:       nil,
			ReturnsError: true,
		},
	}

	for i, c := range cases {
		r := NewReader(strings.NewReader(c.Input))
		out, _, err := r.ReadMessage()
		if c.ReturnsError {
			assert.NotNil(err)
		} else {
			assert.Nil(err)
			assert.Equal(c.Output, out, "test case %v", i)
		}
	}
}

func TestReadingMultipleMessages(t *testing.T) {
	assert := assert.New(t)

	m := "Content-Length: 4\r\n\r\nabcdContent-Length: 5\r\n\r\nefghi"
	r := NewReader(strings.NewReader(m))

	out, _, err := r.ReadMessage()
	assert.Nil(err)
	assert.Equal([]byte("abcd"), out)

	out, _, err = r.ReadMessage()
	assert.Nil(err)
	assert.Equal([]byte("efghi"), out)
}

func TestReadEOF(t *testing.T) {
	assert := assert.New(t)

	m := "Content-Length: 4\r\n\r\nabcd"
	r := NewReader(strings.NewReader(m))

	_, _, err := r.ReadMessage()
	assert.Nil(err)

	_, _, err = r.ReadMessage()
	assert.NotNil(err)
	assert.True(IsEOFError(err))
}

func TestWrite(t *testing.T) {
	assert := assert.New(t)

	input1 := []byte("abcd")
	input2 := []byte("efghi")
	output1 := []byte("Content-Length: 4\r\nContent-Type: application/vscode-jsonrpc; charset=utf-8\r\n\r\nabcd")
	output2 := []byte("Content-Length: 5\r\nContent-Type: application/vscode-jsonrpc; charset=utf-8\r\n\r\nefghi")

	var b strings.Builder
	w := NewWriter(&b)
	err := w.Write(input1)
	assert.Nil(err)
	err = w.Write(input2)
	assert.Nil(err)

	combined := append(output1, output2...)
	assert.Equal(string(combined), b.String())
}
