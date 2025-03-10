package controller

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

// NewOidcController creates a new controller for OIDC related endpoints
// @Summary OIDC controller
// @Description Initializes all OIDC-related API endpoints for authentication and client management
// @Tags OIDC
func NewOidcController(group *gin.RouterGroup, authMiddleware *middleware.AuthMiddleware, fileSizeLimitMiddleware *middleware.FileSizeLimitMiddleware, oidcService *service.OidcService, jwtService *service.JwtService) {
	oc := &OidcController{oidcService: oidcService, jwtService: jwtService}

	group.POST("/oidc/authorize", authMiddleware.WithAdminNotRequired().Add(), oc.authorizeHandler)
	group.POST("/oidc/authorization-required", authMiddleware.WithAdminNotRequired().Add(), oc.authorizationConfirmationRequiredHandler)

	group.POST("/oidc/token", oc.createTokensHandler)
	group.GET("/oidc/userinfo", oc.userInfoHandler)
	group.POST("/oidc/userinfo", oc.userInfoHandler)
	group.POST("/oidc/end-session", authMiddleware.WithSuccessOptional().Add(), oc.EndSessionHandler)
	group.GET("/oidc/end-session", authMiddleware.WithSuccessOptional().Add(), oc.EndSessionHandler)

	group.GET("/oidc/clients", authMiddleware.Add(), oc.listClientsHandler)
	group.POST("/oidc/clients", authMiddleware.Add(), oc.createClientHandler)
	group.GET("/oidc/clients/:id", authMiddleware.Add(), oc.getClientHandler)
	group.GET("/oidc/clients/:id/meta", oc.getClientMetaDataHandler)
	group.PUT("/oidc/clients/:id", authMiddleware.Add(), oc.updateClientHandler)
	group.DELETE("/oidc/clients/:id", authMiddleware.Add(), oc.deleteClientHandler)

	group.PUT("/oidc/clients/:id/allowed-user-groups", authMiddleware.Add(), oc.updateAllowedUserGroupsHandler)
	group.POST("/oidc/clients/:id/secret", authMiddleware.Add(), oc.createClientSecretHandler)

	group.GET("/oidc/clients/:id/logo", oc.getClientLogoHandler)
	group.DELETE("/oidc/clients/:id/logo", oc.deleteClientLogoHandler)
	group.POST("/oidc/clients/:id/logo", authMiddleware.Add(), fileSizeLimitMiddleware.Add(2<<20), oc.updateClientLogoHandler)
}

type OidcController struct {
	oidcService *service.OidcService
	jwtService  *service.JwtService
}

// authorizeHandler godoc
// @Summary Authorize OIDC client
// @Description Start the OIDC authorization process for a client
// @Tags OIDC,Authentication
// @Accept json
// @Produce json
// @Param request body dto.AuthorizeOidcClientRequestDto true "Authorization request parameters"
// @Success 200 {object} dto.AuthorizeOidcClientResponseDto "Authorization code and callback URL"
// @Failure 400 {object} object "Bad request or validation error"
// @Failure 401 {object} object "Unauthorized"
// @Failure 403 {object} object "User not authorized for client"
// @Failure 404 {object} object "Client not found"
// @Failure 500 {object} object "Internal server error"
// @Security BearerAuth
// @Router /oidc/authorize [post]
func (oc *OidcController) authorizeHandler(c *gin.Context) {
	var input dto.AuthorizeOidcClientRequestDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(err)
		return
	}

	code, callbackURL, err := oc.oidcService.Authorize(input, c.GetString("userID"), c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		c.Error(err)
		return
	}

	response := dto.AuthorizeOidcClientResponseDto{
		Code:        code,
		CallbackURL: callbackURL,
	}

	c.JSON(http.StatusOK, response)
}

