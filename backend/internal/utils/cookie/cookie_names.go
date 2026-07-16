package cookie

import (
	"strings"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

var AccessTokenCookieName = "__Host-access_token"
var SessionIdCookieName = "__Host-session"
var DeviceTokenCookieName = "__Secure-device_token"                     //nolint:gosec
var DeviceLoginTokenCookieName = "__Secure-device_login_token"          //nolint:gosec
var ReauthenticationTokenCookieName = "__Secure-reauthentication_token" //nolint:gosec

func init() {
	if strings.HasPrefix(common.EnvConfig.AppURL, "http://") {
		AccessTokenCookieName = "access_token"
		SessionIdCookieName = "session"
		DeviceTokenCookieName = "device_token"
		DeviceLoginTokenCookieName = "device_login_token"
		ReauthenticationTokenCookieName = "reauthentication_token"
	}
}
