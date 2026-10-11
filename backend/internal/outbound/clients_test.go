package outbound

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

func TestPurposeEnvVarsMatchEnvConfig(t *testing.T) {
	// Collect every env tag so a renamed field or purpose can't silently drop its allowlist
	tags := map[string]bool{}
	schema := reflect.TypeFor[common.EnvConfigSchema]()
	for field := range schema.Fields() {
		tags[strings.Split(field.Tag.Get("env"), ",")[0]] = true
	}

	for _, purpose := range AllPurposes {
		assert.True(t, tags[purpose.EnvVar()], "missing env field for %s", purpose.EnvVar())

		// Every purpose must read its own field
		config := &common.EnvConfigSchema{}
		reflect.ValueOf(config).Elem().FieldByIndex(fieldIndexForTag(t, schema, purpose.EnvVar())).SetString("marker")
		assert.Equal(t, "marker", purpose.allowlist(config))
	}
}

func fieldIndexForTag(t *testing.T, schema reflect.Type, tag string) []int {
	t.Helper()

	for field := range schema.Fields() {
		if strings.Split(field.Tag.Get("env"), ",")[0] == tag {
			return field.Index
		}
	}
	t.Fatalf("no field with env tag %s", tag)
	return nil
}

func TestNewKeepsPurposeAllowlistsSeparate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	clients, err := New(&common.EnvConfigSchema{OutboundAllowedHostsSCIM: "loopback"})
	require.NoError(t, err)

	// Opening the loopback range for SCIM must not open it for any other purpose
	for _, purpose := range AllPurposes {
		err := get(t, clients.Client(purpose), server.URL)
		_, blocked := errors.AsType[*BlockedError](err)
		if purpose == PurposeSCIM {
			require.NoError(t, err)
		} else {
			assert.True(t, blocked, "%s should be blocked, got %v", purpose, err)
		}
	}
}

func TestNewNamesTheInvalidVariable(t *testing.T) {
	_, err := New(&common.EnvConfigSchema{OutboundAllowedHostsBackchannelLogout: "10.0.0.0/99"})
	require.ErrorContains(t, err, "OUTBOUND_ALLOWED_HOSTS_BACKCHANNEL_LOGOUT")
}
