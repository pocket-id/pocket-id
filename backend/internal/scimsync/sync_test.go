package scimsync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

const (
	mockSCIMEndpoint           = "https://scim.example.test"
	scimListResponseSchema     = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	scimErrorResponseSchema    = "urn:ietf:params:scim:api:messages:2.0:Error"
	mockSCIMRequestContentType = "application/scim+json"
)

type scimSyncFixture struct {
	db        *gorm.DB
	service   *Service
	transport *mockSCIMTransport
	client    model.OidcClient
	provider  ServiceProvider
}

func newSCIMSyncFixture(t *testing.T, restricted bool) *scimSyncFixture {
	t.Helper()

	db := testutils.NewDatabaseForTest(t)
	providerToken := t.Name()
	client := model.OidcClient{
		Base:              model.Base{ID: "oidc-client"},
		Name:              "SCIM client",
		IsGroupRestricted: restricted,
	}
	err := db.Create(&client).Error
	require.NoError(t, err)

	provider := ServiceProvider{
		Base:         model.Base{ID: "scim-provider"},
		Endpoint:     mockSCIMEndpoint,
		Token:        datatype.EncryptedString(providerToken),
		OidcClientID: client.ID,
	}
	err = db.Create(&provider).Error
	require.NoError(t, err)

	transport := newMockSCIMTransport(providerToken)
	service := newService(db, &http.Client{Transport: transport})

	return &scimSyncFixture{
		db:        db,
		service:   service,
		transport: transport,
		client:    client,
		provider:  provider,
	}
}

func (f *scimSyncFixture) createUser(t *testing.T, id, username string, email *string, disabled bool) model.User {
	t.Helper()

	user := model.User{
		Base:        model.Base{ID: id},
		Username:    username,
		Email:       email,
		FirstName:   strings.ToUpper(username[:1]) + username[1:],
		LastName:    "Example",
		DisplayName: username + " display",
		Disabled:    disabled,
	}
	err := f.db.Create(&user).Error
	require.NoError(t, err)
	return user
}

func (f *scimSyncFixture) createGroup(t *testing.T, id, name string, users ...model.User) model.UserGroup {
	t.Helper()

	group := model.UserGroup{
		Base:         model.Base{ID: id},
		Name:         name,
		FriendlyName: name + " friendly",
	}
	err := f.db.Create(&group).Error
	require.NoError(t, err)
	if len(users) > 0 {
		err = f.db.Model(&group).Association("Users").Replace(users)
		require.NoError(t, err)
	}
	return group
}

func (f *scimSyncFixture) allowGroups(t *testing.T, groups ...model.UserGroup) {
	t.Helper()
	err := f.db.Model(&f.client).Association("AllowedUserGroups").Replace(groups)
	require.NoError(t, err)
}

func (f *scimSyncFixture) requireLastSynced(t *testing.T, expected bool) {
	t.Helper()

	var provider ServiceProvider
	err := f.db.First(&provider, "id = ?", f.provider.ID).Error
	require.NoError(t, err)
	if expected {
		require.NotNil(t, provider.LastSyncedAt)
	} else {
		require.Nil(t, provider.LastSyncedAt)
	}
}

