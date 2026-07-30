package emailverification

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

type handler struct {
	service   *Service
	appConfig AppConfigResolver
}

func newHandler(service *Service, appConfig AppConfigResolver) *handler {
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

	err = h.service.Send(c.Request.Context(), dbConfig, c.GetString("userID"))
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

	err := h.service.Verify(c.Request.Context(), c.GetString("userID"), input.Token)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
}
