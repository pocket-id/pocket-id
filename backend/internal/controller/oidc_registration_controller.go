package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ory/fosite"

	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/service"
)

// NewOidcRegistrationController registers the OIDC Dynamic Client Registration
// (RFC 7591) and Client Configuration (RFC 7592) HTTP endpoints. These routes are
// intentionally registered without the admin/browser auth middleware: registration
// is constrained by the redirect URI allowlist, which is empty by default and so
// denies every registration until an administrator opts in, and the client
// configuration endpoints self-authenticate via the registration access token
// supplied in the Authorization header.
//
// Both carry their own rate limits rather than relying on the generic API limiter:
// registration is unauthenticated and persists a row on every call, and the
// configuration endpoints are the only place a registration access token can be
// guessed.
func NewOidcRegistrationController(group *gin.RouterGroup, oidcService *service.OidcService, registrationRateLimit, configurationRateLimit gin.HandlerFunc) {
	rc := &OidcRegistrationController{oidcService: oidcService}
	group.POST("/oidc/register", registrationRateLimit, rc.registerHandler)
	group.GET("/oidc/register/:id", configurationRateLimit, rc.getHandler)
	group.PUT("/oidc/register/:id", configurationRateLimit, rc.updateHandler)
	group.DELETE("/oidc/register/:id", configurationRateLimit, rc.deleteHandler)
}

type OidcRegistrationController struct {
	oidcService *service.OidcService
}

func (rc *OidcRegistrationController) registrationClientURI(id string) string {
	// The client configuration endpoint is called back-channel by the client, so it
	// is built on the same internal URL the registration endpoint is advertised on.
	return common.EnvConfig.InternalAppURL + "/api/oidc/register/" + id
}

// dynamicClientMetadataResponse builds the portion of the RFC 7591/7592 response
// metadata that is synthesized from the stored model.OidcClient rather than
// persisted directly. Pocket ID does not store grant_types/response_types/scope
// as columns; a dynamic client's capabilities are always the same fixed set
// (authorization_code + refresh_token grants, code response type), and its
// token_endpoint_auth_method is derived from IsPublic. Callers add any
// operation-specific fields (client_secret, registration_access_token,
// client_id_issued_at) on top of the returned value.
func (rc *OidcRegistrationController) dynamicClientMetadataResponse(client model.OidcClient) fosite.ClientRegistrationResponse {
	tokenEndpointAuthMethod := "client_secret_basic"
	if client.IsPublic {
		tokenEndpointAuthMethod = "none"
	}
	return fosite.ClientRegistrationResponse{
		ClientRegistrationRequest: fosite.ClientRegistrationRequest{
			ClientName:              client.Name,
			RedirectURIs:            []string(client.CallbackURLs),
			GrantTypes:              []string{"authorization_code", "refresh_token"},
			ResponseTypes:           []string{"code"},
			TokenEndpointAuthMethod: tokenEndpointAuthMethod,
		},
		ClientID:              client.ID,
		RegistrationClientURI: rc.registrationClientURI(client.ID),
	}
}

func (rc *OidcRegistrationController) registerHandler(c *gin.Context) {
	var input fosite.ClientRegistrationRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client_metadata", "error_description": err.Error()})
		return
	}
	client, secret, regToken, err := rc.oidcService.RegisterDynamicClient(c.Request.Context(), input)
	if err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) && appErr.Code() == apperror.CodeValidationFailed {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_redirect_uri", "error_description": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		}
		return
	}
	resp := rc.dynamicClientMetadataResponse(client)
	resp.ClientSecret = secret
	resp.ClientIDIssuedAt = client.CreatedAt.ToTime().Unix() // model.Base.CreatedAt is datatype.DateTime
	if secret != "" {
		// RFC 7591 section 3.2.1 requires client_secret_expires_at whenever a
		// client_secret is issued. Pocket ID does not expire client secrets, which
		// the spec expresses as 0. A public client gets no secret, so the field
		// stays absent.
		resp.ClientSecretExpiresAt = fosite.NeverExpires()
	}
	resp.RegistrationAccessToken = regToken
	c.JSON(http.StatusCreated, resp)
}

// bearerToken extracts the registration access token from the Authorization header.
// The scheme is matched case-insensitively because RFC 7235 section 2.1 defines the
// auth-scheme as case-insensitive, so "bearer" is as valid as "Bearer".
func bearerToken(c *gin.Context) string {
	scheme, token, found := strings.Cut(c.GetHeader("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "bearer") {
		return ""
	}
	return token
}

// unauthorized writes the RFC 7592 authentication failure. RFC 6750 section 3
// requires a 401 from a bearer-protected resource to carry a WWW-Authenticate
// challenge so the client knows which scheme to use.
func unauthorized(c *gin.Context) {
	c.Header("WWW-Authenticate", `Bearer error="invalid_token"`)
	c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
}

func (rc *OidcRegistrationController) getHandler(c *gin.Context) {
	client, rotatedToken, err := rc.oidcService.GetDynamicClient(c.Request.Context(), c.Param("id"), bearerToken(c))
	if err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) && appErr.Code() == apperror.CodeInvalidToken {
			unauthorized(c)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		}
		return
	}
	resp := rc.dynamicClientMetadataResponse(client)
	// RFC 7592 section 3 requires the registration access token on every client
	// information response. Only a hash is stored, so the token is rotated on read
	// and the new value returned; the caller must discard the previous one.
	resp.RegistrationAccessToken = rotatedToken
	c.JSON(http.StatusOK, resp)
}

func (rc *OidcRegistrationController) updateHandler(c *gin.Context) {
	var input fosite.ClientRegistrationRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client_metadata", "error_description": err.Error()})
		return
	}
	client, secret, rotatedToken, err := rc.oidcService.UpdateDynamicClient(c.Request.Context(), c.Param("id"), bearerToken(c), input)
	if err != nil {
		var appErr *apperror.Error
		isAppErr := errors.As(err, &appErr)
		switch {
		case isAppErr && appErr.Code() == apperror.CodeInvalidToken:
			unauthorized(c)
		case isAppErr && appErr.Code() == apperror.CodeValidationFailed:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_redirect_uri", "error_description": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		}
		return
	}
	resp := rc.dynamicClientMetadataResponse(client)
	resp.ClientSecret = secret
	if secret != "" {
		// A public-to-confidential transition issues a secret here, so the same
		// RFC 7591 section 3.2.1 requirement applies to the update response.
		resp.ClientSecretExpiresAt = fosite.NeverExpires()
	}
	// As on read, the registration access token is rotated and returned so the
	// response carries the value RFC 7592 section 3 requires.
	resp.RegistrationAccessToken = rotatedToken
	c.JSON(http.StatusOK, resp)
}

func (rc *OidcRegistrationController) deleteHandler(c *gin.Context) {
	if err := rc.oidcService.DeleteDynamicClient(c.Request.Context(), c.Param("id"), bearerToken(c)); err != nil {
		var appErr *apperror.Error
		if errors.As(err, &appErr) && appErr.Code() == apperror.CodeInvalidToken {
			unauthorized(c)
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		}
		return
	}
	c.Status(http.StatusNoContent)
}
