package scimsync

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServiceProviderResponseOmitsToken(t *testing.T) {
	output, err := serviceProviderDto(ServiceProvider{Endpoint: "https://scim.example.com", Token: "stored-token"})
	require.NoError(t, err)

	body, err := json.Marshal(output)
	require.NoError(t, err)
	require.NotContains(t, string(body), "stored-token")
	require.Contains(t, string(body), `"hasToken":true`)
}
