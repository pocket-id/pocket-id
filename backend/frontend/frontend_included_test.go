//go:build !exclude_frontend

package frontend

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestIsSPARequest(t *testing.T) {
	distFS := fstest.MapFS{
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('test')")},
	}

	t.Run("root path is spa request", func(t *testing.T) {
		assert.True(t, isSPARequest("", distFS))
	})

	t.Run("existing bundled asset is not spa request", func(t *testing.T) {
		assert.False(t, isSPARequest("assets/app.js", distFS))
	})

	t.Run("unknown path is spa request", func(t *testing.T) {
		assert.True(t, isSPARequest("authorize", distFS))
	})
}
