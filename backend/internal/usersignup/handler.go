package usersignup

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

const defaultSignupTokenDuration = time.Hour

type handler struct {
	service   *Service
	appConfig appconfig.AppConfigResolver
}

func newHandler(service *Service, appConfig appconfig.AppConfigResolver) *handler {
	return &handler{service: service, appConfig: appConfig}
}

func (h *handler) checkInitialAdminSetupAvailable(c *gin.Context) error {
	setupCompleted, err := h.service.IsInitialAdminSetupCompleted(c.Request.Context())
	if err != nil {
		return err
	}

	if setupCompleted {
		return apperror.SetupNotAvailable()
	}

	c.Status(http.StatusNoContent)
	return nil
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
func (h *handler) signUpInitialAdmin(c *gin.Context) error {
	config, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	var input signUpDto
	err = httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	user, token, err := h.service.SignUpInitialAdmin(c.Request.Context(), config, input)
	if err != nil {
		return err
	}

	var userDto dto.UserDto
	err = dto.MapStruct(user, &userDto)
	if err != nil {
		return err
	}

	maxAge := int(config.SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddAccessTokenCookie(c, maxAge, token)

	c.JSON(http.StatusOK, userDto)
	return nil
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
func (h *handler) createSignupToken(c *gin.Context) error {
	var input signupTokenCreateDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}

	ttl := input.TTL.Duration
	if ttl <= 0 {
		ttl = defaultSignupTokenDuration
	}

	signupToken, err := h.service.CreateSignupToken(c.Request.Context(), ttl, input.UsageLimit, input.UserGroupIDs)
	if err != nil {
		return err
	}

	var tokenDto signupTokenDto
	err = dto.MapStruct(signupToken, &tokenDto)
	if err != nil {
		return err
	}

	c.JSON(http.StatusCreated, tokenDto)
	return nil
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
func (h *handler) listSignupTokens(c *gin.Context) error {
	listRequestOptions := utils.ParseListRequestOptions(c)

	tokens, pagination, err := h.service.ListSignupTokens(c.Request.Context(), listRequestOptions)
	if err != nil {
		return err
	}

	var tokensDto []signupTokenDto
	err = dto.MapStructList(tokens, &tokensDto)
	if err != nil {
		return err
	}

	c.JSON(http.StatusOK, dto.Paginated[signupTokenDto]{
		Data:       tokensDto,
		Pagination: pagination,
	})
	return nil
}

// deleteSignupTokenHandler godoc
// @Summary Delete signup token
// @Description Delete a signup token by ID
// @Tags Users
// @Param id path string true "Token ID"
// @Success 204 "No Content"
// @Router /api/signup-tokens/{id} [delete]
func (h *handler) deleteSignupToken(c *gin.Context) error {
	tokenID := c.Param("id")

	err := h.service.DeleteSignupToken(c.Request.Context(), tokenID)
	if err != nil {
		return err
	}

	c.Status(http.StatusNoContent)
	return nil
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
func (h *handler) signup(c *gin.Context) error {
	config, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}

	var input signUpDto
	err = httpserver.BindJSON(c, &input)
	if err != nil {
		return err
	}

	user, accessToken, err := h.service.SignUp(c.Request.Context(), config, input, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		return err
	}

	maxAge := int(config.SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddAccessTokenCookie(c, maxAge, accessToken)

	var userDto dto.UserDto
	err = dto.MapStruct(user, &userDto)
	if err != nil {
		return err
	}

	c.JSON(http.StatusCreated, userDto)
	return nil
}