func TestSyncCreatesCompliantUsersBeforeGroups(t *testing.T) {
	fixture := newSCIMSyncFixture(t, false)
	aliceEmail := "alice@example.com"
	alice := fixture.createUser(t, "user-alice", "alice", &aliceEmail, false)
	bob := fixture.createUser(t, "user-bob", "bob", nil, true)
	group := fixture.createGroup(t, "group-engineering", "engineering", alice, bob)

	require.NoError(t, fixture.service.SyncServiceProvider(t.Context(), fixture.provider.ID))

	users := fixture.transport.usersSnapshot()
	require.Len(t, users, 2)
	remoteAlice := resourceByExternalID(alice.ID, users)
	require.NotNil(t, remoteAlice)
	assert.Equal(t, "alice", remoteAlice.UserName)
	assert.Equal(t, "Alice", remoteAlice.Name.GivenName)
	assert.Equal(t, "Example", remoteAlice.Name.FamilyName)
	assert.Equal(t, alice.DisplayName, remoteAlice.Display)
	assert.True(t, remoteAlice.Active)
	require.Equal(t, []ScimEmail{{Value: aliceEmail, Primary: true}}, remoteAlice.Emails)

	remoteBob := resourceByExternalID(bob.ID, users)
	require.NotNil(t, remoteBob)
	assert.False(t, remoteBob.Active)
	assert.Empty(t, remoteBob.Emails)

	groups := fixture.transport.groupsSnapshot()
	require.Len(t, groups, 1)
	remoteGroup := resourceByExternalID(group.ID, groups)
	require.NotNil(t, remoteGroup)
	assert.Equal(t, group.FriendlyName, remoteGroup.Display)
	assert.ElementsMatch(t, []ScimGroupMember{{Value: remoteAlice.ID}, {Value: remoteBob.ID}}, remoteGroup.Members)

	requests := fixture.transport.requestsSnapshot()
	lastUserCreate := lastRequestIndex(requests, http.MethodPost, "/Users")
	firstGroupCreate := firstRequestIndex(requests, http.MethodPost, "/Groups")
	require.NotEqual(t, -1, lastUserCreate)
	require.Greater(t, firstGroupCreate, lastUserCreate)
	fixture.requireLastSynced(t, true)
	fixture.transport.requireCompliant(t)
}

func TestSyncUpdatesExistingUsersAndGroupsWithPUT(t *testing.T) {
	fixture := newSCIMSyncFixture(t, false)
	email := "updated@example.com"
	user := fixture.createUser(t, "user-updated", "updated", &email, true)
	group := fixture.createGroup(t, "group-updated", "updated", user)
	remoteModified := time.Now().Add(-time.Hour)

	fixture.transport.seedUser(ScimUser{
		ScimResourceData: remoteResourceData("remote-user", user.ID, scimUserSchema, "User", remoteModified),
		UserName:         "stale-name",
		Active:           true,
	})
	fixture.transport.seedGroup(ScimGroup{
		ScimResourceData: remoteResourceData("remote-group", group.ID, scimGroupSchema, "Group", remoteModified),
		Display:          "stale-group",
	})

	require.NoError(t, fixture.service.SyncServiceProvider(t.Context(), fixture.provider.ID))

	remoteUser := fixture.transport.user("remote-user")
	require.NotNil(t, remoteUser)
	assert.Equal(t, user.Username, remoteUser.UserName)
	assert.Equal(t, user.DisplayName, remoteUser.Display)
	assert.False(t, remoteUser.Active)
	require.Equal(t, []ScimEmail{{Value: email, Primary: true}}, remoteUser.Emails)

	remoteGroup := fixture.transport.group("remote-group")
	require.NotNil(t, remoteGroup)
	assert.Equal(t, group.FriendlyName, remoteGroup.Display)
	require.Equal(t, []ScimGroupMember{{Value: remoteUser.ID}}, remoteGroup.Members)

	requests := fixture.transport.requestsSnapshot()
	assert.Equal(t, 1, countRequests(requests, http.MethodPut, "/Users/remote-user"))
	assert.Equal(t, 1, countRequests(requests, http.MethodPut, "/Groups/remote-group"))
	assert.Zero(t, countRequestsWithPrefix(requests, http.MethodPost, "/"))
	fixture.requireLastSynced(t, true)
	fixture.transport.requireCompliant(t)
}

