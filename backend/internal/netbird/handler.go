package netbird

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

type handler struct {
	service   *Service
	appConfig appconfig.AppConfigResolver
}

func newHandler(service *Service, appConfig appconfig.AppConfigResolver) *handler {
	return &handler{
		service:   service,
		appConfig: appConfig,
	}
}

// getIdentity godoc
// @Summary Get NetBird identity
// @Description Get the user that owns the NetBird peer the request came from, when it was routed through the NetBird proxy
// @Tags NetBird
// @Produce json
// @Success 200 {object} identityDto "User behind the NetBird peer"
// @Router /api/netbird/identity [get]
func (h *handler) getIdentity(c *gin.Context) error {
	user, err := h.service.Identify(c.Request.Context(), c.GetHeader(ProxySecretHeader), c.GetHeader(UserIDHeader))
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, identityDto{
		ID:       user.ID,
		Name:     user.FullName(),
		Username: user.Username,
	})
	return nil
}

// login godoc
// @Summary Sign in via NetBird
// @Description Create a session for the user that owns the NetBird peer the request came from
// @Tags NetBird
// @Accept json
// @Produce json
// @Param request body loginDto true "User the browser confirmed"
// @Success 200 {object} dto.UserDto "Signed-in user"
// @Router /api/netbird/login [post]
func (h *handler) login(c *gin.Context) error {
	var input loginDto
	err := httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	browserToken, _ := c.Cookie(cookie.KnownBrowserCookieName)
	user, tokens, err := h.service.Login(c.Request.Context(), dbConfig, c.GetHeader(ProxySecretHeader), c.GetHeader(UserIDHeader), input.UserID, c.ClientIP(), c.Request.UserAgent(), browserToken)
	if err != nil {
		return err
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		return err
	}

	cookie.AddSessionTokenCookie(c, int(dbConfig.SessionDuration.AsDurationMinutes().Seconds()), tokens.SessionToken)
	if tokens.KnownBrowserToken != "" {
		cookie.AddKnownBrowserCookie(c, tokens.KnownBrowserToken, int(auditlogs.KnownBrowserLifetime.Seconds()))
	}
	c.JSON(http.StatusOK, userDto)
	return nil
}
