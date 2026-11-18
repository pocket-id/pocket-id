package session

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

var challengeHeaderPattern = regexp.MustCompile(`^"([^"]+)";id="([^"]+)"$`)

type bindingTest struct {
	service   *Service
	handler   *handler
	deviceKey *ecdsa.PrivateKey
	otherKey  *ecdsa.PrivateKey
}

func newBindingTest(t *testing.T) *bindingTest {
	t.Helper()
	gin.SetMode(gin.TestMode)

	service, _ := newServiceForTest(t)
	return &bindingTest{service: service, handler: &handler{service: service}, deviceKey: newDeviceKey(t), otherKey: newDeviceKey(t)}
}

func newDeviceKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	return key
}

// newSession signs a user in and returns the session token and the principal it authenticates as
func (b *bindingTest) newSession(t *testing.T, userID string) (string, *authz.Principal) {
	t.Helper()
	token, err := b.service.Create(t.Context(), userID, "phr", "", "browser", time.Hour, true)
	require.NoError(t, err)
	_, session, err := b.service.Validate(t.Context(), token)
	require.NoError(t, err)
	return token, &authz.Principal{Kind: authz.KindSession, UserID: userID, SessionID: session.ID}
}

func (b *bindingTest) register(t *testing.T, token string, principal *authz.Principal, proof string) *httptest.ResponseRecorder {
	t.Helper()
	return serveBinding(t, b.handler.bind, principal, map[string]string{proofHeader: proof}, token)
}

func (b *bindingTest) refresh(t *testing.T, id, proof, token string) *httptest.ResponseRecorder {
	t.Helper()
	return serveBinding(t, b.handler.refreshBinding, nil, map[string]string{"Sec-Secure-Session-Id": id, proofHeader: proof}, token)
}

func (b *bindingTest) session(t *testing.T, principal *authz.Principal) Session {
	t.Helper()
	session, err := b.service.find(t.Context(), principal.UserID, principal.SessionID)
	require.NoError(t, err)
	return session
}

