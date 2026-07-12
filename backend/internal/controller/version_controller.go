package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

type versionOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         map[string]string
}

// NewVersionController registers version-related routes
func NewVersionController(api huma.API, authMiddleware *middleware.AuthMiddleware, versionService *service.VersionService) {
	vc := &VersionController{versionService: versionService}
	userAuth := authMiddleware.WithAdminNotRequired().Huma(api)

	httpapi.Register(api, huma.Operation{
		OperationID: "get-latest-version",
		Method:      http.MethodGet,
		Path:        "/api/version/latest",
		Summary:     "Get latest available version of Pocket ID",
		Tags:        []string{"Version"},
	}, vc.getLatestVersionHandler)

	httpapi.Register(api, huma.Operation{
		OperationID: "get-current-version",
		Method:      http.MethodGet,
		Path:        "/api/version/current",
		Summary:     "Get current deployed version of Pocket ID",
		Tags:        []string{"Version"},
	}, vc.getCurrentVersionHandler, userAuth)
}

type VersionController struct {
	versionService *service.VersionService
}

func (vc *VersionController) getLatestVersionHandler(ctx context.Context, _ *httpapi.EmptyInput) (*versionOutput, error) {
	tag, err := vc.versionService.GetLatestVersion(ctx)
	if err != nil {
		return nil, err
	}

	cacheControl := ""
	if !httpapi.QueryPresent(ctx, "skipCache") {
		cacheControl = utils.CacheControlValue(5*time.Minute, 15*time.Minute)
	}
	return &versionOutput{CacheControl: cacheControl, Body: map[string]string{"latestVersion": tag}}, nil
}

func (vc *VersionController) getCurrentVersionHandler(_ context.Context, _ *httpapi.EmptyInput) (*httpapi.BodyOutput[map[string]string], error) {
	return &httpapi.BodyOutput[map[string]string]{Body: map[string]string{"currentVersion": common.Version}}, nil
}
