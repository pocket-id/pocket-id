package controller

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

// RecoveryCodeController manages emergency recovery codes for a user.
type RecoveryCodeController struct {
	recoveryCodeService *service.RecoveryCodeService
	appConfigService    *service.AppConfigService
}

// NewRecoveryCodeController wires up the recovery code endpoints.
// @Summary Recovery code controller
// @Description Initializes endpoints for managing and redeeming recovery codes
// @Tags Recovery Codes
func NewRecoveryCodeController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, rateLimitMiddleware *middleware.RateLimitMiddleware, recoveryCodeService *service.RecoveryCodeService, appConfigService *service.AppConfigService) {
	rc := &RecoveryCodeController{
		recoveryCodeService: recoveryCodeService,
		appConfigService:    appConfigService,
	}

	group.POST("/users/me/recovery-codes", authMiddleware.WithAdminNotRequired().Add(), rc.generateHandler)
	group.GET("/users/me/recovery-codes", authMiddleware.WithAdminNotRequired().Add(), rc.statusHandler)
	group.DELETE("/users/me/recovery-codes", authMiddleware.WithAdminNotRequired().Add(), rc.revokeHandler)
	group.POST("/recovery-code", rateLimitMiddleware.Add(rate.Every(10*time.Second), 5), rc.redeemHandler)
}

// generateHandler godoc
// @Summary Generate recovery codes
// @Description Generate a fresh batch of recovery codes for the current user. Any previously
// @Description issued codes for the user are invalidated.
// @Tags Recovery Codes
// @Success 201 {object} dto.RecoveryCodeGenerateResponseDto
// @Router /api/users/me/recovery-codes [post]
func (c *RecoveryCodeController) generateHandler(ctx *gin.Context) {
	userID := ctx.GetString("userID")

	codes, err := c.recoveryCodeService.GenerateForUser(ctx.Request.Context(), userID, ctx.ClientIP(), ctx.Request.UserAgent())
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, dto.RecoveryCodeGenerateResponseDto{Codes: codes})
}

// statusHandler godoc
// @Summary Recovery code status
// @Description Return the total and unused recovery code count for the current user.
// @Tags Recovery Codes
// @Success 200 {object} dto.RecoveryCodeStatusDto
// @Router /api/users/me/recovery-codes [get]
func (c *RecoveryCodeController) statusHandler(ctx *gin.Context) {
	userID := ctx.GetString("userID")

	total, unused, err := c.recoveryCodeService.StatusForUser(ctx.Request.Context(), userID)
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, dto.RecoveryCodeStatusDto{Total: total, Unused: unused})
}

// revokeHandler godoc
// @Summary Revoke recovery codes
// @Description Delete every recovery code that currently belongs to the user.
// @Tags Recovery Codes
// @Success 204 "No Content"
// @Router /api/users/me/recovery-codes [delete]
func (c *RecoveryCodeController) revokeHandler(ctx *gin.Context) {
	userID := ctx.GetString("userID")

	if err := c.recoveryCodeService.RevokeAllForUser(ctx.Request.Context(), userID, ctx.ClientIP(), ctx.Request.UserAgent()); err != nil {
		_ = ctx.Error(err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

type redeemRecoveryCodeInput struct {
	Code string `json:"code" binding:"required"`
}

// redeemHandler godoc
// @Summary Redeem recovery code
// @Description Exchange an emergency recovery code for a session.
// @Tags Recovery Codes
// @Param code body redeemRecoveryCodeInput true "Recovery code"
// @Success 200 {object} dto.UserDto
// @Router /api/recovery-code [post]
func (c *RecoveryCodeController) redeemHandler(ctx *gin.Context) {
	var input redeemRecoveryCodeInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		_ = ctx.Error(err)
		return
	}

	user, token, err := c.recoveryCodeService.Redeem(ctx.Request.Context(), input.Code, ctx.ClientIP(), ctx.Request.UserAgent())
	if err != nil {
		_ = ctx.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		_ = ctx.Error(err)
		return
	}

	maxAge := int(c.appConfigService.GetDbConfig().SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddAccessTokenCookie(ctx, maxAge, token)

	ctx.JSON(http.StatusOK, userDto)
}
