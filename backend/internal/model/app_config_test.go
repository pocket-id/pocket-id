package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAppConfigVariable_AsMinutesDuration(t *testing.T) {
	tests := []struct {
		name            string
		value           string
		expected        time.Duration
		expectedSeconds int
	}{
		{
			name:            "valid positive integer",
			value:           "60",
			expected:        60 * time.Minute,
			expectedSeconds: 3600,
		},
		{
			name:            "valid zero integer",
			value:           "0",
			expected:        0,
			expectedSeconds: 0,
		},
		{
			name:            "negative integer",
			value:           "-30",
			expected:        -30 * time.Minute,
			expectedSeconds: -1800,
		},
		{
			name:            "invalid non-integer",
			value:           "not-a-number",
			expected:        0,
			expectedSeconds: 0,
		},
		{
			name:            "empty string",
			value:           "",
			expected:        0,
			expectedSeconds: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configVar := AppConfigVariable{
				Value: tt.value,
			}

			result := configVar.AsDurationMinutes()
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.expectedSeconds, int(result.Seconds()))
		})
	}
}
