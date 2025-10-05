package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

type ScimController struct {
	scimService *service.ScimService
}

// NewScimController creates a new controller for SCIM 2.0 endpoints
// @Summary SCIM 2.0 controller
// @Description Initializes all SCIM 2.0 API endpoints for user and group provisioning
// @Tags SCIM
func NewScimController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, scimService *service.ScimService) {
	sc := &ScimController{scimService: scimService}

	// SCIM endpoints - require authentication
	scimGroup := group.Group("/scim/v2", authMiddleware.Add())
	
	// User endpoints
	scimGroup.GET("/Users", sc.listUsersHandler)
	scimGroup.GET("/Users/:id", sc.getUserHandler)
	scimGroup.POST("/Users", sc.createUserHandler)
	scimGroup.PUT("/Users/:id", sc.updateUserHandler)
	scimGroup.PATCH("/Users/:id", sc.patchUserHandler)
	scimGroup.DELETE("/Users/:id", sc.deleteUserHandler)

	// Group endpoints
	scimGroup.GET("/Groups", sc.listGroupsHandler)
	scimGroup.GET("/Groups/:id", sc.getGroupHandler)
	scimGroup.POST("/Groups", sc.createGroupHandler)
	scimGroup.PUT("/Groups/:id", sc.updateGroupHandler)
	scimGroup.PATCH("/Groups/:id", sc.patchGroupHandler)
	scimGroup.DELETE("/Groups/:id", sc.deleteGroupHandler)

	// Discovery endpoints
	scimGroup.GET("/ServiceProviderConfig", sc.getServiceProviderConfigHandler)
	scimGroup.GET("/ResourceTypes", sc.getResourceTypesHandler)
	scimGroup.GET("/Schemas", sc.getSchemasHandler)
}

// listUsersHandler godoc
// @Summary List SCIM users
// @Description Get a paginated list of users in SCIM format
// @Tags SCIM
// @Produce json
// @Param startIndex query int false "Start index for pagination (1-based)" default(1)
// @Param count query int false "Number of results per page" default(100)
// @Param filter query string false "Filter expression (e.g., userName eq \"example\")"
// @Success 200 {object} dto.ScimListResponse
// @Failure 401 {object} dto.ScimError
// @Router /api/scim/v2/Users [get]
func (sc *ScimController) listUsersHandler(c *gin.Context) {
	startIndex := 1
	if si := c.Query("startIndex"); si != "" {
		if parsed, err := strconv.Atoi(si); err == nil && parsed > 0 {
			startIndex = parsed
		}
	}

	count := 100
	if cnt := c.Query("count"); cnt != "" {
		if parsed, err := strconv.Atoi(cnt); err == nil && parsed > 0 {
			count = parsed
		}
	}

	filter := c.Query("filter")

	response, err := sc.scimService.ListUsers(c.Request.Context(), startIndex, count, filter)
	if err != nil {
		sc.scimError(c, http.StatusInternalServerError, "Internal server error")
		return
	}

	c.JSON(http.StatusOK, response)
}

