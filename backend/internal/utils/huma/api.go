package humautils

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

const scalarDocsHTML = `<!doctype html>
<html>
  <head>
    <title>Pocket ID API Reference</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
  </head>
  <body>
    <script
      id="api-reference"
      data-url="/api/openapi.json"
      data-configuration='{"theme":"purple","darkMode":true,"layout":"modern","hiddenClients":["unirest"],"defaultHttpClient":{"targetKey":"shell","clientKey":"curl"}}'></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.62.5" integrity="sha384-jVBCKhcCfx34USN27x4iQK1SBNdL/HxKq3KuBAxTS4WPaP5w80K4fjpwB+DezJL5" crossorigin="anonymous"></script>
  </body>
</html>`

var ginCompatibleJSONFormat = huma.Format{
	Marshal: func(w io.Writer, value any) error {
		data, err := json.Marshal(value)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	},
	Unmarshal: json.Unmarshal,
}

// New creates the Huma API on the existing rate-limited Gin group
func New(r *gin.Engine, group *gin.RouterGroup) huma.API {
	config := huma.DefaultConfig("Pocket ID API", common.Version)
	config.CreateHooks = nil
	config.DocsPath = ""
	config.OpenAPIPath = "/api/openapi"
	config.SchemasPath = "/api/schemas"
	config.AllowAdditionalPropertiesByDefault = true
	config.Security = nil
	config.Formats = map[string]huma.Format{
		"application/json": ginCompatibleJSONFormat,
		"json":             ginCompatibleJSONFormat,
	}
	config.DefaultFormat = "application/json"
	config.OnAddOperation = append(config.OnAddOperation, rewriteValidationResponse)
	if common.EnvConfig.AppURL != "" {
		config.Servers = []*huma.Server{{URL: common.EnvConfig.AppURL}}
	}
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"BearerAuth": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
			Description:  "Pocket ID session JWT sent in the Authorization header",
		},
		"SessionCookie": {
			Type:        "apiKey",
			In:          "cookie",
			Name:        cookie.AccessTokenCookieName,
			Description: "Pocket ID browser session cookie",
		},
		"ApiKeyAuth": {
			Type:        "apiKey",
			In:          "header",
			Name:        "X-API-Key",
			Description: "Pocket ID API key",
		},
		"OIDCAccessToken": {
			Type:        "http",
			Scheme:      "bearer",
			Description: "OIDC access token",
		},
		"OIDCClientBasic": {
			Type:        "http",
			Scheme:      "basic",
			Description: "OIDC client credentials",
		},
	}

	humagin.MultipartMaxMemory = r.MaxMultipartMemory
	api := humagin.NewWithGroup(r, group, config)
	api.UseMiddleware(CaptureRequestContext)
	registerScalarDocs(group)
	return api
}

func rewriteValidationResponse(_ *huma.OpenAPI, operation *huma.Operation) {
	response, ok := operation.Responses["422"]
	if !ok {
		return
	}
	if _, exists := operation.Responses["400"]; !exists {
		operation.Responses["400"] = response
	}
	delete(operation.Responses, "422")
}

func registerScalarDocs(group *gin.RouterGroup) {
	group.GET("/api/docs", func(ctx *gin.Context) {
		nonce := utils.GetCSPNonce(ctx)
		ctx.Header("Content-Security-Policy", utils.BuildAPIDocsCSP(nonce))
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", []byte(scalarDocsHTML))
	})
}
