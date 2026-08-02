package webauthn

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

type handler struct {
	service   *Service
	appConfig AppConfigResolver
}

func newHandler(service *Service, appConfig AppConfigResolver) *handler {
	return &handler{service: service, appConfig: appConfig}
}

func (h *handler) beginRegistration(c *gin.Context) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	userID := c.GetString("userID")
	options, err := h.service.BeginRegistration(c.Request.Context(), dbConfig, userID)
	if err != nil {
		return err
	}

	cookie.AddSessionIdCookie(c, int(options.Timeout.Seconds()), options.SessionID)
	c.JSON(http.StatusOK, options.Response)
	return nil
}

func (h *handler) verifyRegistration(c *gin.Context) error {
	sessionID, err := c.Cookie(cookie.SessionIdCookieName)
	if err != nil {
		return apperror.MissingSessionID()
	}

	userID := c.GetString("userID")
	credential, err := h.service.VerifyRegistration(c.Request.Context(), sessionID, userID, c.Request, c.ClientIP())
	if err != nil {
		return err
	}

	var credentialDto dto.WebauthnCredentialDto
	if err := dto.MapStruct(credential, &credentialDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, credentialDto)
	return nil
}

func (h *handler) beginLogin(c *gin.Context) error {
	options, err := h.service.BeginLogin(c.Request.Context())
	if err != nil {
		return err
	}

	cookie.AddSessionIdCookie(c, int(options.Timeout.Seconds()), options.SessionID)
	c.JSON(http.StatusOK, options.Response)
	return nil
}

func (h *handler) verifyLogin(c *gin.Context) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	sessionID, err := c.Cookie(cookie.SessionIdCookieName)
	if err != nil {
		return apperror.MissingSessionID()
	}

	credentialAssertionData, err := protocol.ParseCredentialRequestResponseBody(c.Request.Body)
	if err != nil {
		return apperror.InvalidWebAuthnResponse(err)
	}

	user, token, err := h.service.VerifyLogin(c.Request.Context(), dbConfig, sessionID, credentialAssertionData, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		return err
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		return err
	}

	maxAge := int(dbConfig.SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddAccessTokenCookie(c, maxAge, token)

	c.JSON(http.StatusOK, userDto)
	return nil
}

func (h *handler) listCredentials(c *gin.Context) error {
	userID := c.GetString("userID")
	credentials, err := h.service.ListCredentials(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	var credentialDtos []dto.WebauthnCredentialDto
	if err := dto.MapStructList(credentials, &credentialDtos); err != nil {
		return err
	}

	c.JSON(http.StatusOK, credentialDtos)
	return nil
}

func (h *handler) deleteCredential(c *gin.Context) error {
	userID := c.GetString("userID")
	credentialID := c.Param("id")
	clientIP := c.ClientIP()
	userAgent := c.Request.UserAgent()

	err := h.service.DeleteCredential(c.Request.Context(), userID, credentialID, clientIP, userAgent, userID)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

func (h *handler) updateCredential(c *gin.Context) error {
	userID := c.GetString("userID")
	credentialID := c.Param("id")

	var input dto.WebauthnCredentialUpdateDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	credential, err := h.service.UpdateCredential(c.Request.Context(), userID, credentialID, input.Name)
	if err != nil {
		return err
	}

	var credentialDto dto.WebauthnCredentialDto
	if err := dto.MapStruct(credential, &credentialDto); err != nil {
		return err
	}

	c.JSON(http.StatusOK, credentialDto)
	return nil
}

func (h *handler) logout(c *gin.Context) error {
	cookie.AddAccessTokenCookie(c, 0, "")
	c.Status(http.StatusNoContent)
	return nil
}

func (h *handler) reauthenticate(c *gin.Context) error {
	sessionID, err := c.Cookie(cookie.SessionIdCookieName)
	if err != nil {
		return apperror.MissingSessionID()
	}

	var token string

	// Try to create a reauthentication token with WebAuthn
	credentialAssertionData, err := protocol.ParseCredentialRequestResponseBody(c.Request.Body)
	if err == nil {
		token, err = h.service.CreateReauthenticationTokenWithWebauthn(c.Request.Context(), sessionID, credentialAssertionData)
		if err != nil {
			return err
		}
	} else {
		// If WebAuthn fails, try to create a reauthentication token with the access token
		accessToken, _ := c.Cookie(cookie.AccessTokenCookieName)
		token, err = h.service.CreateReauthenticationTokenWithAccessToken(c.Request.Context(), accessToken)
		if err != nil {
			return err
		}
	}

	cookie.AddReauthenticationTokenCookie(c, token)
	c.Status(http.StatusNoContent)
	return nil
}
