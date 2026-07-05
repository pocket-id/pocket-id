package webauthn

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

type handler struct {
	service   *Service
	appConfig AppConfigProvider
}

func newHandler(service *Service, appConfig AppConfigProvider) *handler {
	return &handler{service: service, appConfig: appConfig}
}

func (h *handler) beginRegistration(c *gin.Context) {
	userID := c.GetString("userID")
	options, err := h.service.BeginRegistration(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	cookie.AddSessionIdCookie(c, int(options.Timeout.Seconds()), options.SessionID)
	c.JSON(http.StatusOK, options.Response)
}

func (h *handler) verifyRegistration(c *gin.Context) {
	sessionID, err := c.Cookie(cookie.SessionIdCookieName)
	if err != nil {
		_ = c.Error(&common.MissingSessionIdError{})
		return
	}

	userID := c.GetString("userID")
	credential, err := h.service.VerifyRegistration(c.Request.Context(), sessionID, userID, c.Request, c.ClientIP())
	if err != nil {
		_ = c.Error(err)
		return
	}

	var credentialDto dto.WebauthnCredentialDto
	if err := dto.MapStruct(credential, &credentialDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, credentialDto)
}

func (h *handler) beginLogin(c *gin.Context) {
	options, err := h.service.BeginLogin(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}

	cookie.AddSessionIdCookie(c, int(options.Timeout.Seconds()), options.SessionID)
	c.JSON(http.StatusOK, options.Response)
}

func (h *handler) verifyLogin(c *gin.Context) {
	sessionID, err := c.Cookie(cookie.SessionIdCookieName)
	if err != nil {
		_ = c.Error(&common.MissingSessionIdError{})
		return
	}

	credentialAssertionData, err := protocol.ParseCredentialRequestResponseBody(c.Request.Body)
	if err != nil {
		_ = c.Error(err)
		return
	}

	user, token, err := h.service.VerifyLogin(c.Request.Context(), sessionID, credentialAssertionData, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		_ = c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		_ = c.Error(err)
		return
	}

	maxAge := int(h.appConfig.GetDbConfig().SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddAccessTokenCookie(c, maxAge, token)

	c.JSON(http.StatusOK, userDto)
}

func (h *handler) listCredentials(c *gin.Context) {
	userID := c.GetString("userID")
	credentials, err := h.service.ListCredentials(c.Request.Context(), userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var credentialDtos []dto.WebauthnCredentialDto
	if err := dto.MapStructList(credentials, &credentialDtos); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, credentialDtos)
}

func (h *handler) deleteCredential(c *gin.Context) {
	userID := c.GetString("userID")
	credentialID := c.Param("id")
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()

	err := h.service.DeleteCredential(c.Request.Context(), userID, credentialID, clientIP, userAgent, userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *handler) updateCredential(c *gin.Context) {
	userID := c.GetString("userID")
	credentialID := c.Param("id")

	var input dto.WebauthnCredentialUpdateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(err)
		return
	}

	credential, err := h.service.UpdateCredential(c.Request.Context(), userID, credentialID, input.Name)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var credentialDto dto.WebauthnCredentialDto
	if err := dto.MapStruct(credential, &credentialDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, credentialDto)
}

func (h *handler) logout(c *gin.Context) {
	cookie.AddAccessTokenCookie(c, 0, "")
	c.Status(http.StatusNoContent)
}

func (h *handler) reauthenticate(c *gin.Context) {
	sessionID, err := c.Cookie(cookie.SessionIdCookieName)
	if err != nil {
		_ = c.Error(&common.MissingSessionIdError{})
		return
	}

	var token string

	// Try to create a reauthentication token with WebAuthn
	credentialAssertionData, err := protocol.ParseCredentialRequestResponseBody(c.Request.Body)
	if err == nil {
		token, err = h.service.CreateReauthenticationTokenWithWebauthn(c.Request.Context(), sessionID, credentialAssertionData)
		if err != nil {
			_ = c.Error(err)
			return
		}
	} else {
		// If WebAuthn fails, try to create a reauthentication token with the access token
		accessToken, _ := c.Cookie(cookie.AccessTokenCookieName)
		token, err = h.service.CreateReauthenticationTokenWithAccessToken(c.Request.Context(), accessToken, c.ClientIP(), c.Request.UserAgent())
		if err != nil {
			_ = c.Error(err)
			return
		}
	}

	cookie.AddReauthenticationTokenCookie(c, token)
	c.Status(http.StatusNoContent)
}
