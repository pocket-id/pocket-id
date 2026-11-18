package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lestrrat-go/jwx/v4/jwa"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jws"
	"github.com/lestrrat-go/jwx/v4/jwt"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

// Device Bound Session Credentials (DBSC) bind a session to a key that cannot leave the browser's device, see https://w3c.github.io/webappsec-dbsc/
// A bound session additionally requires a short-lived binding cookie, which only the browser holding the device key can renew, so a stolen session token stops working once the cookie expires
// Registration stores the key without setting a binding cookie, so the browser refreshes right away and a session only starts requiring the cookie after a refresh succeeded
// A browser without DBSC, or one whose refresh never succeeds, keeps an unbound session that works like before

// BindingRefreshPath is the endpoint browsers renew binding cookies at
const BindingRefreshPath = "/api/sessions/binding/refresh"

const (
	bindingRegistrationPath = "/api/sessions/current/binding"

	// bindingCookieLifetime is how long a stolen session token keeps working after the browser holding the device key last refreshed its binding cookie
	// Chrome refreshes two minutes before the cookie expires and signs at most six times in nine minutes per site, so a much shorter lifetime would run out of signatures
	bindingCookieLifetime = 10 * time.Minute

	proofHeader = "Secure-Session-Response"
)

// AnnounceBinding asks a browser that supports DBSC to bind the session the token belongs to to a device key
// Browsers without DBSC ignore the header
func AnnounceBinding(c *gin.Context, token string) {
	c.Header("Secure-Session-Registration", fmt.Sprintf(`(ES256 RS256);path="%s";challenge="%s"`, bindingRegistrationPath, registrationChallenge(token)))
}

// registrationChallenge derives the challenge of a session's registration from its token, so it needs no storage and is useless once the session is bound
func registrationChallenge(token string) string {
	return utils.CreateSha256Hash("dbsc-registration." + token)
}

// BindingSatisfiedBy reports whether a request with the binding cookie value may use the session, which every request may until the session's first refresh
func (s Session) BindingSatisfiedBy(value string) bool {
	if s.BindingSecret == "" {
		return true
	}

	_, err := jwt.ParseString(value,
		jwt.WithKey(jwa.HS256(), []byte(s.BindingSecret)),
		jwt.WithValidate(true),
		jwt.WithSubject(s.ID),
		jwt.WithRequiredClaim(jwt.ExpirationKey),
	)
	return err == nil
}

// newBindingCookie creates the value of a binding cookie for the session
// Each value stays valid until it expires, so requests still carrying the previous one keep working while the browser refreshes
func (s Session) newBindingCookie() (string, error) {
	token, err := jwt.NewBuilder().Subject(s.ID).Expiration(time.Now().Add(bindingCookieLifetime)).Build()
	if err != nil {
		return "", fmt.Errorf("failed to build binding cookie: %w", err)
	}
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256(), []byte(s.BindingSecret)))
	if err != nil {
		return "", fmt.Errorf("failed to sign binding cookie: %w", err)
	}
	return string(signed), nil
}

// bind stores the device key a browser created after AnnounceBinding
func (h *handler) bind(c *gin.Context) error {
	principal := authz.PrincipalFrom(c)
	token, _ := c.Cookie(cookie.SessionTokenCookieName)

	// The proof carries its own key and has to sign the challenge AnnounceBinding derived from the session token
	key, challenge, err := verifyProof(c.GetHeader(proofHeader), nil)
	if err != nil || challenge != registrationChallenge(token) {
		c.Status(http.StatusForbidden)
		return nil //nolint:nilerr // An invalid proof is the browser's error, which a 403 tells it about
	}
	keyJSON, err := json.Marshal(key)
	if err != nil {
		return fmt.Errorf("failed to encode device key: %w", err)
	}

	nextChallenge, bound, err := h.service.Bind(c.Request.Context(), principal.UserID, principal.SessionID, keyJSON)
	if err != nil {
		return err
	}
	if !bound {
		c.Status(http.StatusForbidden)
		return nil
	}

	// The challenge for the first refresh comes along, so the browser signs it without asking for one first
	id := principal.UserID + "." + principal.SessionID
	setChallenge(c, id, nextChallenge)
	c.JSON(http.StatusOK, bindingInstructions(id, principal.SessionID))
	return nil
}

