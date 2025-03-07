package controller

import (
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

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

	c.JSON(http.StatusOK, gin.H{
		"data":       clientsDto,
		"pagination": pagination,
	})
}

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

func (oc *OidcController) deleteClientHandler(c *gin.Context) {
	err := oc.oidcService.DeleteClient(c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

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

func (oc *OidcController) createClientSecretHandler(c *gin.Context) {
	secret, err := oc.oidcService.CreateClientSecret(c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"secret": secret})
}

func (oc *OidcController) getClientLogoHandler(c *gin.Context) {
	imagePath, mimeType, err := oc.oidcService.GetClientLogo(c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}

	c.Header("Content-Type", mimeType)
	c.File(imagePath)
}

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

func (oc *OidcController) deleteClientLogoHandler(c *gin.Context) {
	err := oc.oidcService.DeleteClientLogo(c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

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
