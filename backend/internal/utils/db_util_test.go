package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLikeContainsPattern(t *testing.T) {
	assert.Equal(t, "%grafana%", LikeContainsPattern("grafana"))
	assert.Equal(t, `%100\%%`, LikeContainsPattern("100%"))
	assert.Equal(t, `%next\_cloud%`, LikeContainsPattern("next_cloud"))
	assert.Equal(t, `%a\\b%`, LikeContainsPattern(`a\b`))
}
