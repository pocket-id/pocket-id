package controller

import (
	"net/http"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"golang.org/x/time/rate"
)

const defaultSignupTokenDuration = time.Hour

// NewUserSignupController creates a new controller for user signup and signup token management
// @Summary User signup and signup token management controller
// @Description Initializes all user signup-related API endpoints
// @Tags Users
func NewUserSignupController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, rateLimitMiddleware *middleware.RateLimitMiddleware, userSignUpService *service.UserSignUpService, appConfigService *service.AppConfigService) {
	usc := UserSignupController{
		userSignUpService: userSignUpService,
		appConfigService:  appConfigService,
	}

	group.POST("/signup-tokens", authMiddleware.Add(), usc.createSignupTokenHandler)
	group.GET("/signup-tokens", authMiddleware.Add(), usc.listSignupTokensHandler)
	group.DELETE("/signup-tokens/:id", authMiddleware.Add(), usc.deleteSignupTokenHandler)
	group.POST("/signup", rateLimitMiddleware.Add(rate.Every(1*time.Minute), 10), usc.signupHandler)
	group.POST("/signup/setup", usc.signUpInitialAdmin)

}

type UserSignupController struct {
	userSignUpService *service.UserSignUpService
	appConfigService  *service.AppConfigService
}

// signUpInitialAdmin godoc
// @Summary Sign up initial admin user
// @Description Sign up and generate setup access token for initial admin user
// @Tags Users
// @Accept json
// @Produce json
// @Param body body dto.SignUpDto true "User information"
// @Success 200 {object} dto.UserDto
// @Router /api/signup/setup [post]
func (usc *UserSignupController) signUpInitialAdmin(c *gin.Context) {
	var input dto.SignUpDto
	if err := dto.ShouldBindWithNormalizedJSON(c, &input); err != nil {
		_ = c.Error(err)
		return
	}

	user, token, err := usc.userSignUpService.SignUpInitialAdmin(c.Request.Context(), input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		_ = c.Error(err)
		return
	}

	maxAge := int(usc.appConfigService.GetDbConfig().SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddAccessTokenCookie(c, maxAge, token)

	c.JSON(http.StatusOK, userDto)
}

// createSignupTokenHandler godoc
// @Summary Create signup token
// @Description Create a new signup token that allows user registration
// @Tags Users
// @Accept json
// @Produce json
// @Param token body dto.SignupTokenCreateDto true "Signup token information"
// @Success 201 {object} dto.SignupTokenDto
// @Router /api/signup-tokens [post]
func (usc *UserSignupController) createSignupTokenHandler(c *gin.Context) {
	var input dto.SignupTokenCreateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(err)
		return
	}

	ttl := input.TTL.Duration
	if ttl <= 0 {
		ttl = defaultSignupTokenDuration
	}

	signupToken, err := usc.userSignUpService.CreateSignupToken(c.Request.Context(), ttl, input.UsageLimit, input.UserGroupIDs)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var tokenDto dto.SignupTokenDto
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
// @Success 200 {object} dto.Paginated[dto.SignupTokenDto]
// @Router /api/signup-tokens [get]
func (usc *UserSignupController) listSignupTokensHandler(c *gin.Context) {
	listRequestOptions := utils.ParseListRequestOptions(c)

	tokens, pagination, err := usc.userSignUpService.ListSignupTokens(c.Request.Context(), listRequestOptions)
	if err != nil {
		_ = c.Error(err)
		return
	}

	var tokensDto []dto.SignupTokenDto
	if err := dto.MapStructList(tokens, &tokensDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.SignupTokenDto]{
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
func (usc *UserSignupController) deleteSignupTokenHandler(c *gin.Context) {
	tokenID := c.Param("id")

	err := usc.userSignUpService.DeleteSignupToken(c.Request.Context(), tokenID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// signupWithTokenHandler godoc
// @Summary Sign up
// @Description Create a new user account
// @Tags Users
// @Accept json
// @Produce json
// @Param user body dto.SignUpDto true "User information"
// @Success 201 {object} dto.SignUpDto
// @Router /api/signup [post]
func (usc *UserSignupController) signupHandler(c *gin.Context) {
	var input dto.SignUpDto
	if err := dto.ShouldBindWithNormalizedJSON(c, &input); err != nil {
		_ = c.Error(err)
		return
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	user, accessToken, err := usc.userSignUpService.SignUp(c.Request.Context(), input, ipAddress, userAgent)
	if err != nil {
		_ = c.Error(err)
		return
	}

	maxAge := int(usc.appConfigService.GetDbConfig().SessionDuration.AsDurationMinutes().Seconds())
	cookie.AddAccessTokenCookie(c, maxAge, accessToken)

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, userDto)
}