func TestSyncRestrictedClientDeletesDisallowedResources(t *testing.T) {
	fixture := newSCIMSyncFixture(t, true)
	allowedUser := fixture.createUser(t, "user-allowed", "allowed", nil, false)
	deniedUser := fixture.createUser(t, "user-denied", "denied", nil, false)
	allowedGroup := fixture.createGroup(t, "group-allowed", "allowed", allowedUser)
	deniedGroup := fixture.createGroup(t, "group-denied", "denied", deniedUser)
	fixture.allowGroups(t, allowedGroup)
	remoteModified := time.Now().Add(time.Hour)

	fixture.transport.seedUser(ScimUser{
		ScimResourceData: remoteResourceData("remote-allowed-user", allowedUser.ID, scimUserSchema, "User", remoteModified),
		UserName:         allowedUser.Username,
		Active:           true,
	})
	fixture.transport.seedUser(ScimUser{
		ScimResourceData: remoteResourceData("remote-denied-user", deniedUser.ID, scimUserSchema, "User", remoteModified),
		UserName:         deniedUser.Username,
		Active:           true,
	})
	fixture.transport.seedGroup(ScimGroup{
		ScimResourceData: remoteResourceData("remote-allowed-group", allowedGroup.ID, scimGroupSchema, "Group", remoteModified),
		Display:          allowedGroup.FriendlyName,
		Members:          []ScimGroupMember{{Value: "remote-allowed-user"}},
	})
	fixture.transport.seedGroup(ScimGroup{
		ScimResourceData: remoteResourceData("remote-denied-group", deniedGroup.ID, scimGroupSchema, "Group", remoteModified),
		Display:          deniedGroup.FriendlyName,
		Members:          []ScimGroupMember{{Value: "remote-denied-user"}},
	})

	require.NoError(t, fixture.service.SyncServiceProvider(t.Context(), fixture.provider.ID))

	assert.NotNil(t, fixture.transport.user("remote-allowed-user"))
	assert.Nil(t, fixture.transport.user("remote-denied-user"))
	assert.NotNil(t, fixture.transport.group("remote-allowed-group"))
	assert.Nil(t, fixture.transport.group("remote-denied-group"))
	requests := fixture.transport.requestsSnapshot()
	assert.Equal(t, 1, countRequests(requests, http.MethodDelete, "/Users/remote-denied-user"))
	assert.Equal(t, 1, countRequests(requests, http.MethodDelete, "/Groups/remote-denied-group"))
	fixture.requireLastSynced(t, true)
	fixture.transport.requireCompliant(t)
}

func TestSyncSkipsResourcesNewerThanTheLocalSnapshotAndPaginates(t *testing.T) {
	fixture := newSCIMSyncFixture(t, false)
	alice := fixture.createUser(t, "user-alice", "alice", nil, false)
	bob := fixture.createUser(t, "user-bob", "bob", nil, false)
	remoteModified := time.Now().Add(time.Hour)
	fixture.transport.pageSize = 1

	fixture.transport.seedUser(ScimUser{
		ScimResourceData: remoteResourceData("remote-alice", alice.ID, scimUserSchema, "User", remoteModified),
		UserName:         alice.Username,
		Active:           true,
	})
	fixture.transport.seedUser(ScimUser{
		ScimResourceData: remoteResourceData("remote-bob", bob.ID, scimUserSchema, "User", remoteModified),
		UserName:         bob.Username,
		Active:           true,
	})

	require.NoError(t, fixture.service.SyncServiceProvider(t.Context(), fixture.provider.ID))

	requests := fixture.transport.requestsSnapshot()
	assert.Equal(t, []string{"1", "2"}, queryValues(requests, http.MethodGet, "/Users", "startIndex"))
	assert.Equal(t, []string{"1000", "1000"}, queryValues(requests, http.MethodGet, "/Users", "count"))
	assert.Zero(t, countMutationRequests(requests))
	fixture.requireLastSynced(t, true)
	fixture.transport.requireCompliant(t)
}

func TestSyncContinuesAfterResourceFailureAndDoesNotMarkCompletion(t *testing.T) {
	fixture := newSCIMSyncFixture(t, false)
	fixture.createUser(t, "user-success", "success", nil, false)
	fixture.createUser(t, "user-failure", "failure", nil, false)
	fixture.transport.failCreates["user-failure"] = http.StatusInternalServerError

	err := fixture.service.SyncServiceProvider(t.Context(), fixture.provider.ID)
	require.Error(t, err)
	require.ErrorContains(t, err, "status 500")

	users := fixture.transport.usersSnapshot()
	assert.NotNil(t, resourceByExternalID("user-success", users))
	assert.Nil(t, resourceByExternalID("user-failure", users))
	fixture.requireLastSynced(t, false)
	fixture.transport.requireCompliant(t)
}