// refreshBinding renews the binding cookie for a browser that proves it still holds the session's device key
// It trusts no cookie, because a refresh that a relying party's redirect triggers carries none of Pocket ID's SameSite=Lax cookies
// It never answers with 401 or a redirect, because Chrome then drops the session for good
func (h *handler) refreshBinding(c *gin.Context) error {
	ctx := c.Request.Context()

	// Chrome sends the session identifier bare although the spec asks for a structured-field string, so quotes are optional
	id := strings.Trim(c.GetHeader("Sec-Secure-Session-Id"), `"`)
	userID, sessionID, _ := strings.Cut(id, ".")
	if userID == "" || sessionID == "" {
		endBinding(c, id)
		return nil
	}

	// A browser that signed in again holds the token of another session, so the session it still refreshes is stale
	token, err := c.Cookie(cookie.SessionTokenCookieName)
	if err == nil {
		tokenUserID, tokenSessionID, _, ok := parseToken(token)
		if ok && (tokenUserID != userID || tokenSessionID != sessionID) {
			endBinding(c, id)
			return nil
		}
	}

	// A session that ended or was never bound has nothing to refresh
	session, err := h.service.find(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	if len(session.BindingKey) == 0 {
		endBinding(c, id)
		return nil
	}

	// A missing or stale proof is answered with the current challenge, which the browser signs and retries with
	key, err := jwk.ParseKey(session.BindingKey)
	if err != nil {
		return fmt.Errorf("failed to parse device key: %w", err)
	}
	_, challenge, err := verifyProof(c.GetHeader(proofHeader), key)
	if err != nil || challenge != session.BindingChallenge {
		requestProof(c, id, session.BindingChallenge)
		return nil //nolint:nilerr // An invalid proof is the browser's error, which a 403 tells it about
	}

	// Moving to the next challenge only succeeds once per challenge, so a captured proof can't be replayed
	session, refreshed, err := h.service.RefreshBinding(ctx, userID, sessionID, challenge)
	if err != nil {
		return err
	}
	if session.ID == "" {
		endBinding(c, id)
		return nil
	}
	if !refreshed {
		requestProof(c, id, session.BindingChallenge)
		return nil
	}

	// Issue the binding cookie along with the challenge for the next refresh
	value, err := session.newBindingCookie()
	if err != nil {
		return err
	}
	cookie.AddSessionBindingCookie(c, sessionID, value, int(bindingCookieLifetime.Seconds()))
	setChallenge(c, id, session.BindingChallenge)
	c.JSON(http.StatusOK, bindingInstructions(id, sessionID))
	return nil
}

// verifyProof verifies the JWT a browser signs with its device key, and returns the key it was verified with and the challenge it signed
// A registration proof carries its key in its header, while a refresh proof must be verified with the key the session is bound to, so key is nil only for a registration
func verifyProof(header string, key jwk.Key) (jwk.Key, string, error) {
	// Chrome sends the JWT bare although the spec asks for a structured-field string, so quotes are optional
	raw := []byte(strings.Trim(header, `"`))
	msg, err := jws.Parse(raw)
	if err != nil {
		return nil, "", err
	}
	if len(msg.Signatures()) != 1 {
		return nil, "", errors.New("proof must have exactly one signature")
	}

	// Only the algorithms AnnounceBinding offers are accepted
	headers := msg.Signatures()[0].ProtectedHeaders()
	typ, _ := headers.Type()
	alg, _ := headers.Algorithm()
	if typ != "dbsc+jwt" || (alg.String() != jwa.ES256().String() && alg.String() != jwa.RS256().String()) {
		return nil, "", errors.New("proof is not a DBSC JWT")
	}

	if key == nil {
		headerKey, ok := headers.JWK()
		if !ok {
			return nil, "", errors.New("registration proof has no key")
		}
		key, err = jwk.PublicKeyOf(headerKey)
		if err != nil {
			return nil, "", err
		}
	}

	payload, err := jws.Verify(raw, jws.WithKey(alg, key))
	if err != nil {
		return nil, "", err
	}
	var claims struct {
		JTI string `json:"jti"`
	}
	err = json.Unmarshal(payload, &claims)
	if err != nil {
		return nil, "", err
	}
	return key, claims.JTI, nil
}

// bindingInstructions tells the browser how to keep a bound session alive
func bindingInstructions(id, sessionID string) gin.H {
	return gin.H{
		"session_identifier": id,
		"refresh_url":        BindingRefreshPath,
		"scope":              gin.H{"include_site": false},
		// Chrome rejects the session if the attributes differ from the cookie's or contain its lifetime
		"credentials": []gin.H{{"type": "cookie", "name": cookie.SessionBindingCookieName(sessionID), "attributes": cookie.RootCookieAttributes}},
		// A relying party's redirect to the authorization endpoint has to trigger a refresh too, or it arrives without a binding cookie
		"allowed_refresh_initiators": []string{"*"},
	}
}

// setChallenge gives the browser the challenge to sign on the next refresh of the session, which Chrome ignores without the session identifier
func setChallenge(c *gin.Context, id, challenge string) {
	c.Header("Secure-Session-Challenge", fmt.Sprintf(`"%s";id="%s"`, challenge, id))
}

// requestProof asks the browser to retry the refresh with a proof over the challenge
func requestProof(c *gin.Context, id, challenge string) {
	setChallenge(c, id, challenge)
	c.Status(http.StatusForbidden)
}

// endBinding makes the browser drop the session, which Chrome only accepts with the session identifier
func endBinding(c *gin.Context, id string) {
	c.JSON(http.StatusOK, gin.H{"session_identifier": id, "continue": false})
}