// authorizationConfirmationRequiredHandler godoc
// @Summary Check if authorization confirmation is required
// @Description Check if the user needs to confirm authorization for the client
// @Tags OIDC,Authentication
// @Accept json
// @Produce json
// @Param request body dto.AuthorizationRequiredDto true "Authorization check parameters"
// @Success 200 {object} object "{ \"authorizationRequired\": true/false }"
// @Failure 400 {object} object "Bad request"
// @Failure 401 {object} object "Unauthorized"
// @Failure 404 {object} object "Client not found"
// @Failure 500 {object} object "Internal server error"
// @Security BearerAuth
// @Router /oidc/authorization-required [post]
func (oc *OidcController) authorizationConfirmationRequiredHandler(c *gin.Context) {
	var input dto.AuthorizationRequiredDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(err)
		return
	}

	hasAuthorizedClient, err := oc.oidcService.HasAuthorizedClient(input.ClientID, c.GetString("userID"), input.Scope)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"authorizationRequired": !hasAuthorizedClient})
}

// createTokensHandler godoc
// @Summary Create OIDC tokens
// @Description Exchange authorization code for ID and access tokens
// @Tags OIDC,Authentication
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param client_id formData string false "Client ID (if not using Basic Auth)"
// @Param client_secret formData string false "Client secret (if not using Basic Auth)"
// @Param code formData string true "Authorization code"
// @Param grant_type formData string true "Grant type (must be 'authorization_code')"
// @Param code_verifier formData string false "PKCE code verifier"
// @Success 200 {object} object "{ \"id_token\": \"string\", \"access_token\": \"string\", \"token_type\": \"Bearer\" }"
// @Failure 400 {object} object "Bad request or invalid code"
// @Failure 401 {object} object "Invalid client credentials"
// @Failure 500 {object} object "Internal server error"
// @Router /oidc/token [post]
func (oc *OidcController) createTokensHandler(c *gin.Context) {
	// Disable cors for this endpoint
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	var input dto.OidcCreateTokensDto

	if err := c.ShouldBind(&input); err != nil {
		c.Error(err)
		return
	}

	clientID := input.ClientID
	clientSecret := input.ClientSecret

	// Client id and secret can also be passed over the Authorization header
	if clientID == "" && clientSecret == "" {
		clientID, clientSecret, _ = c.Request.BasicAuth()
	}

	idToken, accessToken, err := oc.oidcService.CreateTokens(input.Code, input.GrantType, clientID, clientSecret, input.CodeVerifier)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"id_token": idToken, "access_token": accessToken, "token_type": "Bearer"})
}

// userInfoHandler godoc
// @Summary Get user information
// @Description Get user information based on the access token
// @Tags OIDC,Authentication
// @Accept json
// @Produce json
// @Success 200 {object} object "User claims based on requested scopes"
// @Failure 400 {object} object "Missing access token"
// @Failure 401 {object} object "Invalid access token"
// @Failure 500 {object} object "Internal server error"
// @Security OAuth2AccessToken
// @Router /oidc/userinfo [get]
func (oc *OidcController) userInfoHandler(c *gin.Context) {
	authHeaderSplit := strings.Split(c.GetHeader("Authorization"), " ")
	if len(authHeaderSplit) != 2 {
		c.Error(&common.MissingAccessToken{})
		return
	}

	token := authHeaderSplit[1]

	jwtClaims, err := oc.jwtService.VerifyOauthAccessToken(token)
	if err != nil {
		c.Error(err)
		return
	}
	userID := jwtClaims.Subject
	clientId := jwtClaims.Audience[0]
	claims, err := oc.oidcService.GetUserClaimsForClient(userID, clientId)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, claims)
}

// userInfoHandler godoc (POST method)
// @Summary Get user information (POST method)
// @Description Get user information based on the access token using POST
// @Tags OIDC,Authentication
// @Accept json
// @Produce json
// @Success 200 {object} object "User claims based on requested scopes"
// @Failure 400 {object} object "Missing access token"
// @Failure 401 {object} object "Invalid access token"
// @Failure 500 {object} object "Internal server error"
// @Security OAuth2AccessToken
// @Router /oidc/userinfo [post]
func (oc *OidcController) userInfoHandlerPost(c *gin.Context) {
	// Implementation is the same as GET
}