func TestSyncRetriesRateLimitedSCIMRequests(t *testing.T) {
	fixture := newSCIMSyncFixture(t, false)
	fixture.transport.rateLimits[http.MethodGet+" /Users"] = 2

	require.NoError(t, fixture.service.SyncServiceProvider(t.Context(), fixture.provider.ID))

	requests := fixture.transport.requestsSnapshot()
	assert.Equal(t, 3, countRequests(requests, http.MethodGet, "/Users"))
	fixture.requireLastSynced(t, true)
	fixture.transport.requireCompliant(t)
}

func remoteResourceData(id, externalID, schema, resourceType string, modified time.Time) ScimResourceData {
	return ScimResourceData{
		ID:         id,
		ExternalID: externalID,
		Schemas:    []string{schema},
		Meta: &ScimResourceMeta{
			Location:     mockSCIMEndpoint + "/" + resourceType + "s/" + id,
			ResourceType: resourceType,
			Created:      modified.Add(-time.Hour),
			LastModified: modified,
			Version:      `W/"seed"`,
		},
	}
}

func resourceByExternalID[T ScimResource](externalID string, resources map[string]T) *T {
	for _, resource := range resources {
		if resource.GetExternalID() == externalID {
			result := resource
			return &result
		}
	}
	return nil
}

type mockSCIMRequest struct {
	method string
	path   string
	query  url.Values
	body   []byte
}

// mockSCIMTransport is a *http.Transport that mocks HTTP server responses to test for compliance with SCIM specs
type mockSCIMTransport struct {
	mu sync.Mutex

	expectedToken string
	pageSize      int
	nextID        int
	users         map[string]ScimUser
	groups        map[string]ScimGroup
	requests      []mockSCIMRequest
	violations    []string
	rateLimits    map[string]int
	failCreates   map[string]int
}

func newMockSCIMTransport(expectedToken string) *mockSCIMTransport {
	return &mockSCIMTransport{
		expectedToken: expectedToken,
		users:         map[string]ScimUser{},
		groups:        map[string]ScimGroup{},
		rateLimits:    map[string]int{},
		failCreates:   map[string]int{},
	}
}

func (m *mockSCIMTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.requests = append(m.requests, mockSCIMRequest{
		method: req.Method,
		path:   req.URL.Path,
		query:  req.URL.Query(),
		body:   slices.Clone(body),
	})

	if violation := m.validateRequest(req, body); violation != "" {
		m.violations = append(m.violations, violation)
		return mockSCIMErrorResponse(req, http.StatusBadRequest, violation), nil
	}

	requestKey := req.Method + " " + req.URL.Path
	if m.rateLimits[requestKey] > 0 {
		m.rateLimits[requestKey]--
		response := mockSCIMErrorResponse(req, http.StatusTooManyRequests, "rate limited")
		response.Header.Set("Retry-After", "0")
		return response, nil
	}

	segments := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	if len(segments) == 0 || segments[0] == "" {
		return mockSCIMErrorResponse(req, http.StatusNotFound, "resource path is empty"), nil
	}

	switch segments[0] {
	case "Users":
		return m.handleUsers(req, segments, body), nil
	case "Groups":
		return m.handleGroups(req, segments, body), nil
	default:
		return mockSCIMErrorResponse(req, http.StatusNotFound, "resource type is unknown"), nil
	}
}

func (m *mockSCIMTransport) validateRequest(req *http.Request, body []byte) string {
	if req.URL.Scheme != "https" || req.URL.Host != "scim.example.test" {
		return fmt.Sprintf("request used unexpected SCIM endpoint %s", req.URL.String())
	}
	if req.Header.Get("Accept") != mockSCIMRequestContentType {
		return "request did not accept application/scim+json"
	}
	if req.Header.Get("Authorization") != "Bearer "+m.expectedToken {
		return "request did not use the configured bearer token"
	}
	if len(body) > 0 && req.Header.Get("Content-Type") != mockSCIMRequestContentType {
		return "request body did not use application/scim+json"
	}

	return ""
}

