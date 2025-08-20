package utils

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONDuration_MarshalJSON(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{time.Minute + 30*time.Second, "1m30s"},
		{0, "0s"},
	}
	for _, tc := range tests {
		d := JSONDuration{Duration: tc.duration}
		b, err := json.Marshal(d)
		require.NoError(t, err)
		assert.Equal(t, `"`+tc.want+`"`, string(b))
	}
}

func TestJSONDuration_UnmarshalJSON_String(t *testing.T) {
	var d JSONDuration
	err := json.Unmarshal([]byte(`"2h15m5s"`), &d)
	require.NoError(t, err)
	want := 2*time.Hour + 15*time.Minute + 5*time.Second
	assert.Equal(t, want, d.Duration)
}

func TestJSONDuration_UnmarshalJSON_NumberSeconds(t *testing.T) {
	tests := []struct {
		json string
		want time.Duration
	}{
		{"0", 0},
		{"1", 1 * time.Second},
		{"2.25", 2 * time.Second}, // Milliseconds are truncated
	}
	for _, tc := range tests {
		var d JSONDuration
		err := json.Unmarshal([]byte(tc.json), &d)
		require.NoError(t, err, "input: %s", tc.json)
		assert.Equal(t, tc.want, d.Duration, "input: %s", tc.json)
	}
}

func TestJSONDuration_UnmarshalJSON_Invalid(t *testing.T) {
	cases := [][]byte{
		[]byte(`true`),
		[]byte(`{}`),
		[]byte(`"not-a-duration"`),
	}
	for _, b := range cases {
		var d JSONDuration
		err := json.Unmarshal(b, &d)
		require.Error(t, err, "input: %s", string(b))
	}
}
