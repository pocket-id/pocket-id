package email

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	"github.com/pocket-id/pocket-id/backend/internal/apperror"
	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
)

const maxSigningUpload = 1 << 20

func (m *Module) RegisterSigningRoutes(r *authz.Router, config appconfig.AppConfigResolver) {
	sender := func(c *gin.Context) (string, error) {
		cfg, err := config.GetConfig(c.Request.Context())
		if err != nil {
			return "", err
		}
		return cfg.SmtpFrom.String(), nil
	}
	status := func(c *gin.Context) error {
		from, err := sender(c)
		if err != nil {
			return err
		}
		result, err := m.SigningStatus(c.Request.Context(), from)
		if err != nil {
			return err
		}
		c.JSON(http.StatusOK, result)
		return nil
	}
	writable := func() error {
		if common.EnvConfig.UiConfigDisabled {
			return apperror.UIConfigDisabled()
		}
		return nil
	}
	r.GET("/application-configuration/smime", authz.ConfigRead, httpserver.Handle(status))
	r.PUT("/application-configuration/smime/certificate", authz.ConfigWrite, httpserver.Handle(func(c *gin.Context) error {
		if err := writable(); err != nil {
			return err
		}
		// Limit the complete multipart request before parsing to prevent oversized files and passwords from consuming memory
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSigningUpload)
		if err := c.Request.ParseMultipartForm(maxSigningUpload); err != nil {
			return signingError("Upload a PKCS#12 file smaller than 1 MB")
		}
		if c.Request.MultipartForm != nil {
			defer c.Request.MultipartForm.RemoveAll()
		}
		file, _, err := c.Request.FormFile("file")
		if err != nil {
			return signingError("A PKCS#12 certificate file is required")
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			return signingError("Unable to read the certificate file")
		}
		from, err := sender(c)
		if err != nil {
			return err
		}
		if err := m.UploadSigningCertificate(c.Request.Context(), data, c.Request.FormValue("password"), from); err != nil {
			return err
		}
		return status(c)
	}))
	r.PUT("/application-configuration/smime", authz.ConfigWrite, httpserver.Handle(func(c *gin.Context) error {
		if err := writable(); err != nil {
			return err
		}
		var input struct {
			Enabled *bool `json:"enabled" binding:"required"`
		}
		if err := httpserver.BindJSON(c, &input); err != nil {
			return err
		}
		from, err := sender(c)
		if err != nil {
			return err
		}
		if err := m.SetSigningEnabled(c.Request.Context(), *input.Enabled, from); err != nil {
			return err
		}
		return status(c)
	}))
	r.DELETE("/application-configuration/smime/certificate", authz.ConfigWrite, httpserver.Handle(func(c *gin.Context) error {
		if err := writable(); err != nil {
			return err
		}
		if err := m.RemoveSigningCertificate(c.Request.Context()); err != nil {
			return err
		}
		return status(c)
	}))
}
