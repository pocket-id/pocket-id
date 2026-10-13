package emailverification

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
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

// send godoc
// @Summary Send email verification
// @Description Send an email verification to the currently authenticated user
// @Tags Users
// @Produce json
// @Success 204 "No Content"
// @Router /api/users/me/send-email-verification [post]
func (h *handler) send(c *gin.Context) error {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	err = h.service.Send(c.Request.Context(), dbConfig, authz.PrincipalFrom(c).UserID)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// verify godoc
// @Summary Verify email
// @Description Verify the currently authenticated user's email using a verification token
// @Tags Users
// @Param body body dto.EmailVerificationDto true "Email verification token"
// @Success 204 "No Content"
// @Router /api/users/me/verify-email [post]
func (h *handler) verify(c *gin.Context) error {
	var input dto.EmailVerificationDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	err := h.service.Verify(c.Request.Context(), authz.PrincipalFrom(c).UserID, input.Token)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}

// pendingEmailChange godoc
// @Summary Get pending email change
// @Description Get the email change of the currently authenticated user that still has to be confirmed through the emailed link
// @Tags Users
// @Produce json
// @Success 200 {object} emailChangeDto
// @Success 204 "No pending email change"
// @Router /api/users/me/email-change [get]
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

// cancelEmailChange godoc
// @Summary Cancel pending email change
// @Description Cancel the pending email change of the currently authenticated user, so its verification link stops working
// @Tags Users
// @Success 204 "No Content"
// @Router /api/users/me/email-change [delete]
func (h *handler) cancelEmailChange(c *gin.Context) error {
	err := h.service.CancelEmailChange(c.Request.Context(), authz.PrincipalFrom(c).UserID)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}
