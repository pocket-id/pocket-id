package cookie

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func AddSessionTokenCookie(c *gin.Context, maxAgeInSeconds int, token string) {
	addCookie(c, SessionTokenCookieName, token, maxAgeInSeconds, "/")
}

// RememberSignIn reports whether a sign-in should outlast the browser, which it does unless the request has remember=false
func RememberSignIn(c *gin.Context) bool {
	return c.Query("remember") != "false"
}

// AddSignInSessionTokenCookie sets the session cookie of a sign-in
// A sign-in that is not remembered gets a cookie without Max-Age, which the browser drops when it closes
func AddSignInSessionTokenCookie(c *gin.Context, sessionDuration time.Duration, remember bool, token string) {
	maxAge := 0
	if remember {
		maxAge = int(sessionDuration.Seconds())
	}
	AddSessionTokenCookie(c, maxAge, token)
}

func AddSessionIdCookie(c *gin.Context, maxAgeInSeconds int, sessionID string) {
	addCookie(c, SessionIdCookieName, sessionID, maxAgeInSeconds, "/")
}

func AddDeviceTokenCookie(c *gin.Context, deviceToken string) {
	addCookie(c, DeviceTokenCookieName, deviceToken, int(15*time.Minute.Seconds()), "/api/one-time-access-token")
}

func AddDeviceLoginTokenCookie(c *gin.Context, requestID, deviceToken string) {
	path := "/api/device-login/requests/" + requestID + "/exchange"
	addCookie(c, DeviceLoginTokenCookieName, deviceToken, int(15*time.Minute.Seconds()), path)
}

func AddReauthenticationTokenCookie(c *gin.Context, reauthenticationToken string) {
	addCookie(c, ReauthenticationTokenCookieName, reauthenticationToken, int(3*time.Minute.Seconds()), "/")
}

func addCookie(c *gin.Context, name, value string, maxAge int, path string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, value, maxAge, path, "", true, true)
}

func AddKnownBrowserCookie(c *gin.Context, token string, maxAge int) {
	addCookie(c, KnownBrowserCookieName, token, maxAge, "/")
}
