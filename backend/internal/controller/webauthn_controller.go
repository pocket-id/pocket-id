package controller

import (
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"golang.org/x/time/rate"
)

func NewWebauthnController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, rateLimitMiddleware *middleware.RateLimitMiddleware, webauthnService *service.WebAuthnService, appConfigService *service.AppConfigService) {
	wc := &WebauthnController{webAuthnService: webauthnService, appConfigService: appConfigService}
	group.GET("/webauthn/register/start", authMiddleware.WithAdminNotRequired().Add(), wc.beginRegistrationHandler)
	group.POST("/webauthn/register/finish", authMiddleware.WithAdminNotRequired().Add(), wc.verifyRegistrationHandler)

	group.GET("/webauthn/login/start", wc.beginLoginHandler)
	group.POST("/webauthn/login/finish", rateLimitMiddleware.Add(rate.Every(10*time.Second), 5), wc.verifyLoginHandler)

	group.POST("/webauthn/logout", authMiddleware.WithAdminNotRequired().Add(), wc.logoutHandler)

	group.GET("/webauthn/credentials", authMiddleware.WithAdminNotRequired().Add(), wc.listCredentialsHandler)
	group.PATCH("/webauthn/credentials/:id", authMiddleware.WithAdminNotRequired().Add(), wc.updateCredentialHandler)
	group.DELETE("/webauthn/credentials/:id", authMiddleware.WithAdminNotRequired().Add(), wc.deleteCredentialHandler)
}

type WebauthnController struct {
	webAuthnService  *service.WebAuthnService
	appConfigService *service.AppConfigService
}

func (wc *WebauthnController) beginRegistrationHandler(c *gin.Context) {
	userID := c.GetString("userID")
	options, err := wc.webAuthnService.BeginRegistration(userID)
	if err != nil {
		c.Error(err)
		return
	}

	cookie.AddSessionIdCookie(c, int(options.Timeout.Seconds()), options.SessionID)
	c.JSON(http.StatusOK, options.Response)
}

func (wc *WebauthnController) verifyRegistrationHandler(c *gin.Context) {
	sessionID, err := c.Cookie(cookie.SessionIdCookieName)
	if err != nil {
		c.Error(&common.MissingSessionIdError{})
		return
	}

	userID := c.GetString("userID")
	credential, err := wc.webAuthnService.VerifyRegistration(sessionID, userID, c.Request)
	if err != nil {
		c.Error(err)
		return
	}

	var credentialDto dto.WebauthnCredentialDto
	if err := dto.MapStruct(credential, &credentialDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, credentialDto)
}

func (wc *WebauthnController) beginLoginHandler(c *gin.Context) {
	options, err := wc.webAuthnService.BeginLogin()
	if err != nil {
		c.Error(err)
		return
	}

	cookie.AddSessionIdCookie(c, int(options.Timeout.Seconds()), options.SessionID)
	c.JSON(http.StatusOK, options.Response)
}

func (wc *WebauthnController) verifyLoginHandler(c *gin.Context) {
	sessionID, err := c.Cookie(cookie.SessionIdCookieName)
	if err != nil {
		c.Error(&common.MissingSessionIdError{})
		return
	}

	credentialAssertionData, err := protocol.ParseCredentialRequestResponseBody(c.Request.Body)
	if err != nil {
		c.Error(err)
		return
	}

	user, token, err := wc.webAuthnService.VerifyLogin(sessionID, credentialAssertionData, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		c.Error(err)
		return
	}

	maxAge := int(wc.appConfigService.DbConfig.SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddAccessTokenCookie(c, maxAge, token)

	c.JSON(http.StatusOK, userDto)
}

func (wc *WebauthnController) listCredentialsHandler(c *gin.Context) {
	userID := c.GetString("userID")
	credentials, err := wc.webAuthnService.ListCredentials(userID)
	if err != nil {
		c.Error(err)
		return
	}

	var credentialDtos []dto.WebauthnCredentialDto
	if err := dto.MapStructList(credentials, &credentialDtos); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, credentialDtos)
}

func (wc *WebauthnController) deleteCredentialHandler(c *gin.Context) {
	userID := c.GetString("userID")
	credentialID := c.Param("id")

	err := wc.webAuthnService.DeleteCredential(userID, credentialID)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (wc *WebauthnController) updateCredentialHandler(c *gin.Context) {
	userID := c.GetString("userID")
	credentialID := c.Param("id")

	var input dto.WebauthnCredentialUpdateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(err)
		return
	}

	credential, err := wc.webAuthnService.UpdateCredential(userID, credentialID, input.Name)
	if err != nil {
		c.Error(err)
		return
	}

	var credentialDto dto.WebauthnCredentialDto
	if err := dto.MapStruct(credential, &credentialDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, credentialDto)
}

func (wc *WebauthnController) logoutHandler(c *gin.Context) {
	cookie.AddAccessTokenCookie(c, 0, "")
	c.Status(http.StatusNoContent)
}
