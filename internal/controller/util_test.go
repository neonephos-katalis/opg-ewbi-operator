package controller

import (
	"testing"

	"github.com/nbycomp/neonephos-opg-ewbi-operator/api/v1beta1"
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
			labels: map[string]string{
				v1beta1.FederationRelationLabel: string(v1beta1.FederationRelationGuest),
				"other":                         "value",
			},
			expected: true,
		},
		{
			name: "Host label present",
			labels: map[string]string{
				v1beta1.FederationRelationLabel: string(v1beta1.FederationRelationHost),
				"other":                         "value",
			},
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
			labels: map[string]string{
				v1beta1.FederationRelationLabel: "something-else",
				"other":                         "value",
			},
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
			result := IsGuestResource(tt.labels)
			assert.Equal(t, tt.expected, result)
		})
	}
}
