package controller

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/middleware"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/service"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	httpapi "github.com/pocket-id/pocket-id/backend/internal/utils/huma"
)

type oidcClientIDInput struct {
	ID string `path:"id"`
}

type oidcClientListInput struct {
	utils.ListRequestOptions
	Search string `query:"search" required:"false"`
}

type oidcClientCreateInput struct {
	Body dto.OidcClientCreateDto
}

type oidcClientUpdateInput struct {
	ID   string `path:"id"`
	Body dto.OidcClientUpdateDto
}

type oidcAllowedGroupsInput struct {
	ID   string `path:"id"`
	Body dto.OidcUpdateAllowedUserGroupsDto
}

type oidcLogoInput struct {
	ID    string `path:"id"`
	Light string `query:"light" default:"true" required:"false"`
}

type oidcLogoUploadInput struct {
	ID      string `path:"id"`
	Light   string `query:"light" default:"true" required:"false"`
	RawBody huma.MultipartFormFiles[imageUploadForm]
}

type oidcUserAuthorizedClientsInput struct {
	utils.ListRequestOptions
	ID string `path:"id"`
}

type oidcOwnAuthorizedClientsInput struct {
	utils.ListRequestOptions
}

type oidcClientAuthorizationInput struct {
	ClientID string `path:"clientId"`
}

type oidcPreviewInput struct {
	ID     string `path:"id"`
	UserID string `path:"userId"`
	Scopes string `query:"scopes" required:"true"`
}

type oidcLogoOutput struct {
	ContentType   string `header:"Content-Type"`
	ContentLength int64  `header:"Content-Length"`
	CacheControl  string `header:"Cache-Control"`
	Body          func(huma.Context)
}

