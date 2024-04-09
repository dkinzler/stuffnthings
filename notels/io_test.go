package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUTF16RangeConversion(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		S        string
		Start    int
		End      int
		OutStart int
		OutEnd   int
	}{
		{"abcdefg", 0, 4, 0, 4},
		{"abcdefg", 2, 4, 2, 4},
		{"ä", 0, 1, 0, 0},
		{"ää", 0, 3, 0, 1},
		{"ääabc", 4, 6, 2, 4},
		// Multi-byte UTF-8, but single code unit UTF-16
		{"こんにちは", 0, 2, 0, 0},
		{"こんにちは", 3, 5, 1, 1},
		// Multi-byte UTF-8, multi code unit UTF-16
		{"\U0001F050\U0001F065\U0001F08D", 0, 3, 0, 1},
		{"🁐🁥🂍", 0, 3, 0, 1},
		{"🁐🁥🂍", 4, 7, 2, 3},
		{"🁐🁥🂍", 4, 11, 2, 5},
		{"🁐🁥a🂍", 4, 12, 2, 6},
	}

	for i, c := range cases {
		start, end := ToUTF16Range([]byte(c.S), c.Start, c.End)
		assert.Equal(c.OutStart, start, "test case %v", i)
		assert.Equal(c.OutEnd, end, "test case %v", i)
	}
}
