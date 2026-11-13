package emailverification

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/auditlogs"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// emailChangeDto describes an email change that waits for the user to confirm the new address
type emailChangeDto struct {
	Email     string            `json:"email"`
	ExpiresAt datatype.DateTime `json:"expiresAt"`
}

type handler struct {
	service   *Service
	appConfig appconfig.AppConfigResolver
}

func newHandler(service *Service, appConfig appconfig.AppConfigResolver) *handler {
	return &handler{service: service, appConfig: appConfig}
}

func (h *handler) send(c *gin.Context) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	return h.service.Send(c.Request.Context(), dbConfig, authz.PrincipalFrom(c).UserID)
}

func (h *handler) verify(c *gin.Context, input dto.EmailVerificationDto) error {
	origin := auditlogs.OriginFrom(c)
	return h.service.Verify(c.Request.Context(), origin.Principal.UserID, input.Token, origin)
}

func (h *handler) pendingEmailChange(c *gin.Context) error {
	pending, err := h.service.PendingEmailChange(c.Request.Context(), authz.PrincipalFrom(c).UserID)
	if err != nil {
		return err
	}
	if pending == nil {
		c.Status(http.StatusNoContent)
		return nil
	}

	c.JSON(http.StatusOK, emailChangeDto{
		Email:     pending.Email,
		ExpiresAt: datatype.DateTime(pending.ExpiresAt),
	})
	return nil
}

func (h *handler) cancelEmailChange(c *gin.Context) error {
	return h.service.CancelEmailChange(c.Request.Context(), authz.PrincipalFrom(c).UserID)
}