// EndSessionHandler godoc
// @Summary End OIDC session
// @Description End user session and handle OIDC logout
// @Tags OIDC,Authentication
// @Accept application/x-www-form-urlencoded
// @Produce html
// @Param id_token_hint query string false "ID token"
// @Param post_logout_redirect_uri query string false "URL to redirect to after logout"
// @Param state query string false "State parameter to include in the redirect"
// @Success 302 "Redirect to post-logout URL or application logout page"
// @Failure 400 {object} object "Bad request"
// @Failure 500 {object} object "Internal server error"
// @Router /oidc/end-session [get]
func (oc *OidcController) EndSessionHandler(c *gin.Context) {
	var input dto.OidcLogoutDto

	// Bind query parameters to the struct
	if c.Request.Method == http.MethodGet {
		if err := c.ShouldBindQuery(&input); err != nil {
			c.Error(err)
			return
		}
	} else if c.Request.Method == http.MethodPost {
		// Bind form parameters to the struct
		if err := c.ShouldBind(&input); err != nil {
			c.Error(err)
			return
		}
	}

	callbackURL, err := oc.oidcService.ValidateEndSession(input, c.GetString("userID"))
	if err != nil {
		// If the validation fails, the user has to confirm the logout manually and doesn't get redirected
		log.Printf("Error getting logout callback URL, the user has to confirm the logout manually: %v", err)
		c.Redirect(http.StatusFound, common.EnvConfig.AppURL+"/logout")
		return
	}

	// The validation was successful, so we can log out and redirect the user to the callback URL without confirmation
	cookie.AddAccessTokenCookie(c, 0, "")

	logoutCallbackURL, _ := url.Parse(callbackURL)
	if input.State != "" {
		q := logoutCallbackURL.Query()
		q.Set("state", input.State)
		logoutCallbackURL.RawQuery = q.Encode()
	}

	c.Redirect(http.StatusFound, logoutCallbackURL.String())
}

// EndSessionHandler godoc (POST method)
// @Summary End OIDC session (POST method)
// @Description End user session and handle OIDC logout using POST
// @Tags OIDC,Authentication
// @Accept application/x-www-form-urlencoded
// @Produce html
// @Param id_token_hint formData string false "ID token"
// @Param post_logout_redirect_uri formData string false "URL to redirect to after logout"
// @Param state formData string false "State parameter to include in the redirect"
// @Success 302 "Redirect to post-logout URL or application logout page"
// @Failure 400 {object} object "Bad request"
// @Failure 500 {object} object "Internal server error"
// @Router /oidc/end-session [post]
func (oc *OidcController) EndSessionHandlerPost(c *gin.Context) {
	// Implementation is the same as GET
}

// getClientMetaDataHandler godoc
// @Summary Get client metadata
// @Description Get OIDC client metadata for discovery and configuration
// @Tags OIDC,Clients
// @Produce json
// @Param id path string true "Client ID"
// @Success 200 {object} dto.OidcClientMetaDataDto "Client metadata"
// @Failure 404 {object} object "Client not found"
// @Failure 500 {object} object "Internal server error"
// @Router /oidc/clients/{id}/meta [get]
func (oc *OidcController) getClientMetaDataHandler(c *gin.Context) {
	clientId := c.Param("id")
	client, err := oc.oidcService.GetClient(clientId)
	if err != nil {
		c.Error(err)
		return
	}

	clientDto := dto.OidcClientMetaDataDto{}
	err = dto.MapStruct(client, &clientDto)
	if err == nil {
		c.JSON(http.StatusOK, clientDto)
		return
	}

	c.Error(err)
}

// getClientHandler godoc
// @Summary Get OIDC client
// @Description Get detailed information about an OIDC client
// @Tags OIDC,Clients
// @Produce json
// @Param id path string true "Client ID"
// @Success 200 {object} dto.OidcClientWithAllowedUserGroupsDto "Client information"
// @Failure 401 {object} object "Unauthorized"
// @Failure 403 {object} object "Forbidden"
// @Failure 404 {object} object "Client not found"
// @Failure 500 {object} object "Internal server error"
// @Security BearerAuth
// @Router /oidc/clients/{id} [get]
func (oc *OidcController) getClientHandler(c *gin.Context) {
	clientId := c.Param("id")
	client, err := oc.oidcService.GetClient(clientId)
	if err != nil {
		c.Error(err)
		return
	}

	clientDto := dto.OidcClientWithAllowedUserGroupsDto{}
	err = dto.MapStruct(client, &clientDto)
	if err == nil {
		c.JSON(http.StatusOK, clientDto)
		return
	}

	c.Error(err)
}

