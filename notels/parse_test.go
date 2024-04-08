package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLinksInLine(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		Line          string
		LineIndex     int
		ExpectedLinks []Link
		ReturnsError  bool
	}{
		{
			Line:          "",
			LineIndex:     0,
			ExpectedLinks: nil,
			ReturnsError:  false,
		},
		{
			Line:      "[[abc/def.hij]]",
			LineIndex: 42,
			ExpectedLinks: []Link{
				{
					Path: "abc/def.hij",
					Range: Range{
						Start: Position{
							Line:      42,
							Character: 0,
						},
						End: Position{
							Line:      42,
							Character: 15,
						},
					},
				},
			},
			ReturnsError: false,
		},
		{
			Line:      "[[abc/def.hij]] [[x.y]]",
			LineIndex: 42,
			ExpectedLinks: []Link{
				{
					Path: "abc/def.hij",
					Range: Range{
						Start: Position{
							Line:      42,
							Character: 0,
						},
						End: Position{
							Line:      42,
							Character: 15,
						},
					},
				},
				{
					Path: "x.y",
					Range: Range{
						Start: Position{
							Line:      42,
							Character: 16,
						},
						End: Position{
							Line:      42,
							Character: 23,
						},
					},
				},
			},
			ReturnsError: false,
		},
	}

	for i, c := range cases {
		links, err := ParseLinksInLine([]byte(c.Line), c.LineIndex)
		if c.ReturnsError {
			assert.NotNil(err, "test case %v", i)
		} else {
			assert.Nil(err, "test case %v", i)
			assert.Equal(c.ExpectedLinks, links, "test case %v", i)
		}
	}
}
