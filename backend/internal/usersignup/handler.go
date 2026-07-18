package usersignup

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

const defaultSignupTokenDuration = time.Hour

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler {
	return &handler{service: service}
}

func (h *handler) checkInitialAdminSetupAvailable(c *gin.Context) {
	setupCompleted, err := h.service.IsInitialAdminSetupCompleted(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}

	if setupCompleted {
		_ = c.Error(&common.SetupNotAvailableError{})
		return
	}

	c.Status(http.StatusNoContent)
}

// signUpInitialAdmin godoc
// @Summary Sign up initial admin user
// @Description Sign up and generate setup access token for initial admin user
// @Tags Users
// @Accept json
// @Produce json
// @Param body body signUpDto true "User information"
// @Success 200 {object} dto.UserDto
// @Router /api/signup/setup [post]
func (h *handler) signUpInitialAdmin(c *gin.Context) {
	config, err := appconfig.FromCtx(c.Request.Context())
	if err != nil {
		_ = c.Error(fmt.Errorf("error loading app configuration: %w", err))
		return
	}

	var input signUpDto
	if err := dto.ShouldBindWithNormalizedJSON(c, &input); err != nil {
		_ = c.Error(err)
		return
	}

	user, token, err := h.service.SignUpInitialAdmin(c.Request.Context(), input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		_ = c.Error(err)
		return
	}

	maxAge := int(config.SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddAccessTokenCookie(c, maxAge, token)

	c.JSON(http.StatusOK, userDto)
}

// createSignupTokenHandler godoc
// @Summary Create signup token
// @Description Create a new signup token that allows user registration
// @Tags Users
// @Accept json
// @Produce json
// @Param token body signupTokenCreateDto true "Signup token information"
// @Success 201 {object} signupTokenDto
// @Router /api/signup-tokens [post]
func (h *handler) createSignupToken(c *gin.Context) {
	var input signupTokenCreateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(err)
		return
	}

	ttl := input.TTL.Duration
	if ttl <= 0 {
		ttl = defaultSignupTokenDuration
	}

	signupToken, err := h.service.CreateSignupToken(c.Request.Context(), ttl, input.UsageLimit, input.UserGroupIDs)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var tokenDto signupTokenDto
	err = dto.MapStruct(signupToken, &tokenDto)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, tokenDto)
}

// listSignupTokensHandler godoc
// @Summary List signup tokens
// @Description Get a paginated list of signup tokens
// @Tags Users
// @Param pagination[page] query int false "Page number for pagination" default(1)
// @Param pagination[limit] query int false "Number of items per page" default(20)
// @Param sort[column] query string false "Column to sort by"
// @Param sort[direction] query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[signupTokenDto]
// @Router /api/signup-tokens [get]
func (h *handler) listSignupTokens(c *gin.Context) {
	listRequestOptions := utils.ParseListRequestOptions(c)

	tokens, pagination, err := h.service.ListSignupTokens(c.Request.Context(), listRequestOptions)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var tokensDto []signupTokenDto
	if err := dto.MapStructList(tokens, &tokensDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, dto.Paginated[signupTokenDto]{
		Data:       tokensDto,
		Pagination: pagination,
	})
}

// deleteSignupTokenHandler godoc
// @Summary Delete signup token
// @Description Delete a signup token by ID
// @Tags Users
// @Param id path string true "Token ID"
// @Success 204 "No Content"
// @Router /api/signup-tokens/{id} [delete]
func (h *handler) deleteSignupToken(c *gin.Context) {
	tokenID := c.Param("id")

	err := h.service.DeleteSignupToken(c.Request.Context(), tokenID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// signupHandler godoc
// @Summary Sign up
// @Description Create a new user account
// @Tags Users
// @Accept json
// @Produce json
// @Param user body signUpDto true "User information"
// @Success 201 {object} dto.UserDto
// @Router /api/signup [post]
func (h *handler) signup(c *gin.Context) {
	config, err := appconfig.FromCtx(c.Request.Context())
	if err != nil {
		_ = c.Error(fmt.Errorf("error loading app configuration: %w", err))
		return
	}

	var input signUpDto
	if err := dto.ShouldBindWithNormalizedJSON(c, &input); err != nil {
		_ = c.Error(err)
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	user, accessToken, err := h.service.SignUp(c.Request.Context(), input, ipAddress, userAgent)
	if err != nil {
		_ = c.Error(err)
		return
	}

	maxAge := int(config.SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddAccessTokenCookie(c, maxAge, accessToken)

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, userDto)
}