// NewOidcController registers typed OIDC client management endpoints
func NewOidcController(api huma.API, authMiddleware *middleware.AuthMiddleware, fileSizeLimitMiddleware *middleware.FileSizeLimitMiddleware, oidcService *service.OidcService) {
	controller := &OidcController{oidcService: oidcService}
	adminAuth := authMiddleware.Huma(api)
	userAuth := authMiddleware.WithAdminNotRequired().Huma(api)

	listClients := oidcOperation("list-oidc-clients", http.MethodGet, "/api/oidc/clients", "List OIDC clients")
	adminAuth(&listClients)
	httpapi.Register(api, listClients, controller.listClientsHandler)

	createClient := oidcOperation("create-oidc-client", http.MethodPost, "/api/oidc/clients", "Create OIDC client")
	createClient.DefaultStatus = http.StatusCreated
	adminAuth(&createClient)
	httpapi.Register(api, createClient, controller.createClientHandler)

	getClient := oidcOperation("get-oidc-client", http.MethodGet, "/api/oidc/clients/{id}", "Get OIDC client")
	adminAuth(&getClient)
	httpapi.Register(api, getClient, controller.getClientHandler)

	httpapi.Register(api, oidcOperation("get-oidc-client-metadata", http.MethodGet, "/api/oidc/clients/{id}/meta", "Get OIDC client metadata"), controller.getClientMetaDataHandler)

	updateClient := oidcOperation("update-oidc-client", http.MethodPut, "/api/oidc/clients/{id}", "Update OIDC client")
	adminAuth(&updateClient)
	httpapi.Register(api, updateClient, controller.updateClientHandler)

	deleteClient := oidcOperation("delete-oidc-client", http.MethodDelete, "/api/oidc/clients/{id}", "Delete OIDC client")
	deleteClient.DefaultStatus = http.StatusNoContent
	adminAuth(&deleteClient)
	httpapi.Register(api, deleteClient, controller.deleteClientHandler)

	allowedGroups := oidcOperation("update-oidc-client-allowed-user-groups", http.MethodPut, "/api/oidc/clients/{id}/allowed-user-groups", "Update allowed user groups")
	adminAuth(&allowedGroups)
	httpapi.Register(api, allowedGroups, controller.updateAllowedUserGroupsHandler)

	createSecret := oidcOperation("create-oidc-client-secret", http.MethodPost, "/api/oidc/clients/{id}/secret", "Create client secret")
	adminAuth(&createSecret)
	httpapi.Register(api, createSecret, controller.createClientSecretHandler)

	httpapi.Register(api, oidcOperation("get-oidc-client-logo", http.MethodGet, "/api/oidc/clients/{id}/logo", "Get client logo"), controller.getClientLogoHandler)

	deleteLogo := oidcOperation("delete-oidc-client-logo", http.MethodDelete, "/api/oidc/clients/{id}/logo", "Delete client logo")
	deleteLogo.DefaultStatus = http.StatusNoContent
	adminAuth(&deleteLogo)
	httpapi.Register(api, deleteLogo, controller.deleteClientLogoHandler)

	updateLogo := oidcOperation("update-oidc-client-logo", http.MethodPost, "/api/oidc/clients/{id}/logo", "Update client logo")
	updateLogo.DefaultStatus = http.StatusNoContent
	adminAuth(&updateLogo)
	updateLogo.Middlewares = append(updateLogo.Middlewares, fileSizeLimitMiddleware.Huma(api, 2<<20))
	httpapi.Register(api, updateLogo, controller.updateClientLogoHandler)

	preview := oidcOperation("preview-oidc-client-data", http.MethodGet, "/api/oidc/clients/{id}/preview/{userId}", "Preview OIDC client data for user")
	adminAuth(&preview)
	httpapi.Register(api, preview, controller.getClientPreviewHandler)

	ownAuthorized := oidcOperation("list-own-authorized-oidc-clients", http.MethodGet, "/api/oidc/users/me/authorized-clients", "List authorized clients for current user")
	userAuth(&ownAuthorized)
	httpapi.Register(api, ownAuthorized, controller.listOwnAuthorizedClientsHandler)

	userAuthorized := oidcOperation("list-user-authorized-oidc-clients", http.MethodGet, "/api/oidc/users/{id}/authorized-clients", "List authorized clients for a user")
	adminAuth(&userAuthorized)
	httpapi.Register(api, userAuthorized, controller.listAuthorizedClientsHandler)

	revokeAuthorization := oidcOperation("revoke-own-oidc-client-authorization", http.MethodDelete, "/api/oidc/users/me/authorized-clients/{clientId}", "Revoke authorization for an OIDC client")
	revokeAuthorization.DefaultStatus = http.StatusNoContent
	userAuth(&revokeAuthorization)
	httpapi.Register(api, revokeAuthorization, controller.revokeOwnClientAuthorizationHandler)

	accessibleClients := oidcOperation("list-own-accessible-oidc-clients", http.MethodGet, "/api/oidc/users/me/clients", "List accessible OIDC clients for current user")
	userAuth(&accessibleClients)
	httpapi.Register(api, accessibleClients, controller.listOwnAccessibleClientsHandler)

	clientSCIM := oidcOperation("get-oidc-client-scim-service-provider", http.MethodGet, "/api/oidc/clients/{id}/scim-service-provider", "Get SCIM service provider")
	adminAuth(&clientSCIM)
	httpapi.Register(api, clientSCIM, controller.getClientScimServiceProviderHandler)
}

func oidcOperation(id, method, path, summary string) huma.Operation {
	return huma.Operation{OperationID: id, Method: method, Path: path, Summary: summary, Tags: []string{"OIDC"}}
}

type OidcController struct {
	oidcService *service.OidcService
}

