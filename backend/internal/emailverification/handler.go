package emailverification

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
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
func (h *handler) send(c *gin.Context) {
	dbConfig, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		_ = c.Error(fmt.Errorf("error loading app configuration: %w", err))
		return
	}

	err = h.service.Send(c.Request.Context(), dbConfig, c.GetString("userID"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// verify godoc
// @Summary Verify email
// @Description Verify the currently authenticated user's email using a verification token
// @Tags Users
// @Param body body dto.EmailVerificationDto true "Email verification token"
// @Success 204 "No Content"
// @Router /api/users/me/verify-email [post]
func (h *handler) verify(c *gin.Context) {
	var input dto.EmailVerificationDto
	if err := dto.ShouldBindWithNormalizedJSON(c, &input); err != nil {
		_ = c.Error(err)
		return
	}

	err := h.service.Verify(c.Request.Context(), c.GetString("userID"), input.Token)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
