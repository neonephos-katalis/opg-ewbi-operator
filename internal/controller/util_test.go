package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsGuestResource(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected bool
	}{
		{
			name: "Guest label present and correct",

			expected: true,
		},
		{
			name: "Host label present",

			expected: false,
		},
		{
			name: "Relation label missing",
			labels: map[string]string{
				"other": "value",
			},
			expected: false,
		},
		{
			name: "Relation label present but wrong value",

			expected: false,
		},
		{
			name:     "Nil labels map",
			labels:   nil,
			expected: false,
		},
		{
			name:     "Empty labels map",
			labels:   map[string]string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGuestResource(tt.name)
			assert.Equal(t, tt.expected, result)
		})
	}
}
