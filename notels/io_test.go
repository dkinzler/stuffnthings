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
		{
			S:        "abcdefg",
			Start:    0,
			End:      4,
			OutStart: 0,
			OutEnd:   4,
		},
		{
			S:        "abcdefg",
			Start:    2,
			End:      4,
			OutStart: 2,
			OutEnd:   4,
		},
		{
			S:        "ä",
			Start:    0,
			End:      1,
			OutStart: 0,
			OutEnd:   0,
		},
		{
			S:        "ää",
			Start:    0,
			End:      3,
			OutStart: 0,
			OutEnd:   1,
		},
		{
			S:        "ääabc",
			Start:    4,
			End:      6,
			OutStart: 2,
			OutEnd:   4,
		},
	}

	for i, c := range cases {
		start, end := ToUTF16Range(c.S, c.Start, c.End)
		assert.Equal(c.OutStart, start, "test case %v", i)
		assert.Equal(c.OutEnd, end, "test case %v", i)
	}
}