func (m *mockSCIMTransport) handleUsers(req *http.Request, segments []string, body []byte) *http.Response {
	switch req.Method {
	case http.MethodGet:
		if len(segments) != 1 {
			return mockSCIMErrorResponse(req, http.StatusMethodNotAllowed, "individual user reads are unsupported")
		}
		resources := make([]ScimUser, 0, len(m.users))
		for _, user := range m.users {
			resources = append(resources, user)
		}
		sort.Slice(resources, func(i, j int) bool { return resources[i].ID < resources[j].ID })
		return mockSCIMListResponse(req, resources, m.pageSize)

	case http.MethodPost:
		if len(segments) != 1 {
			return mockSCIMErrorResponse(req, http.StatusNotFound, "user collection path is invalid")
		}
		var user ScimUser
		err := json.Unmarshal(body, &user)
		if err != nil {
			return mockSCIMErrorResponse(req, http.StatusBadRequest, "user payload is invalid JSON")
		}
		violation := validateUserPayload(body, user)
		if violation != "" {
			m.violations = append(m.violations, violation)
			return mockSCIMErrorResponse(req, http.StatusBadRequest, violation)
		}
		status := m.failCreates[user.ExternalID]
		if status != 0 {
			return mockSCIMErrorResponse(req, status, "injected user creation failure")
		}
		m.nextID++
		user.ID = fmt.Sprintf("remote-user-%d", m.nextID)
		user.Meta = newMockMeta("User", "/Users/"+user.ID, m.nextID)
		m.users[user.ID] = user
		return mockSCIMJSONResponse(req, http.StatusCreated, user)

	case http.MethodPut:
		if len(segments) != 2 {
			return mockSCIMErrorResponse(req, http.StatusNotFound, "user resource path is invalid")
		}
		_, ok := m.users[segments[1]]
		if !ok {
			return mockSCIMErrorResponse(req, http.StatusNotFound, "user does not exist")
		}
		var user ScimUser
		err := json.Unmarshal(body, &user)
		if err != nil {
			return mockSCIMErrorResponse(req, http.StatusBadRequest, "user payload is invalid JSON")
		}
		violation := validateUserPayload(body, user)
		if violation != "" {
			m.violations = append(m.violations, violation)
			return mockSCIMErrorResponse(req, http.StatusBadRequest, violation)
		}
		m.nextID++
		user.ID = segments[1]
		user.Meta = newMockMeta("User", "/Users/"+user.ID, m.nextID)
		m.users[user.ID] = user
		return mockSCIMJSONResponse(req, http.StatusOK, user)

	case http.MethodDelete:
		if len(segments) != 2 {
			return mockSCIMErrorResponse(req, http.StatusNotFound, "user resource path is invalid")
		}
		_, ok := m.users[segments[1]]
		if !ok {
			return mockSCIMErrorResponse(req, http.StatusNotFound, "user does not exist")
		}
		delete(m.users, segments[1])
		return mockSCIMNoContentResponse(req)

	default:
		return mockSCIMErrorResponse(req, http.StatusMethodNotAllowed, "user method is unsupported")
	}
}

