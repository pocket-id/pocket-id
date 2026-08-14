package runtimecredential

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pocket-id/pocket-id/backend/internal/appconfig"
	dto "github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/httpserver"
	"github.com/pocket-id/pocket-id/backend/internal/utils/cookie"
)

type handler struct {
	service   *Service
	appConfig appconfig.AppConfigResolver
}

func newHandler(service *Service, appConfig appconfig.AppConfigResolver) *handler {
	return &handler{service: service, appConfig: appConfig}
}

func (h *handler) beginRegistration(c *gin.Context) error {
	var input registrationStartDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}
	deviceToken, _ := c.Cookie(cookie.DeviceTokenCookieName)
	challenge, err := h.service.BeginRegistration(c.Request.Context(), input, deviceToken)
	if err != nil {
		return err
	}
	c.JSON(http.StatusCreated, challenge)
	return nil
}

func (h *handler) finishRegistration(c *gin.Context) error {
	cfg, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}
	var input proofFinishDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}
	user, credential, token, err := h.service.FinishRegistration(c.Request.Context(), cfg, input, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		return err
	}

	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		return err
	}
	var credentialDto RuntimeCredentialDto
	if err := dto.MapStruct(credential, &credentialDto); err != nil {
		return err
	}
	cookie.AddAccessTokenCookie(c, int(cfg.SessionDuration.AsDurationMinutes().Seconds()), token)
	c.JSON(http.StatusCreated, registrationFinishDto{User: userDto, Credential: credentialDto})
	return nil
}

func (h *handler) beginLogin(c *gin.Context) error {
	var input loginStartDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}
	challenge, err := h.service.BeginLogin(c.Request.Context(), input)
	if err != nil {
		return err
	}
	c.JSON(http.StatusCreated, challenge)
	return nil
}

func (h *handler) finishLogin(c *gin.Context) error {
	cfg, err := h.appConfig.GetConfig(c.Request.Context())
	if err != nil {
		return fmt.Errorf("error loading app configuration: %w", err)
	}
	var input proofFinishDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}
	user, token, err := h.service.FinishLogin(c.Request.Context(), cfg, input, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		return err
	}
	var userDto dto.UserDto
	if err := dto.MapStruct(user, &userDto); err != nil {
		return err
	}
	cookie.AddAccessTokenCookie(c, int(cfg.SessionDuration.AsDurationMinutes().Seconds()), token)
	c.JSON(http.StatusOK, userDto)
	return nil
}

func (h *handler) beginReauthentication(c *gin.Context) error {
	var input reauthenticationStartDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}
	challenge, err := h.service.BeginReauthentication(c.Request.Context(), c.GetString("userID"), input.CredentialID)
	if err != nil {
		return err
	}
	c.JSON(http.StatusCreated, challenge)
	return nil
}

func (h *handler) finishReauthentication(c *gin.Context) error {
	var input proofFinishDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}
	token, err := h.service.FinishReauthentication(c.Request.Context(), c.GetString("userID"), input, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		return err
	}
	cookie.AddReauthenticationTokenCookie(c, token)
	c.Status(http.StatusNoContent)
	return nil
}

func (h *handler) listOwnCredentials(c *gin.Context) error {
	return h.listCredentials(c, c.GetString("userID"))
}

func (h *handler) listUserCredentials(c *gin.Context) error {
	return h.listCredentials(c, c.Param("id"))
}

func (h *handler) listCredentials(c *gin.Context, userID string) error {
	credentials, err := h.service.ListCredentials(c.Request.Context(), userID)
	if err != nil {
		return err
	}
	var output []RuntimeCredentialDto
	if err := dto.MapStructList(credentials, &output); err != nil {
		return err
	}
	c.JSON(http.StatusOK, output)
	return nil
}

func (h *handler) updateOwnCredential(c *gin.Context) error {
	var input runtimeCredentialUpdateDto
	if err := httpserver.BindJSON(c, &input); err != nil {
		return err
	}
	credential, err := h.service.UpdateCredential(c.Request.Context(), c.GetString("userID"), c.Param("id"), input.Name)
	if err != nil {
		return err
	}
	var output RuntimeCredentialDto
	if err := dto.MapStruct(credential, &output); err != nil {
		return err
	}
	c.JSON(http.StatusOK, output)
	return nil
}

func (h *handler) revokeOwnCredential(c *gin.Context) error {
	userID := c.GetString("userID")
	if err := h.service.RevokeCredential(c.Request.Context(), userID, c.Param("id"), c.ClientIP(), c.Request.UserAgent(), userID); err != nil {
		return err
	}
	c.Status(http.StatusNoContent)
	return nil
}

func (h *handler) revokeUserCredential(c *gin.Context) error {
	if err := h.service.RevokeCredential(c.Request.Context(), c.Param("id"), c.Param("credentialId"), c.ClientIP(), c.Request.UserAgent(), c.GetString("userID")); err != nil {
		return err
	}
	c.Status(http.StatusNoContent)
	return nil
}
