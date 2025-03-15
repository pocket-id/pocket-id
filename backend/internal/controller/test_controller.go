package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

func NewTestController(group *gin.RouterGroup, testService *service.TestService) {
	testController := &TestController{TestService: testService}

	group.POST("/test/reset", testController.resetAndSeedHandler)
	group.GET("/test/refresh-token", testController.getTestRefreshTokenHandler)
}

type TestController struct {
	TestService *service.TestService
}

func (tc *TestController) getTestRefreshTokenHandler(c *gin.Context) {
	// Get the clientId from query parameter, default to Nextcloud client
	clientId := c.Query("client_id")
	if clientId == "" {
		clientId = "3654a746-35d4-4321-ac61-0bdcff2b4055" // Nextcloud client ID
	}

	// Get the test refresh token from database
	var refreshToken model.OidcRefreshToken
	if err := tc.TestService.DB().Where("token = ? AND client_id = ?", "test-refresh-token", clientId).First(&refreshToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get test refresh token",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"refresh_token": refreshToken.Token,
		"client_id":     refreshToken.ClientID,
		"scope":         refreshToken.Scope,
	})
}

func (tc *TestController) resetAndSeedHandler(c *gin.Context) {
	if err := tc.TestService.ResetDatabase(); err != nil {
		c.Error(err)
		return
	}

	if err := tc.TestService.ResetApplicationImages(); err != nil {
		c.Error(err)
		return
	}

	if err := tc.TestService.SeedDatabase(); err != nil {
		c.Error(err)
		return
	}

	if err := tc.TestService.ResetAppConfig(); err != nil {
		c.Error(err)
		return
	}

	tc.TestService.SetJWTKeys()

	c.Status(http.StatusNoContent)
}