// listClientsHandler godoc
// @Summary List OIDC clients
// @Description Get a paginated list of OIDC clients with optional search and sorting
// @Tags OIDC
// @Param search query string false "Search term to filter clients by name"
// @Param page query int false "Page number, starting from 1" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Param sort_column query string false "Column to sort by" default("name")
// @Param sort_direction query string false "Sort direction (asc or desc)" default("asc")
// @Success 200 {object} dto.Paginated[dto.OidcClientDto]
// @Security BearerAuth
// @Router /oidc/clients [get]
func (oc *OidcController) listClientsHandler(c *gin.Context) {
	searchTerm := c.Query("search")
	var sortedPaginationRequest utils.SortedPaginationRequest
	if err := c.ShouldBindQuery(&sortedPaginationRequest); err != nil {
		c.Error(err)
		return
	}

	clients, pagination, err := oc.oidcService.ListClients(searchTerm, sortedPaginationRequest)
	if err != nil {
		c.Error(err)
		return
	}

	var clientsDto []dto.OidcClientDto
	if err := dto.MapStructList(clients, &clientsDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, dto.Paginated[dto.OidcClientDto]{
		Data:       clientsDto,
		Pagination: pagination,
	})
}

// createClientHandler godoc
// @Summary Create OIDC client
// @Description Create a new OIDC client
// @Tags OIDC,Clients
// @Accept json
// @Produce json
// @Param client body dto.OidcClientCreateDto true "Client information"
// @Success 201 {object} dto.OidcClientWithAllowedUserGroupsDto "Created client"
// @Failure 400 {object} object "Bad request or validation error"
// @Failure 401 {object} object "Unauthorized"
// @Failure 403 {object} object "Forbidden"
// @Failure 409 {object} object "Conflict - client ID already exists"
// @Failure 500 {object} object "Internal server error"
// @Security BearerAuth
// @Router /oidc/clients [post]
func (oc *OidcController) createClientHandler(c *gin.Context) {
	var input dto.OidcClientCreateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(err)
		return
	}

	client, err := oc.oidcService.CreateClient(input, c.GetString("userID"))
	if err != nil {
		c.Error(err)
		return
	}

	var clientDto dto.OidcClientWithAllowedUserGroupsDto
	if err := dto.MapStruct(client, &clientDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, clientDto)
}

// deleteClientHandler godoc
// @Summary Delete OIDC client
// @Description Delete an OIDC client by ID
// @Tags OIDC,Clients
// @Param id path string true "Client ID"
// @Success 204 "No Content"
// @Failure 401 {object} object "Unauthorized"
// @Failure 403 {object} object "Forbidden"
// @Failure 404 {object} object "Client not found"
// @Failure 500 {object} object "Internal server error"
// @Security BearerAuth
// @Router /oidc/clients/{id} [delete]
func (oc *OidcController) deleteClientHandler(c *gin.Context) {
	err := oc.oidcService.DeleteClient(c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// updateClientHandler godoc
// @Summary Update OIDC client
// @Description Update an existing OIDC client
// @Tags OIDC,Clients
// @Accept json
// @Produce json
// @Param id path string true "Client ID"
// @Param client body dto.OidcClientCreateDto true "Client information"
// @Success 200 {object} dto.OidcClientWithAllowedUserGroupsDto "Updated client"
// @Failure 400 {object} object "Bad request or validation error"
// @Failure 401 {object} object "Unauthorized"
// @Failure 403 {object} object "Forbidden"
// @Failure 404 {object} object "Client not found"
// @Failure 500 {object} object "Internal server error"
// @Security BearerAuth
// @Router /oidc/clients/{id} [put]
func (oc *OidcController) updateClientHandler(c *gin.Context) {
	var input dto.OidcClientCreateDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(err)
		return
	}

	client, err := oc.oidcService.UpdateClient(c.Param("id"), input)
	if err != nil {
		c.Error(err)
		return
	}

	var clientDto dto.OidcClientWithAllowedUserGroupsDto
	if err := dto.MapStruct(client, &clientDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, clientDto)
}

// createClientSecretHandler godoc
// @Summary Create client secret
// @Description Generate a new secret for an OIDC client
// @Tags OIDC,Clients
// @Produce json
// @Param id path string true "Client ID"
// @Success 200 {object} object "{ \"secret\": \"string\" }"
// @Failure 401 {object} object "Unauthorized"
// @Failure 403 {object} object "Forbidden"
// @Failure 404 {object} object "Client not found"
// @Failure 500 {object} object "Internal server error"
// @Security BearerAuth
// @Router /oidc/clients/{id}/secret [post]
func (oc *OidcController) createClientSecretHandler(c *gin.Context) {
	secret, err := oc.oidcService.CreateClientSecret(c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"secret": secret})
}