func (m *mockSCIMTransport) handleGroups(req *http.Request, segments []string, body []byte) *http.Response {
	switch req.Method {
	case http.MethodGet:
		if len(segments) != 1 {
			return mockSCIMErrorResponse(req, http.StatusMethodNotAllowed, "individual group reads are unsupported")
		}
		resources := make([]ScimGroup, 0, len(m.groups))
		for _, group := range m.groups {
			resources = append(resources, group)
		}
		sort.Slice(resources, func(i, j int) bool { return resources[i].ID < resources[j].ID })
		return mockSCIMListResponse(req, resources, m.pageSize)

	case http.MethodPost:
		if len(segments) != 1 {
			return mockSCIMErrorResponse(req, http.StatusNotFound, "group collection path is invalid")
		}
		var group ScimGroup
		err := json.Unmarshal(body, &group)
		if err != nil {
			return mockSCIMErrorResponse(req, http.StatusBadRequest, "group payload is invalid JSON")
		}
		violation := m.validateGroupPayload(body, group)
		if violation != "" {
			m.violations = append(m.violations, violation)
			return mockSCIMErrorResponse(req, http.StatusBadRequest, violation)
		}
		status := m.failCreates[group.ExternalID]
		if status != 0 {
			return mockSCIMErrorResponse(req, status, "injected group creation failure")
		}
		m.nextID++
		group.ID = fmt.Sprintf("remote-group-%d", m.nextID)
		group.Meta = newMockMeta("Group", "/Groups/"+group.ID, m.nextID)
		m.groups[group.ID] = group
		return mockSCIMJSONResponse(req, http.StatusCreated, group)

	case http.MethodPut:
		if len(segments) != 2 {
			return mockSCIMErrorResponse(req, http.StatusNotFound, "group resource path is invalid")
		}
		_, ok := m.groups[segments[1]]
		if !ok {
			return mockSCIMErrorResponse(req, http.StatusNotFound, "group does not exist")
		}
		var group ScimGroup
		err := json.Unmarshal(body, &group)
		if err != nil {
			return mockSCIMErrorResponse(req, http.StatusBadRequest, "group payload is invalid JSON")
		}
		violation := m.validateGroupPayload(body, group)
		if violation != "" {
			m.violations = append(m.violations, violation)
			return mockSCIMErrorResponse(req, http.StatusBadRequest, violation)
		}
		m.nextID++
		group.ID = segments[1]
		group.Meta = newMockMeta("Group", "/Groups/"+group.ID, m.nextID)
		m.groups[group.ID] = group
		return mockSCIMJSONResponse(req, http.StatusOK, group)

	case http.MethodDelete:
		if len(segments) != 2 {
			return mockSCIMErrorResponse(req, http.StatusNotFound, "group resource path is invalid")
		}
		_, ok := m.groups[segments[1]]
		if !ok {
			return mockSCIMErrorResponse(req, http.StatusNotFound, "group does not exist")
		}
		delete(m.groups, segments[1])
		return mockSCIMNoContentResponse(req)

	default:
		return mockSCIMErrorResponse(req, http.StatusMethodNotAllowed, "group method is unsupported")
	}
}

func validateUserPayload(body []byte, user ScimUser) string {
	if violation := validateResourcePayload(body, user.ScimResourceData, scimUserSchema); violation != "" {
		return violation
	}
	if user.UserName == "" {
		return "SCIM user payload omitted userName"
	}
	return ""
}

func (m *mockSCIMTransport) validateGroupPayload(body []byte, group ScimGroup) string {
	if violation := validateResourcePayload(body, group.ScimResourceData, scimGroupSchema); violation != "" {
		return violation
	}
	if group.Display == "" {
		return "SCIM group payload omitted displayName"
	}
	for _, member := range group.Members {
		if _, ok := m.users[member.Value]; !ok {
			return fmt.Sprintf("SCIM group referenced unknown user %q", member.Value)
		}
	}
	return ""
}

func validateResourcePayload(body []byte, resource ScimResourceData, expectedSchema string) string {
	if resource.ExternalID == "" {
		return "SCIM resource payload omitted externalId"
	}
	if !slices.Contains(resource.Schemas, expectedSchema) {
		return fmt.Sprintf("SCIM resource payload omitted schema %q", expectedSchema)
	}

	var raw map[string]json.RawMessage
	err := json.Unmarshal(body, &raw)
	if err != nil {
		return "SCIM resource payload is invalid JSON"
	}
	_, ok := raw["id"]
	if ok {
		return "SCIM write payload included read-only id"
	}
	_, ok = raw["meta"]
	if ok {
		return "SCIM write payload included read-only meta"
	}

	return ""
}

func mockSCIMListResponse[T any](req *http.Request, resources []T, pageSize int) *http.Response {
	startIndex, err := strconv.Atoi(req.URL.Query().Get("startIndex"))
	if err != nil || startIndex < 1 {
		return mockSCIMErrorResponse(req, http.StatusBadRequest, "startIndex must be a one-based integer")
	}
	count, err := strconv.Atoi(req.URL.Query().Get("count"))
	if err != nil || count < 1 {
		return mockSCIMErrorResponse(req, http.StatusBadRequest, "count must be a positive integer")
	}
	if pageSize > 0 && pageSize < count {
		count = pageSize
	}

	start := min(startIndex-1, len(resources))
	end := min(start+count, len(resources))
	page := resources[start:end]
	return mockSCIMJSONResponse(req, http.StatusOK, struct {
		Schemas      []string `json:"schemas"`
		Resources    []T      `json:"Resources"`
		TotalResults int      `json:"totalResults"`
		StartIndex   int      `json:"startIndex"`
		ItemsPerPage int      `json:"itemsPerPage"`
	}{
		Schemas:      []string{scimListResponseSchema},
		Resources:    page,
		TotalResults: len(resources),
		StartIndex:   startIndex,
		ItemsPerPage: len(page),
	})
}

