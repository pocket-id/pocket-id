package cookie

import (
	"strings"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

var KnownBrowserCookieName = "__Host-known_browser"
var SessionTokenCookieName = "__Host-session_token" // #nosec G101 -- cookie name, not a credential
var SessionIdCookieName = "__Host-session"
var DeviceTokenCookieName = "__Secure-device_token"                     // #nosec G101 -- cookie name, not a credential
var DeviceLoginTokenCookieName = "__Secure-device_login_token"          // #nosec G101 -- cookie name, not a credential
var ReauthenticationTokenCookieName = "__Secure-reauthentication_token" // #nosec G101 -- cookie name, not a credential

func init() {
	if strings.HasPrefix(common.EnvConfig.AppURL, "http://") {
		KnownBrowserCookieName = "known_browser"
		SessionTokenCookieName = "session_token"
		SessionIdCookieName = "session"
		DeviceTokenCookieName = "device_token"
		DeviceLoginTokenCookieName = "device_login_token"
		ReauthenticationTokenCookieName = "reauthentication_token"
	}
}