func (oc *OidcController) getClientMetaDataHandler(ctx context.Context, input *oidcClientIDInput) (*httpapi.BodyOutput[dto.OidcClientMetaDataDto], error) {
	client, err := oc.oidcService.GetClient(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	var output dto.OidcClientMetaDataDto
	if err := dto.MapStruct(client, &output); err != nil {
		return nil, err
	}
	output.HasDarkLogo = client.HasDarkLogo()
	return &httpapi.BodyOutput[dto.OidcClientMetaDataDto]{Body: output}, nil
}

func (oc *OidcController) getClientHandler(ctx context.Context, input *oidcClientIDInput) (*httpapi.BodyOutput[dto.OidcClientWithAllowedUserGroupsDto], error) {
	client, err := oc.oidcService.GetClient(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	return mapOIDCClient(client)
}

func (oc *OidcController) listClientsHandler(ctx context.Context, input *oidcClientListInput) (*httpapi.BodyOutput[dto.Paginated[dto.OidcClientWithAllowedGroupsCountDto]], error) {
	clients, pagination, err := oc.oidcService.ListClients(ctx, input.Search, input.ListRequestOptions)
	if err != nil {
		return nil, err
	}
	output := make([]dto.OidcClientWithAllowedGroupsCountDto, len(clients))
	for i := range clients {
		if err := dto.MapStruct(clients[i], &output[i]); err != nil {
			return nil, err
		}
		output[i].HasDarkLogo = clients[i].HasDarkLogo()
		output[i].AllowedUserGroupsCount, err = oc.oidcService.GetAllowedGroupsCountOfClient(ctx, clients[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return &httpapi.BodyOutput[dto.Paginated[dto.OidcClientWithAllowedGroupsCountDto]]{Body: dto.Paginated[dto.OidcClientWithAllowedGroupsCountDto]{Data: output, Pagination: pagination}}, nil
}

func (oc *OidcController) createClientHandler(ctx context.Context, input *oidcClientCreateInput) (*httpapi.BodyOutput[dto.OidcClientWithAllowedUserGroupsDto], error) {
	client, err := oc.oidcService.CreateClient(ctx, input.Body, httpapi.UserID(ctx))
	if err != nil {
		return nil, err
	}
	return mapOIDCClient(client)
}

func (oc *OidcController) deleteClientHandler(ctx context.Context, input *oidcClientIDInput) (*httpapi.EmptyOutput, error) {
	if err := oc.oidcService.DeleteClient(ctx, input.ID); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (oc *OidcController) updateClientHandler(ctx context.Context, input *oidcClientUpdateInput) (*httpapi.BodyOutput[dto.OidcClientWithAllowedUserGroupsDto], error) {
	client, err := oc.oidcService.UpdateClient(ctx, input.ID, input.Body)
	if err != nil {
		return nil, err
	}
	return mapOIDCClient(client)
}

func (oc *OidcController) createClientSecretHandler(ctx context.Context, input *oidcClientIDInput) (*httpapi.BodyOutput[map[string]string], error) {
	secret, err := oc.oidcService.CreateClientSecret(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[map[string]string]{Body: map[string]string{"secret": secret}}, nil
}

func (oc *OidcController) getClientLogoHandler(ctx context.Context, input *oidcLogoInput) (*oidcLogoOutput, error) {
	light, _ := strconv.ParseBool(input.Light)
	reader, size, mimeType, err := oc.oidcService.GetClientLogo(ctx, input.ID, light)
	if err != nil {
		return nil, err
	}
	cacheControl := ""
	if !httpapi.QueryPresent(ctx, "skipCache") {
		cacheControl = utils.CacheControlValue(15*time.Minute, 12*time.Hour)
	}
	return &oidcLogoOutput{
		ContentType:   mimeType,
		ContentLength: size,
		CacheControl:  cacheControl,
		Body: func(streamCtx huma.Context) {
			defer reader.Close()
			_, _ = io.Copy(streamCtx.BodyWriter(), reader)
		},
	}, nil
}

func (oc *OidcController) updateClientLogoHandler(ctx context.Context, input *oidcLogoUploadInput) (*httpapi.EmptyOutput, error) {
	file, err := uploadFile(input.RawBody.Form)
	if err != nil {
		return nil, err
	}
	light, _ := strconv.ParseBool(input.Light)
	if err := oc.oidcService.UpdateClientLogo(ctx, input.ID, file, light); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (oc *OidcController) deleteClientLogoHandler(ctx context.Context, input *oidcLogoInput) (*httpapi.EmptyOutput, error) {
	light, _ := strconv.ParseBool(input.Light)
	var err error
	if light {
		err = oc.oidcService.DeleteClientLogo(ctx, input.ID)
	} else {
		err = oc.oidcService.DeleteClientDarkLogo(ctx, input.ID)
	}
	if err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (oc *OidcController) updateAllowedUserGroupsHandler(ctx context.Context, input *oidcAllowedGroupsInput) (*httpapi.BodyOutput[dto.OidcClientDto], error) {
	client, err := oc.oidcService.UpdateAllowedUserGroups(ctx, input.ID, input.Body)
	if err != nil {
		return nil, err
	}
	var output dto.OidcClientDto
	if err := dto.MapStruct(client, &output); err != nil {
		return nil, err
	}
	output.HasDarkLogo = client.HasDarkLogo()
	return &httpapi.BodyOutput[dto.OidcClientDto]{Body: output}, nil
}

func (oc *OidcController) listOwnAuthorizedClientsHandler(ctx context.Context, input *oidcOwnAuthorizedClientsInput) (*httpapi.BodyOutput[dto.Paginated[dto.AuthorizedOidcClientDto]], error) {
	return oc.listAuthorizedClients(ctx, httpapi.UserID(ctx), input.ListRequestOptions)
}

func (oc *OidcController) listAuthorizedClientsHandler(ctx context.Context, input *oidcUserAuthorizedClientsInput) (*httpapi.BodyOutput[dto.Paginated[dto.AuthorizedOidcClientDto]], error) {
	return oc.listAuthorizedClients(ctx, input.ID, input.ListRequestOptions)
}

func (oc *OidcController) listAuthorizedClients(ctx context.Context, userID string, options utils.ListRequestOptions) (*httpapi.BodyOutput[dto.Paginated[dto.AuthorizedOidcClientDto]], error) {
	clients, pagination, err := oc.oidcService.ListAuthorizedClients(ctx, userID, options)
	if err != nil {
		return nil, err
	}
	var output []dto.AuthorizedOidcClientDto
	if err := dto.MapStructList(clients, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[dto.Paginated[dto.AuthorizedOidcClientDto]]{Body: dto.Paginated[dto.AuthorizedOidcClientDto]{Data: output, Pagination: pagination}}, nil
}

func (oc *OidcController) revokeOwnClientAuthorizationHandler(ctx context.Context, input *oidcClientAuthorizationInput) (*httpapi.EmptyOutput, error) {
	if err := oc.oidcService.RevokeAuthorizedClient(ctx, httpapi.UserID(ctx), input.ClientID); err != nil {
		return nil, err
	}
	return &httpapi.EmptyOutput{}, nil
}

func (oc *OidcController) listOwnAccessibleClientsHandler(ctx context.Context, input *oidcOwnAuthorizedClientsInput) (*httpapi.BodyOutput[dto.Paginated[dto.AccessibleOidcClientDto]], error) {
	clients, pagination, err := oc.oidcService.ListAccessibleOidcClients(ctx, httpapi.UserID(ctx), input.ListRequestOptions)
	if err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[dto.Paginated[dto.AccessibleOidcClientDto]]{Body: dto.Paginated[dto.AccessibleOidcClientDto]{Data: clients, Pagination: pagination}}, nil
}

func (oc *OidcController) getClientPreviewHandler(ctx context.Context, input *oidcPreviewInput) (*httpapi.BodyOutput[dto.OidcClientPreviewDto], error) {
	if input.ID == "" {
		return nil, &common.ValidationError{Message: "client ID is required"}
	}
	if input.UserID == "" {
		return nil, &common.ValidationError{Message: "user ID is required"}
	}
	if input.Scopes == "" {
		return nil, &common.ValidationError{Message: "scopes are required"}
	}
	preview, err := oc.oidcService.GetClientPreview(ctx, input.ID, input.UserID, strings.Split(input.Scopes, " "), httpapi.AuthenticationMethod(ctx))
	if err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[dto.OidcClientPreviewDto]{Body: *preview}, nil
}

func (oc *OidcController) getClientScimServiceProviderHandler(ctx context.Context, input *oidcClientIDInput) (*httpapi.BodyOutput[dto.ScimServiceProviderDTO], error) {
	provider, err := oc.oidcService.GetClientScimServiceProvider(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	var output dto.ScimServiceProviderDTO
	if err := dto.MapStruct(provider, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[dto.ScimServiceProviderDTO]{Body: output}, nil
}

func mapOIDCClient(client model.OidcClient) (*httpapi.BodyOutput[dto.OidcClientWithAllowedUserGroupsDto], error) {
	var output dto.OidcClientWithAllowedUserGroupsDto
	if err := dto.MapStruct(client, &output); err != nil {
		return nil, err
	}
	return &httpapi.BodyOutput[dto.OidcClientWithAllowedUserGroupsDto]{Body: output}, nil
}