func mockSCIMJSONResponse(req *http.Request, status int, payload any) *http.Response {
	body, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	return &http.Response{
		StatusCode:    status,
		Header:        http.Header{"Content-Type": []string{mockSCIMRequestContentType}},
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}
}

func mockSCIMErrorResponse(req *http.Request, status int, detail string) *http.Response {
	return mockSCIMJSONResponse(req, status, struct {
		Schemas []string `json:"schemas"`
		Status  string   `json:"status"`
		Detail  string   `json:"detail"`
	}{
		Schemas: []string{scimErrorResponseSchema},
		Status:  strconv.Itoa(status),
		Detail:  detail,
	})
}

func mockSCIMNoContentResponse(req *http.Request) *http.Response {
	return &http.Response{
		StatusCode: http.StatusNoContent,
		Header:     make(http.Header),
		Body:       http.NoBody,
		Request:    req,
	}
}

func newMockMeta(resourceType, resourcePath string, version int) *ScimResourceMeta {
	now := time.Now().UTC()
	return &ScimResourceMeta{
		Location:     mockSCIMEndpoint + resourcePath,
		ResourceType: resourceType,
		Created:      now,
		LastModified: now,
		Version:      fmt.Sprintf(`W/"%d"`, version),
	}
}

func (m *mockSCIMTransport) seedUser(user ScimUser) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.ID] = user
}

func (m *mockSCIMTransport) seedGroup(group ScimGroup) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.groups[group.ID] = group
}

func (m *mockSCIMTransport) user(id string) *ScimUser {
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[id]
	if !ok {
		return nil
	}

	return &user
}

func (m *mockSCIMTransport) group(id string) *ScimGroup {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, ok := m.groups[id]
	if !ok {
		return nil
	}

	return &group
}

func (m *mockSCIMTransport) usersSnapshot() map[string]ScimUser {
	m.mu.Lock()
	defer m.mu.Unlock()
	return cloneMap(m.users)
}

func (m *mockSCIMTransport) groupsSnapshot() map[string]ScimGroup {
	m.mu.Lock()
	defer m.mu.Unlock()
	return cloneMap(m.groups)
}

func (m *mockSCIMTransport) requestsSnapshot() []mockSCIMRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.requests)
}

func (m *mockSCIMTransport) requireCompliant(t *testing.T) {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	require.Empty(t, m.violations)
}

func cloneMap[K comparable, V any](input map[K]V) map[K]V {
	result := make(map[K]V, len(input))
	maps.Copy(result, input)
	return result
}

func firstRequestIndex(requests []mockSCIMRequest, method, path string) int {
	for i, request := range requests {
		if request.method == method && request.path == path {
			return i
		}
	}
	return -1
}

func lastRequestIndex(requests []mockSCIMRequest, method, path string) int {
	for i := len(requests) - 1; i >= 0; i-- {
		if requests[i].method == method && requests[i].path == path {
			return i
		}
	}
	return -1
}

func countRequests(requests []mockSCIMRequest, method, path string) int {
	count := 0
	for _, request := range requests {
		if request.method == method && request.path == path {
			count++
		}
	}
	return count
}

func countRequestsWithPrefix(requests []mockSCIMRequest, method, pathPrefix string) int {
	count := 0
	for _, request := range requests {
		if request.method == method && strings.HasPrefix(request.path, pathPrefix) {
			count++
		}
	}
	return count
}

func countMutationRequests(requests []mockSCIMRequest) int {
	count := 0
	for _, request := range requests {
		if request.method == http.MethodPost || request.method == http.MethodPut || request.method == http.MethodDelete {
			count++
		}
	}
	return count
}

func queryValues(requests []mockSCIMRequest, method, path, key string) []string {
	values := make([]string, 0)
	for _, request := range requests {
		if request.method == method && request.path == path {
			values = append(values, request.query.Get(key))
		}
	}
	return values
}