// getUserHandler godoc
// @Summary Get SCIM user
// @Description Get a user by ID in SCIM format
// @Tags SCIM
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} dto.ScimUser
// @Failure 404 {object} dto.ScimError
// @Router /api/scim/v2/Users/{id} [get]
func (sc *ScimController) getUserHandler(c *gin.Context) {
	id := c.Param("id")

	user, err := sc.scimService.GetUser(c.Request.Context(), id)
	if err != nil {
		sc.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// createUserHandler godoc
// @Summary Create SCIM user
// @Description Create a new user from SCIM data
// @Tags SCIM
// @Accept json
// @Produce json
// @Param user body dto.ScimUser true "SCIM user object"
// @Success 201 {object} dto.ScimUser
// @Failure 400 {object} dto.ScimError
// @Router /api/scim/v2/Users [post]
func (sc *ScimController) createUserHandler(c *gin.Context) {
	var scimUser dto.ScimUser
	if err := c.ShouldBindJSON(&scimUser); err != nil {
		sc.scimError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := sc.scimService.CreateUser(c.Request.Context(), &scimUser)
	if err != nil {
		sc.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, user)
}

// updateUserHandler godoc
// @Summary Update SCIM user
// @Description Replace an existing user (PUT - full replacement)
// @Tags SCIM
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param user body dto.ScimUser true "SCIM user object"
// @Success 200 {object} dto.ScimUser
// @Failure 404 {object} dto.ScimError
// @Router /api/scim/v2/Users/{id} [put]
func (sc *ScimController) updateUserHandler(c *gin.Context) {
	id := c.Param("id")

	var scimUser dto.ScimUser
	if err := c.ShouldBindJSON(&scimUser); err != nil {
		sc.scimError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := sc.scimService.UpdateUser(c.Request.Context(), id, &scimUser)
	if err != nil {
		sc.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// patchUserHandler godoc
// @Summary Patch SCIM user
// @Description Apply partial updates to a user (PATCH)
// @Tags SCIM
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param patch body dto.ScimPatchOp true "SCIM patch operations"
// @Success 200 {object} dto.ScimUser
// @Failure 404 {object} dto.ScimError
// @Router /api/scim/v2/Users/{id} [patch]
func (sc *ScimController) patchUserHandler(c *gin.Context) {
	id := c.Param("id")

	var patchOp dto.ScimPatchOp
	if err := c.ShouldBindJSON(&patchOp); err != nil {
		sc.scimError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := sc.scimService.PatchUser(c.Request.Context(), id, &patchOp)
	if err != nil {
		sc.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// deleteUserHandler godoc
// @Summary Delete SCIM user
// @Description Delete a user (soft delete - disables the user)
// @Tags SCIM
// @Param id path string true "User ID"
// @Success 204 "No Content"
// @Failure 404 {object} dto.ScimError
// @Router /api/scim/v2/Users/{id} [delete]
func (sc *ScimController) deleteUserHandler(c *gin.Context) {
	id := c.Param("id")

	err := sc.scimService.DeleteUser(c.Request.Context(), id)
	if err != nil {
		sc.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// listGroupsHandler godoc
// @Summary List SCIM groups
// @Description Get a paginated list of groups in SCIM format
// @Tags SCIM
// @Produce json
// @Param startIndex query int false "Start index for pagination (1-based)" default(1)
// @Param count query int false "Number of results per page" default(100)
// @Param filter query string false "Filter expression (e.g., displayName eq \"example\")"
// @Success 200 {object} dto.ScimListResponse
// @Failure 401 {object} dto.ScimError
// @Router /api/scim/v2/Groups [get]
func (sc *ScimController) listGroupsHandler(c *gin.Context) {
	startIndex := 1
	if si := c.Query("startIndex"); si != "" {
		if parsed, err := strconv.Atoi(si); err == nil && parsed > 0 {
			startIndex = parsed
		}
	}

	count := 100
	if cnt := c.Query("count"); cnt != "" {
		if parsed, err := strconv.Atoi(cnt); err == nil && parsed > 0 {
			count = parsed
		}
	}

	filter := c.Query("filter")

	response, err := sc.scimService.ListGroups(c.Request.Context(), startIndex, count, filter)
	if err != nil {
		sc.scimError(c, http.StatusInternalServerError, "Internal server error")
		return
	}

	c.JSON(http.StatusOK, response)
}

// getGroupHandler godoc
// @Summary Get SCIM group
// @Description Get a group by ID in SCIM format
// @Tags SCIM
// @Produce json
// @Param id path string true "Group ID"
// @Success 200 {object} dto.ScimGroup
// @Failure 404 {object} dto.ScimError
// @Router /api/scim/v2/Groups/{id} [get]
func (sc *ScimController) getGroupHandler(c *gin.Context) {
	id := c.Param("id")

	group, err := sc.scimService.GetGroup(c.Request.Context(), id)
	if err != nil {
		sc.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, group)
}

// createGroupHandler godoc
// @Summary Create SCIM group
// @Description Create a new group from SCIM data
// @Tags SCIM
// @Accept json
// @Produce json
// @Param group body dto.ScimGroup true "SCIM group object"
// @Success 201 {object} dto.ScimGroup
// @Failure 400 {object} dto.ScimError
// @Router /api/scim/v2/Groups [post]
func (sc *ScimController) createGroupHandler(c *gin.Context) {
	var scimGroup dto.ScimGroup
	if err := c.ShouldBindJSON(&scimGroup); err != nil {
		sc.scimError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	group, err := sc.scimService.CreateGroup(c.Request.Context(), &scimGroup)
	if err != nil {
		sc.handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, group)
}

// updateGroupHandler godoc
// @Summary Update SCIM group
// @Description Replace an existing group (PUT - full replacement)
// @Tags SCIM
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param group body dto.ScimGroup true "SCIM group object"
// @Success 200 {object} dto.ScimGroup
// @Failure 404 {object} dto.ScimError
// @Router /api/scim/v2/Groups/{id} [put]
func (sc *ScimController) updateGroupHandler(c *gin.Context) {
	id := c.Param("id")

	var scimGroup dto.ScimGroup
	if err := c.ShouldBindJSON(&scimGroup); err != nil {
		sc.scimError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	group, err := sc.scimService.UpdateGroup(c.Request.Context(), id, &scimGroup)
	if err != nil {
		sc.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, group)
}

// patchGroupHandler godoc
// @Summary Patch SCIM group
// @Description Apply partial updates to a group (PATCH)
// @Tags SCIM
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param patch body dto.ScimPatchOp true "SCIM patch operations"
// @Success 200 {object} dto.ScimGroup
// @Failure 404 {object} dto.ScimError
// @Router /api/scim/v2/Groups/{id} [patch]
func (sc *ScimController) patchGroupHandler(c *gin.Context) {
	id := c.Param("id")

	var patchOp dto.ScimPatchOp
	if err := c.ShouldBindJSON(&patchOp); err != nil {
		sc.scimError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	group, err := sc.scimService.PatchGroup(c.Request.Context(), id, &patchOp)
	if err != nil {
		sc.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, group)
}

// deleteGroupHandler godoc
// @Summary Delete SCIM group
// @Description Delete a group
// @Tags SCIM
// @Param id path string true "Group ID"
// @Success 204 "No Content"
// @Failure 404 {object} dto.ScimError
// @Router /api/scim/v2/Groups/{id} [delete]
func (sc *ScimController) deleteGroupHandler(c *gin.Context) {
	id := c.Param("id")

	err := sc.scimService.DeleteGroup(c.Request.Context(), id)
	if err != nil {
		sc.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// getServiceProviderConfigHandler godoc
// @Summary Get SCIM service provider configuration
// @Description Get the SCIM 2.0 service provider configuration
// @Tags SCIM
// @Produce json
// @Success 200 {object} dto.ScimServiceProviderConfig
// @Router /api/scim/v2/ServiceProviderConfig [get]
func (sc *ScimController) getServiceProviderConfigHandler(c *gin.Context) {
	config := sc.scimService.GetServiceProviderConfig()
	c.JSON(http.StatusOK, config)
}

// getResourceTypesHandler godoc
// @Summary Get SCIM resource types
// @Description Get the supported SCIM resource types
// @Tags SCIM
// @Produce json
// @Success 200 {object} dto.ScimListResponse
// @Router /api/scim/v2/ResourceTypes [get]
func (sc *ScimController) getResourceTypesHandler(c *gin.Context) {
	resourceTypes := sc.scimService.GetResourceTypes()
	response := dto.ScimListResponse{
		Schemas:      []string{dto.ScimSchemaListResponse},
		TotalResults: len(resourceTypes),
		StartIndex:   1,
		ItemsPerPage: len(resourceTypes),
		Resources:    resourceTypes,
	}
	c.JSON(http.StatusOK, response)
}

// getSchemasHandler godoc
// @Summary Get SCIM schemas
// @Description Get the supported SCIM schemas
// @Tags SCIM
// @Produce json
// @Success 200 {object} dto.ScimListResponse
// @Router /api/scim/v2/Schemas [get]
func (sc *ScimController) getSchemasHandler(c *gin.Context) {
	schemas := sc.scimService.GetSchemas()
	response := dto.ScimListResponse{
		Schemas:      []string{dto.ScimSchemaListResponse},
		TotalResults: len(schemas),
		StartIndex:   1,
		ItemsPerPage: len(schemas),
		Resources:    schemas,
	}
	c.JSON(http.StatusOK, response)
}

// Helper functions

func (sc *ScimController) scimError(c *gin.Context, status int, detail string) {
	c.JSON(status, dto.ScimError{
		Schemas: []string{dto.ScimSchemaError},
		Status:  strconv.Itoa(status),
		Detail:  detail,
	})
}

func (sc *ScimController) handleError(c *gin.Context, err error) {
	// Use the standard error handler for consistent error handling
	_ = c.Error(err)
}