// getClientLogoHandler godoc
// @Summary Get client logo
// @Description Get the logo image for an OIDC client
// @Tags OIDC,Clients
// @Produce image/png
// @Produce image/jpeg
// @Produce image/svg+xml
// @Param id path string true "Client ID"
// @Success 200 {file} binary "Logo image"
// @Failure 404 {object} object "Client or logo not found"
// @Failure 500 {object} object "Internal server error"
// @Router /oidc/clients/{id}/logo [get]
func (oc *OidcController) getClientLogoHandler(c *gin.Context) {
	imagePath, mimeType, err := oc.oidcService.GetClientLogo(c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}

	c.Header("Content-Type", mimeType)
	c.File(imagePath)
}

// updateClientLogoHandler godoc
// @Summary Update client logo
// @Description Upload or update the logo for an OIDC client
// @Tags OIDC,Clients
// @Accept multipart/form-data
// @Param id path string true "Client ID"
// @Param file formData file true "Logo image file (PNG, JPG, or SVG, max 2MB)"
// @Success 204 "No Content"
// @Failure 400 {object} object "Bad request or invalid file"
// @Failure 401 {object} object "Unauthorized"
// @Failure 403 {object} object "Forbidden"
// @Failure 404 {object} object "Client not found"
// @Failure 413 {object} object "File too large (max 2MB)"
// @Failure 500 {object} object "Internal server error"
// @Security BearerAuth
// @Router /oidc/clients/{id}/logo [post]
func (oc *OidcController) updateClientLogoHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.Error(err)
		return
	}

	err = oc.oidcService.UpdateClientLogo(c.Param("id"), file)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// deleteClientLogoHandler godoc
// @Summary Delete client logo
// @Description Delete the logo for an OIDC client
// @Tags OIDC,Clients
// @Param id path string true "Client ID"
// @Success 204 "No Content"
// @Failure 401 {object} object "Unauthorized"
// @Failure 403 {object} object "Forbidden"
// @Failure 404 {object} object "Client or logo not found"
// @Failure 500 {object} object "Internal server error"
// @Security BearerAuth
// @Router /oidc/clients/{id}/logo [delete]
func (oc *OidcController) deleteClientLogoHandler(c *gin.Context) {
	err := oc.oidcService.DeleteClientLogo(c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// updateAllowedUserGroupsHandler godoc
// @Summary Update allowed user groups
// @Description Update the user groups allowed to access an OIDC client
// @Tags OIDC,Clients,User Groups
// @Accept json
// @Produce json
// @Param id path string true "Client ID"
// @Param groups body dto.OidcUpdateAllowedUserGroupsDto true "User group IDs"
// @Success 200 {object} dto.OidcClientDto "Updated client"
// @Failure 400 {object} object "Bad request"
// @Failure 401 {object} object "Unauthorized"
// @Failure 403 {object} object "Forbidden"
// @Failure 404 {object} object "Client not found"
// @Failure 500 {object} object "Internal server error"
// @Security BearerAuth
// @Router /oidc/clients/{id}/allowed-user-groups [put]
func (oc *OidcController) updateAllowedUserGroupsHandler(c *gin.Context) {
	var input dto.OidcUpdateAllowedUserGroupsDto
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(err)
		return
	}

	oidcClient, err := oc.oidcService.UpdateAllowedUserGroups(c.Param("id"), input)
	if err != nil {
		c.Error(err)
		return
	}

	var oidcClientDto dto.OidcClientDto
	if err := dto.MapStruct(oidcClient, &oidcClientDto); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, oidcClientDto)
}
