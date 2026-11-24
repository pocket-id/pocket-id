package utils

import (
	"strings"
	"testing"

	"github.com/mileusna/useragent"
	"github.com/stretchr/testify/assert"
)

func TestParseUserAgent(t *testing.T) {
	// The parser copies a URL token again at every separator, so it takes seconds for an unbounded header like this one
	userAgent := "http://" + strings.Repeat("a", 50_000) + strings.Repeat(";", 50_000)

	assert.Equal(t, useragent.Parse(userAgent[:512]), ParseUserAgent(userAgent))
}
