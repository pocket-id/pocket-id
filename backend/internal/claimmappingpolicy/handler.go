package claimmappingpolicy

import (
	"github.com/gin-gonic/gin"
	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type handler struct {
	service *Service
}

func newHandler(service *Service) *handler {
	return &handler{service: service}
}

func (h *handler) list(c *gin.Context) (dto.Paginated[OidcClaimMappingPolicyMetadataDto], error) {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	claimMappingPolicies, pagination, err := h.service.ListClaimMappingPolicy(c.Request.Context(), searchTerm, listRequestOptions)
	if err != nil {
		return dto.Paginated[OidcClaimMappingPolicyMetadataDto]{}, err
	}

	clientCounts, err := h.service.ClientCounts(c.Request.Context(), claimMappingPolicies)
	if err != nil {
		return dto.Paginated[OidcClaimMappingPolicyMetadataDto]{}, err
	}

	dtos := make([]OidcClaimMappingPolicyMetadataDto, len(claimMappingPolicies))
	for i, claimMappingPolicy := range claimMappingPolicies {
		dtos[i] = OidcClaimMappingPolicyMetadataDto{
			ID:          claimMappingPolicy.ID,
			Name:        claimMappingPolicy.Name,
			IsDefault:   claimMappingPolicy.IsDefault,
			CreatedAt:   claimMappingPolicy.CreatedAt,
			ClaimCount:  len(claimMappingPolicy.ClaimMappings),
			ClientCount: clientCounts[claimMappingPolicy.ID],
		}
	}
	return dto.Paginated[OidcClaimMappingPolicyMetadataDto]{
		Data:       dtos,
		Pagination: pagination,
	}, nil
}

func (h *handler) create(c *gin.Context, input OidcClaimMappingPolicyDto) (OidcClaimMappingPolicyDto, error) {
	claimMappingPolicy, err := h.service.CreateClaimMappingPolicy(c.Request.Context(), input)
	if err != nil {
		return OidcClaimMappingPolicyDto{}, err
	}

	return h.toDto(c, claimMappingPolicy)
}

func (h *handler) get(c *gin.Context) (OidcClaimMappingPolicyDto, error) {
	claimMappingPolicy, err := h.service.GetClaimMappingPolicy(c.Request.Context(), c.Param("id"))
	if err != nil {
		return OidcClaimMappingPolicyDto{}, err
	}

	return h.toDto(c, claimMappingPolicy)
}

func (h *handler) toDto(c *gin.Context, claimMappingPolicy OidcClaimMappingPolicy) (OidcClaimMappingPolicyDto, error) {
	clientCounts, err := h.service.ClientCounts(c.Request.Context(), []OidcClaimMappingPolicy{claimMappingPolicy})
	if err != nil {
		return OidcClaimMappingPolicyDto{}, err
	}

	var claimMappingPolicyDto OidcClaimMappingPolicyDto
	err = dto.MapStruct(claimMappingPolicy, &claimMappingPolicyDto)
	if err != nil {
		return OidcClaimMappingPolicyDto{}, err
	}
	claimMappingPolicyDto.ClaimCount = len(claimMappingPolicy.ClaimMappings)
	claimMappingPolicyDto.ClientCount = clientCounts[claimMappingPolicy.ID]
	return claimMappingPolicyDto, nil
}

func (h *handler) delete(c *gin.Context) error {
	return h.service.DeleteClaimMappingPolicy(c.Request.Context(), c.Param("id"))
}

func (h *handler) update(c *gin.Context, update dto.Update[OidcClaimMappingPolicyDto]) (OidcClaimMappingPolicyDto, error) {
	claimMappingPolicy, err := h.service.UpdateClaimMappingPolicy(c.Request.Context(), c.Param("id"), update)
	if err != nil {
		return OidcClaimMappingPolicyDto{}, err
	}

	return h.toDto(c, claimMappingPolicy)
}

func (h *handler) listClients(c *gin.Context) (dto.Paginated[dto.OidcClientMetaDataDto], error) {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	clients, pagination, err := h.service.ListClientsByClaimMappingPolicy(
		c.Request.Context(), c.Param("id"), searchTerm, listRequestOptions,
	)
	if err != nil {
		return dto.Paginated[dto.OidcClientMetaDataDto]{}, err
	}

	var dtos []dto.OidcClientMetaDataDto
	err = dto.MapStructList(clients, &dtos)
	if err != nil {
		return dto.Paginated[dto.OidcClientMetaDataDto]{}, err
	}

	return dto.Paginated[dto.OidcClientMetaDataDto]{
		Data:       dtos,
		Pagination: pagination,
	}, nil
}

func (h *handler) listAssignableClients(c *gin.Context) (dto.Paginated[dto.OidcClientMetaDataDto], error) {
	searchTerm := c.Query("search")
	listRequestOptions := utils.ParseListRequestOptions(c)

	clients, pagination, err := h.service.ListAssignableClients(
		c.Request.Context(), c.Param("id"), searchTerm, listRequestOptions,
	)
	if err != nil {
		return dto.Paginated[dto.OidcClientMetaDataDto]{}, err
	}

	var dtos []dto.OidcClientMetaDataDto
	err = dto.MapStructList(clients, &dtos)
	if err != nil {
		return dto.Paginated[dto.OidcClientMetaDataDto]{}, err
	}

	return dto.Paginated[dto.OidcClientMetaDataDto]{
		Data:       dtos,
		Pagination: pagination,
	}, nil
}

func (h *handler) assignClient(c *gin.Context) error {
	err := h.service.AssignClientToClaimMappingPolicy(
		c.Request.Context(), c.Param("id"), c.Param("clientId"),
	)
	if err != nil {
		return err
	}

	return nil
}

func (h *handler) removeClient(c *gin.Context) error {
	err := h.service.RemoveClientFromClaimMappingPolicy(
		c.Request.Context(), c.Param("id"), c.Param("clientId"),
	)
	if err != nil {
		return err
	}

	return nil
}
