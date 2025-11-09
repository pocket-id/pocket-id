package storage

import (
	"errors"
	"testing"

	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/stretchr/testify/assert"
)

func TestS3Helpers(t *testing.T) {
	t.Run("buildObjectKey trims and joins prefix", func(t *testing.T) {
		tests := []struct {
			name     string
			prefix   string
			path     string
			expected string
		}{
			{name: "no prefix no path", prefix: "", path: "", expected: ""},
			{name: "prefix no path", prefix: "root", path: "", expected: "root"},
			{name: "prefix with nested path", prefix: "root", path: "foo/bar/baz", expected: "root/foo/bar/baz"},
			{name: "trimmed path and prefix", prefix: "root", path: "/foo//bar/", expected: "root/foo/bar"},
			{name: "no prefix path only", prefix: "", path: "./images/logo.png", expected: "images/logo.png"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				s := &s3Storage{
					bucket: "bucket",
					prefix: tc.prefix,
				}
				assert.Equal(t, tc.expected, s.buildObjectKey(tc.path))
			})
		}
	})

	t.Run("isS3NotFound detects expected errors", func(t *testing.T) {
		assert.True(t, isS3NotFound(&smithy.GenericAPIError{Code: "NoSuchKey"}))
		assert.True(t, isS3NotFound(&smithy.GenericAPIError{Code: "NotFound"}))
		assert.True(t, isS3NotFound(&s3types.NoSuchKey{}))
		assert.False(t, isS3NotFound(errors.New("boom")))
	})
}
