package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/dto"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

func TestUserGroupListSearchMatchesFriendlyName(t *testing.T) {
	_, groupService := newTestUserService(t)

	_, err := groupService.Create(t.Context(), dto.UserGroupCreateDto{
		Name:         "qa_team",
		FriendlyName: "QA Engineering",
	})
	require.NoError(t, err)

	// A search for the friendly name must find the group, since the
	// friendly name is the name shown in the UI.
	groups, pagination, err := groupService.List(t.Context(), "Engineering", utils.ListRequestOptions{})
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, "QA Engineering", groups[0].FriendlyName)
	require.EqualValues(t, 1, pagination.TotalItems)

	// The internal name must remain searchable too.
	groups, _, err = groupService.List(t.Context(), "qa_team", utils.ListRequestOptions{})
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, "QA Engineering", groups[0].FriendlyName)

	// A fragment matching neither field returns nothing.
	groups, _, err = groupService.List(t.Context(), "Design", utils.ListRequestOptions{})
	require.NoError(t, err)
	require.Empty(t, groups)
}
