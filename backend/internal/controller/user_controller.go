package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"golang.org/x/time/rate"
)

func NewUserController(group *gin.RouterGroup, jwtAuthMiddleware *middleware.JwtAuthMiddleware, rateLimitMiddleware *middleware.RateLimitMiddleware, userService *service.UserService, appConfigService *service.AppConfigService) {
	uc := UserController{
		userService:      userService,
		appConfigService: appConfigService,
	}

	group.GET("/users", jwtAuthMiddleware.Add(true), uc.listUsersHandler)
	group.GET("/users/me", jwtAuthMiddleware.Add(false), uc.getCurrentUserHandler)
	group.GET("/users/:id", jwtAuthMiddleware.Add(true), uc.getUserHandler)
	group.POST("/users", jwtAuthMiddleware.Add(true), uc.createUserHandler)
	group.PUT("/users/:id", jwtAuthMiddleware.Add(true), uc.updateUserHandler)
	group.PUT("/users/me", jwtAuthMiddleware.Add(false), uc.updateCurrentUserHandler)
	group.DELETE("/users/:id", jwtAuthMiddleware.Add(true), uc.deleteUserHandler)

	group.GET("/users/:id/profile-picture.png", uc.getUserProfilePictureHandler)
	group.GET("/users/me/profile-picture.png", jwtAuthMiddleware.Add(false), uc.getCurrentUserProfilePictureHandler)
	group.PUT("/users/:id/profile-picture", jwtAuthMiddleware.Add(true), uc.updateUserProfilePictureHandler)
	group.PUT("/users/me/profile-picture", jwtAuthMiddleware.Add(false), uc.updateUserProfilePictureHandler)

	group.POST("/users/:id/one-time-access-token", jwtAuthMiddleware.Add(true), uc.createOneTimeAccessTokenHandler)
	group.POST("/one-time-access-token/:token", rateLimitMiddleware.Add(rate.Every(10*time.Second), 5), uc.exchangeOneTimeAccessTokenHandler)
	group.POST("/one-time-access-token/setup", uc.getSetupAccessTokenHandler)
	group.POST("/one-time-access-email", rateLimitMiddleware.Add(rate.Every(10*time.Minute), 3), uc.requestOneTimeAccessEmailHandler)
}

type UserController struct {
	userService      *service.UserService
	appConfigService *service.AppConfigService
}

func (uc *UserController) listUsersHandler(c *gin.Context) {
	searchTerm := c.Query("search")
	var sortedPaginationRequest utils.SortedPaginationRequest
	if err := c.ShouldBindQuery(&sortedPaginationRequest); err != nil {
		c.Error(err)
		return
	}

	users, pagination, err := uc.userService.ListUsers(searchTerm, sortedPaginationRequest)
	if err != nil {
		c.Error(err)
		return
	}

	var usersDto []dto.UserDto
	if err := dto.MapStructList(users, &usersDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       usersDto,
		"pagination": pagination,
	})
}

func (uc *UserController) getUserHandler(c *gin.Context) {
	user, err := uc.userService.GetUser(c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, userDto)
}

func (uc *UserController) getCurrentUserHandler(c *gin.Context) {
	user, err := uc.userService.GetUser(c.GetString("userID"))
	if err != nil {
		c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, userDto)
}

func (uc *UserController) deleteUserHandler(c *gin.Context) {
	if err := uc.userService.DeleteUser(c.Param("id")); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (uc *UserController) createUserHandler(c *gin.Context) {
	var input dto.UserCreateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(err)
		return
	}

	user, err := uc.userService.CreateUser(input)
	if err != nil {
		c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, userDto)
}

func (uc *UserController) updateUserHandler(c *gin.Context) {
	uc.updateUser(c, false)
}

func (uc *UserController) updateCurrentUserHandler(c *gin.Context) {
	if uc.appConfigService.DbConfig.AllowOwnAccountEdit.Value != "true" {
		c.Error(&common.AccountEditNotAllowedError{})
		return
	}
	uc.updateUser(c, true)
}

func (uc *UserController) getUserProfilePictureHandler(c *gin.Context) {
	userID := c.Param("id")

	picture, size, err := uc.userService.GetProfilePicture(userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.DataFromReader(http.StatusOK, size, "image/png", picture, nil)
}

func (uc *UserController) getCurrentUserProfilePictureHandler(c *gin.Context) {
	userID := c.GetString("userID")

	picture, size, err := uc.userService.GetProfilePicture(userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.DataFromReader(http.StatusOK, size, "image/png", picture, nil)
}

func (uc *UserController) updateUserProfilePictureHandler(c *gin.Context) {
	userID := c.GetString("userID")
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.Error(err)
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.Error(err)
		return
	}
	defer file.Close()

	if err := uc.userService.UpdateProfilePicture(userID, file); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (uc *UserController) updateCurrentUserProfilePictureHandler(c *gin.Context) {
	userID := c.GetString("userID")
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.Error(err)
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.Error(err)
		return
	}
	defer file.Close()

	if err := uc.userService.UpdateProfilePicture(userID, file); err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (uc *UserController) createOneTimeAccessTokenHandler(c *gin.Context) {
	var input dto.OneTimeAccessTokenCreateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(err)
		return
	}

	token, err := uc.userService.CreateOneTimeAccessToken(input.UserID, input.ExpiresAt)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": token})
}

func (uc *UserController) requestOneTimeAccessEmailHandler(c *gin.Context) {
	var input dto.OneTimeAccessEmailDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(err)
		return
	}

	err := uc.userService.RequestOneTimeAccessEmail(input.Email, input.RedirectPath)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (uc *UserController) exchangeOneTimeAccessTokenHandler(c *gin.Context) {
	user, token, err := uc.userService.ExchangeOneTimeAccessToken(c.Param("token"), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		c.Error(err)
		return
	}

	sessionDurationInMinutesParsed, _ := strconv.Atoi(uc.appConfigService.DbConfig.SessionDuration.Value)
	maxAge := sessionDurationInMinutesParsed * 60
	cookie.AddAccessTokenCookie(c, maxAge, token)

	c.JSON(http.StatusOK, userDto)
}

func (uc *UserController) getSetupAccessTokenHandler(c *gin.Context) {
	user, token, err := uc.userService.SetupInitialAdmin()
	if err != nil {
		c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		c.Error(err)
		return
	}

	sessionDurationInMinutesParsed, _ := strconv.Atoi(uc.appConfigService.DbConfig.SessionDuration.Value)
	maxAge := sessionDurationInMinutesParsed * 60
	cookie.AddAccessTokenCookie(c, maxAge, token)

	c.JSON(http.StatusOK, userDto)
}

func (uc *UserController) updateUser(c *gin.Context, updateOwnUser bool) {
	var input dto.UserCreateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(err)
		return
	}

	var userID string
	if updateOwnUser {
		userID = c.GetString("userID")
	} else {
		userID = c.Param("id")
	}

	user, err := uc.userService.UpdateUser(userID, input, updateOwnUser, false)
	if err != nil {
		c.Error(err)
		return
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, userDto)
}
