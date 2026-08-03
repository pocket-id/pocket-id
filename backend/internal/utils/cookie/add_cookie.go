package cookie

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func AddAccessTokenCookie(c *gin.Context, maxAgeInSeconds int, token string) {
	addCookie(c, AccessTokenCookieName, token, maxAgeInSeconds, "/")
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
