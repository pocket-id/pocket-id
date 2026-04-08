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

	t.Run("pathFromKey strips prefix to honor relative-path contract", func(t *testing.T) {
		tests := []struct {
			name     string
			prefix   string
			key      string
			expected string
		}{
			{name: "no prefix returns key unchanged", prefix: "", key: "images/logo.png", expected: "images/logo.png"},
			{name: "no prefix empty key", prefix: "", key: "", expected: ""},
			{name: "prefix matches and is stripped", prefix: "data/uploads", key: "data/uploads/application-images/logo.svg", expected: "application-images/logo.svg"},
			{name: "single-segment prefix stripped", prefix: "root", key: "root/foo/bar.txt", expected: "foo/bar.txt"},
			{name: "prefix equal to key without trailing slash is unchanged", prefix: "root", key: "root", expected: "root"},
			{name: "key without expected prefix returned unchanged", prefix: "data/uploads", key: "other/path.txt", expected: "other/path.txt"},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				s := &s3Storage{
					bucket: "bucket",
					prefix: tc.prefix,
				}
				assert.Equal(t, tc.expected, s.pathFromKey(tc.key))
			})
		}
	})

	t.Run("pathFromKey is the inverse of buildObjectKey for clean paths", func(t *testing.T) {
		paths := []string{
			"images/logo.png",
			"application-images/logo.svg",
			"oidc-client-images/abc.png",
			"deeply/nested/file.bin",
		}
		prefixes := []string{"", "root", "data/uploads"}

		for _, prefix := range prefixes {
			for _, p := range paths {
				s := &s3Storage{bucket: "bucket", prefix: prefix}
				assert.Equal(t, p, s.pathFromKey(s.buildObjectKey(p)),
					"round-trip failed for prefix=%q path=%q", prefix, p)
			}
		}
	})

	t.Run("isS3NotFound detects expected errors", func(t *testing.T) {
		assert.True(t, isS3NotFound(&smithy.GenericAPIError{Code: "NoSuchKey"}))
		assert.True(t, isS3NotFound(&smithy.GenericAPIError{Code: "NotFound"}))
		assert.True(t, isS3NotFound(&s3types.NoSuchKey{}))
		assert.False(t, isS3NotFound(errors.New("boom")))
	})
}