func serveBinding(t *testing.T, handle func(*gin.Context) error, principal *authz.Principal, headers map[string]string, token string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)
	for name, value := range headers {
		if value != "" {
			c.Request.Header.Set(name, value)
		}
	}
	if token != "" {
		c.Request.AddCookie(&http.Cookie{Name: cookie.SessionTokenCookieName, Value: token, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	}
	if principal != nil {
		authz.SetPrincipal(c, principal)
	}

	require.NoError(t, handle(c))
	c.Writer.WriteHeaderNow()
	return recorder
}

// requireChallenge asserts the response carries a challenge for the session and returns it
func requireChallenge(t *testing.T, recorder *httptest.ResponseRecorder, id string) string {
	t.Helper()
	match := challengeHeaderPattern.FindStringSubmatch(recorder.Header().Get("Secure-Session-Challenge"))
	require.NotNil(t, match, recorder.Header())
	require.Equal(t, id, match[2])
	return match[1]
}

func requireEnded(t *testing.T, recorder *httptest.ResponseRecorder, id string) {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"session_identifier": "`+id+`", "continue": false}`, recorder.Body.String())
}

func TestBind(t *testing.T) {
	b := newBindingTest(t)
	token, principal := b.newSession(t, "user-1")
	challenge := registrationChallenge(token)

	t.Run("rejects a proof over another challenge", func(t *testing.T) {
		recorder := b.register(t, token, principal, testutils.DBSCProof(t, b.deviceKey, "other", b.deviceKey))
		require.Equal(t, http.StatusForbidden, recorder.Code)
	})

	t.Run("rejects a proof signed with another key than it carries", func(t *testing.T) {
		recorder := b.register(t, token, principal, testutils.DBSCProof(t, b.otherKey, challenge, b.deviceKey))
		require.Equal(t, http.StatusForbidden, recorder.Code)
	})

	t.Run("binds the session without requiring a binding cookie yet", func(t *testing.T) {
		recorder := b.register(t, token, principal, testutils.DBSCProof(t, b.deviceKey, challenge, b.deviceKey))
		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

		id := "user-1." + principal.SessionID
		requireChallenge(t, recorder, id)
		require.Empty(t, recorder.Result().Cookies())

		var instructions map[string]any
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &instructions))
		require.Equal(t, id, instructions["session_identifier"])
		require.Equal(t, BindingRefreshPath, instructions["refresh_url"])

		session := b.session(t, principal)
		require.NotEmpty(t, session.BindingKey)
		require.True(t, session.BindingSatisfiedBy(""))
	})

	t.Run("keeps the key of a bound session", func(t *testing.T) {
		boundKey := b.session(t, principal).BindingKey

		recorder := b.register(t, token, principal, testutils.DBSCProof(t, b.otherKey, challenge, b.otherKey))
		require.Equal(t, http.StatusForbidden, recorder.Code)
		require.Equal(t, boundKey, b.session(t, principal).BindingKey)
	})
}

func TestRefreshBinding(t *testing.T) {
	b := newBindingTest(t)
	token, principal := b.newSession(t, "user-1")
	id := "user-1." + principal.SessionID
	recorder := b.register(t, token, principal, testutils.DBSCProof(t, b.deviceKey, registrationChallenge(token), b.deviceKey))
	require.Equal(t, http.StatusOK, recorder.Code)
	challenge := requireChallenge(t, recorder, id)

	t.Run("asks for a proof when none is sent", func(t *testing.T) {
		recorder := b.refresh(t, id, "", "")
		require.Equal(t, http.StatusForbidden, recorder.Code)
		require.Equal(t, challenge, requireChallenge(t, recorder, id))
	})

	t.Run("rejects a proof over a stale challenge", func(t *testing.T) {
		recorder := b.refresh(t, id, testutils.DBSCProof(t, b.deviceKey, "stale", nil), "")
		require.Equal(t, http.StatusForbidden, recorder.Code)
		require.Equal(t, challenge, requireChallenge(t, recorder, id))
	})

	t.Run("verifies with the bound key and ignores a key in the proof", func(t *testing.T) {
		recorder := b.refresh(t, id, testutils.DBSCProof(t, b.otherKey, challenge, b.otherKey), "")
		require.Equal(t, http.StatusForbidden, recorder.Code)
	})

	proof := testutils.DBSCProof(t, b.deviceKey, challenge, nil)
	var nextChallenge string

	t.Run("issues a binding cookie for a valid proof", func(t *testing.T) {
		// Chrome sends both headers bare, but the spec asks for quoted strings
		recorder := b.refresh(t, `"`+id+`"`, `"`+proof+`"`, token)
		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())

		nextChallenge = requireChallenge(t, recorder, id)
		require.NotEqual(t, challenge, nextChallenge)

		cookies := recorder.Result().Cookies()
		require.Len(t, cookies, 1)
		require.Equal(t, cookie.SessionBindingCookieName(principal.SessionID), cookies[0].Name)
		require.Equal(t, 600, cookies[0].MaxAge)

		session := b.session(t, principal)
		require.True(t, session.BindingSatisfiedBy(cookies[0].Value))
		require.False(t, session.BindingSatisfiedBy(""))
	})

	t.Run("rejects a replayed proof", func(t *testing.T) {
		recorder := b.refresh(t, id, proof, "")
		require.Equal(t, http.StatusForbidden, recorder.Code)
		require.Equal(t, nextChallenge, requireChallenge(t, recorder, id))
	})

	t.Run("ends the session of a browser that signed in to another one", func(t *testing.T) {
		otherToken, _ := b.newSession(t, "user-1")
		requireEnded(t, b.refresh(t, id, testutils.DBSCProof(t, b.deviceKey, nextChallenge, nil), otherToken), id)
	})

	t.Run("ends an unknown session", func(t *testing.T) {
		requireEnded(t, b.refresh(t, "user-1.unknown", "", ""), "user-1.unknown")
		requireEnded(t, b.refresh(t, "malformed", "", ""), "malformed")
	})

	t.Run("ends an unbound session", func(t *testing.T) {
		_, unbound := b.newSession(t, "user-1")
		unboundID := "user-1." + unbound.SessionID
		requireEnded(t, b.refresh(t, unboundID, "", ""), unboundID)
	})

	t.Run("ends a revoked session", func(t *testing.T) {
		require.NoError(t, b.service.Revoke(t.Context(), "user-1", principal.SessionID, auditlogs.Origin{}))
		requireEnded(t, b.refresh(t, id, testutils.DBSCProof(t, b.deviceKey, nextChallenge, nil), ""), id)
	})
}
