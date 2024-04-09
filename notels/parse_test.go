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
	}{
		{
			Line:          "",
			LineIndex:     0,
			ExpectedLinks: nil,
		},
		{
			Line:          "[[]",
			LineIndex:     0,
			ExpectedLinks: nil,
		},
		// skip empty links
		{
			Line:          "[[]]",
			LineIndex:     0,
			ExpectedLinks: nil,
		},
		{
			Line:          "[[abc",
			LineIndex:     0,
			ExpectedLinks: nil,
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
		},
	}

	for i, c := range cases {
		links := ParseLinksInLine([]byte(c.Line), c.LineIndex)
		assert.Equal(c.ExpectedLinks, links, "test case %v", i)
	}
}
