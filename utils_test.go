package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPrefix(t *testing.T) {
	assert := assert.New(t)
	testCases := []struct {
		input  string
		output string
	}{
		{
			input:  "a_b_c",
			output: "a_b",
		},
		{
			input:  "c",
			output: "c",
		},
		{
			input:  "a_c",
			output: "a_c",
		},
	}

	for _, tc := range testCases {
		r := getPrefix(tc.input)
		assert.Equal(tc.output, r)
	}
}
