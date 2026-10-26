package service

import (
	"fmt"
	"testing"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	"github.com/stretchr/testify/require"
)

func TestListUsersFiltersMembershipBeforePagination(t *testing.T) {
	service, _ := newTestUserService(t)
	readers := model.UserGroup{Base: model.Base{ID: "group-readers"}, Name: "readers", FriendlyName: "Readers"}
	editors := model.UserGroup{Base: model.Base{ID: "group-editors"}, Name: "editors", FriendlyName: "Editors"}
	require.NoError(t, service.db.Create(&readers).Error)
	require.NoError(t, service.db.Create(&editors).Error)

	// A matching set spans pages, with one member in both groups and one disabled member
	for i := range 21 {
		user := model.User{Base: model.Base{ID: fmt.Sprintf("member-%02d", i)},
			Username: fmt.Sprintf("member-%02d", i), IsAdmin: i == 0, Disabled: i == 1}
		require.NoError(t, service.db.Create(&user).Error)
		require.NoError(t, service.db.Model(&user).Association("UserGroups").Append(&readers))
		if i < 2 {
			require.NoError(t, service.db.Model(&user).Association("UserGroups").Append(&editors))
		}
	}
	outsider := model.User{Base: model.Base{ID: "outsider"}, Username: "outside", FirstName: "outside"}
	require.NoError(t, service.db.Create(&outsider).Error)
	require.NoError(t, service.db.Model(&outsider).Association("UserGroups").Append(&editors))

	options := utils.ListRequestOptions{Filters: map[string][]any{"userGroups": {readers.ID}}}
	options.Pagination.Page, options.Pagination.Limit = 2, 20
	options.Sort.Column, options.Sort.Direction = "username", "asc"
	users, page, err := service.ListUsers(t.Context(), "", options)
	require.NoError(t, err)
	require.EqualValues(t, 21, page.TotalItems)
	require.EqualValues(t, 2, page.TotalPages)
	require.Len(t, users, 1)
	require.Equal(t, "member-20", users[0].Username)

	// Any-of membership must not duplicate users who belong to both selected groups
	options.Pagination.Page, options.Pagination.Limit = 1, 100
	options.Filters["userGroups"] = []any{readers.ID, editors.ID}
	users, page, err = service.ListUsers(t.Context(), "", options)
	require.NoError(t, err)
	require.EqualValues(t, 22, page.TotalItems)
	require.Len(t, users, 22)
	require.Len(t, users[0].UserGroups, 2)

	// Membership must compose with ordinary role, lifecycle and search filters
	options.Filters = map[string][]any{"userGroups": {editors.ID}, "isAdmin": {true}, "disabled": {false}}
	users, _, err = service.ListUsers(t.Context(), "member", options)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, "member-00", users[0].Username)
	users, page, err = service.ListUsers(t.Context(), "outside", options)
	require.NoError(t, err)
	require.Empty(t, users)
	require.Zero(t, page.TotalItems)

	// Unknown group IDs and SQL-shaped values remain parameters, never a wider grant
	for _, group := range []string{"missing", "group-readers') OR 1=1--"} {
		options.Filters = map[string][]any{"userGroups": {group}}
		users, page, err = service.ListUsers(t.Context(), "", options)
		require.NoError(t, err)
		require.Empty(t, users)
		require.Zero(t, page.TotalItems)
	}
	options.Filters = map[string][]any{"userGroups": {}}
	users, page, err = service.ListUsers(t.Context(), "", options)
	require.NoError(t, err)
	require.Len(t, users, 22)
	require.EqualValues(t, 22, page.TotalItems)
}

func TestListUsersRejectsInvalidMembershipFilter(t *testing.T) {
	service, _ := newTestUserService(t)
	for _, value := range []any{true, 123, "", nil} {
		_, _, err := service.ListUsers(t.Context(), "", utils.ListRequestOptions{
			Filters: map[string][]any{"userGroups": {value}},
		})
		require.Error(t, err)
	}
}
