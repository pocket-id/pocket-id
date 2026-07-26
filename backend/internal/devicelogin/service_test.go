//go:build exclude_frontend && unit

package devicelogin

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/italypaleale/francis/actor"
	"github.com/italypaleale/francis/host/local"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

const (
	testActorIDKey      = "0123456789abcdef0123456789abcdef"
	testSessionDuration = time.Hour
)

type fakeReauthenticationTokenConsumer struct {
	expectedValue string
	createdAt     time.Time
}

func (f *fakeReauthenticationTokenConsumer) ConsumeReauthenticationToken(_ context.Context, _ *gorm.DB, token string, _ string) (time.Time, error) {
	if token != f.expectedValue {
		return time.Time{}, &common.ReauthenticationRequiredError{}
	}
	if !f.createdAt.IsZero() {
		return f.createdAt, nil
	}
	return time.Now(), nil
}

type fakeTokenService struct {
	mu                   sync.Mutex
	userID               string
	authenticationMethod string
	sessionDuration      time.Duration
	generated            int
}

func (f *fakeTokenService) GenerateAccessToken(user model.User, authenticationMethod string, sessionDuration time.Duration) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.userID = user.ID
	f.authenticationMethod = authenticationMethod
	f.sessionDuration = sessionDuration
	f.generated++
	return "device-login-access-token", nil
}

func (f *fakeTokenService) generatedToken() (string, string, time.Duration, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.userID, f.authenticationMethod, f.sessionDuration, f.generated
}

type auditEntry struct {
	event     model.AuditLogEvent
	ipAddress string
	userAgent string
	userID    string
}

type fakeAuditLogger struct {
	mu      sync.Mutex
	entries []auditEntry
}

func (f *fakeAuditLogger) Create(_ context.Context, event model.AuditLogEvent, ipAddress, userAgent, userID string, _ model.AuditLogData, _ *gorm.DB) (model.AuditLog, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, auditEntry{event: event, ipAddress: ipAddress, userAgent: userAgent, userID: userID})
	return model.AuditLog{}, true
}

func (f *fakeAuditLogger) DeviceStringFromUserAgent(userAgent string) string {
	return "Parsed " + userAgent
}

func (f *fakeAuditLogger) lastEntry() auditEntry {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.entries[len(f.entries)-1]
}

func (f *fakeAuditLogger) entryCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.entries)
}

type serviceFixture struct {
	service  *Service
	actors   *actor.Service
	signer   *fakeTokenService
	auditLog *fakeAuditLogger
	reauth   *fakeReauthenticationTokenConsumer
}

func TestRequestLifecycle(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	fixture := newServiceFixture(t, db)

	user := model.User{
		Base:     model.Base{ID: "device-login-user"},
		Username: "device-login-user",
	}
	require.NoError(t, db.Create(&user).Error)

	request, deviceToken, err := fixture.service.Create(t.Context(), "192.0.2.10", "Mozilla/5.0 Chrome/125.0.0.0")
	require.NoError(t, err)
	require.Regexp(t, `^P[ABCDEFGHJKMNPQRSTUVWXYZ23456789]{7}$`, request.Code)
	require.Regexp(t, `^[a-f0-9]{64}$`, request.ID)
	require.Equal(t, RequestStatusPending, request.Status)

	state := getRequestActorState(t, fixture.actors, request.ID)
	require.Equal(t, utils.CreateSha256Hash(deviceToken), state.DeviceTokenHash)
	require.NotEqual(t, deviceToken, state.DeviceTokenHash)
	require.Equal(t, RequestStatusPending, state.Status)

	info, err := fixture.service.Inspect(t.Context(), strings.ToLower(request.Code))
	require.NoError(t, err)
	require.Equal(t, request.Code, info.UserCode)
	require.Equal(t, "192.0.2.10", info.IPAddress)
	require.Equal(t, "Parsed Mozilla/5.0 Chrome/125.0.0.0", info.Device)

	err = fixture.service.Decide(t.Context(), strings.ToLower(request.Code), "approve", user.ID, "fresh-proof")
	require.NoError(t, err)

	exchangedUser, accessToken, status, err := fixture.service.Exchange(t.Context(), request.ID, deviceToken, "198.51.100.20", "target-agent", testSessionDuration)
	require.NoError(t, err)
	require.Equal(t, RequestStatusApproved, status)
	require.Equal(t, user.ID, exchangedUser.ID)
	require.Equal(t, "device-login-access-token", accessToken)

	signedUserID, authenticationMethod, sessionDuration, generated := fixture.signer.generatedToken()
	require.Equal(t, user.ID, signedUserID)
	require.Equal(t, authenticationMethodOneTimePassword, authenticationMethod)
	require.Equal(t, testSessionDuration, sessionDuration)
	require.Equal(t, 1, generated)
	requireRequestActorStateDeleted(t, fixture.actors, request.ID)

	entry := fixture.auditLog.lastEntry()
	require.Equal(t, model.AuditLogEventRemoteSignIn, entry.event)
	require.Equal(t, "198.51.100.20", entry.ipAddress)
	require.Equal(t, "target-agent", entry.userAgent)
	require.Equal(t, user.ID, entry.userID)

	_, _, _, err = fixture.service.Exchange(t.Context(), request.ID, deviceToken, "", "", testSessionDuration)
	assertInvalidRequestError(t, err)

	_, _, _, err = fixture.service.Exchange(t.Context(), request.ID, "wrong-token", "", "", testSessionDuration)
	assertInvalidRequestError(t, err)
	_, _, _, generated = fixture.signer.generatedToken()
	require.Equal(t, 1, generated)
	require.Equal(t, 1, fixture.auditLog.entryCount())
}

func TestPendingAndDeniedRequests(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	fixture := newServiceFixture(t, db)

	request, deviceToken, err := fixture.service.Create(t.Context(), "", "requesting-agent")
	require.NoError(t, err)

	result, err := fixture.service.peek(t.Context(), request.ID, requestActorMethodPoll, requestActorPollInput{
		DeviceTokenHash: utils.CreateSha256Hash(deviceToken),
	})
	require.NoError(t, err)
	require.Equal(t, RequestStatusPending, result.Status)

	err = fixture.service.Decide(t.Context(), request.Code, "deny", "device-login-user", "")
	require.NoError(t, err)

	user, accessToken, status, err := fixture.service.Exchange(t.Context(), request.ID, deviceToken, "", "", testSessionDuration)
	var deniedError *common.DeviceLoginDeniedError
	require.ErrorAs(t, err, &deniedError)
	require.Equal(t, RequestStatusDenied, status)
	require.Empty(t, user.ID)
	require.Empty(t, accessToken)
}

func TestPendingExchangeObservesDecisionDuringLongPoll(t *testing.T) {
	db := testutils.NewConcurrentDatabaseForTest(t)
	fixture := newServiceFixture(t, db)

	request, deviceToken, err := fixture.service.Create(t.Context(), "", "requesting-agent")
	require.NoError(t, err)

	type exchangeOutcome struct {
		status RequestStatus
		err    error
	}
	result := make(chan exchangeOutcome, 1)
	go func() {
		_, _, status, exchangeErr := fixture.service.Exchange(t.Context(), request.ID, deviceToken, "", "", testSessionDuration)
		result <- exchangeOutcome{status: status, err: exchangeErr}
	}()

	require.NoError(t, fixture.service.Decide(t.Context(), request.Code, "deny", "device-login-user", ""))

	select {
	case outcome := <-result:
		var deniedError *common.DeviceLoginDeniedError
		require.ErrorAs(t, outcome.err, &deniedError)
		require.Equal(t, RequestStatusDenied, outcome.status)
	case <-time.After(2 * time.Second):
		t.Fatal("exchange did not observe the actor decision")
	}
}

func TestRejectsInvalidAndExpiredRequestsWhileActorIsActive(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	fixture := newServiceFixture(t, db)

	request, deviceToken, err := fixture.service.Create(t.Context(), "", "requesting-agent")
	require.NoError(t, err)

	unknownRequestID := strings.Repeat("a", 64)
	_, _, _, err = fixture.service.Exchange(t.Context(), unknownRequestID, "device-token", "", "", testSessionDuration)
	assertInvalidRequestError(t, err)
	_, err = fixture.actors.Peek(t.Context(), requestActorType, unknownRequestID, requestActorMethodInspect, nil, actor.WithInvokeActiveOnly())
	require.ErrorIs(t, err, actor.ErrActorNotActive)

	_, _, _, err = fixture.service.Exchange(t.Context(), request.ID, "wrong-token", "", "", testSessionDuration)
	assertInvalidRequestError(t, err)

	state := getRequestActorState(t, fixture.actors, request.ID)
	state.ExpiresAt = time.Now().Add(-time.Second)
	require.NoError(t, fixture.actors.Halt(requestActorType, request.ID))
	require.NoError(t, fixture.actors.SetState(t.Context(), requestActorType, request.ID, state, nil))
	_, err = fixture.service.Inspect(t.Context(), request.Code)
	assertInvalidRequestError(t, err)
	err = fixture.service.Decide(t.Context(), request.Code, "deny", "device-login-user", "")
	assertInvalidRequestError(t, err)
	_, _, _, err = fixture.service.Exchange(t.Context(), request.ID, deviceToken, "", "", testSessionDuration)
	assertInvalidRequestError(t, err)
}

func TestRejectsDisabledUserAtExchange(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	fixture := newServiceFixture(t, db)

	user := model.User{
		Base:     model.Base{ID: "disabled-device-login-user"},
		Username: "disabled-device-login-user",
		Disabled: true,
	}
	require.NoError(t, db.Create(&user).Error)

	request, deviceToken, err := fixture.service.Create(t.Context(), "", "requesting-agent")
	require.NoError(t, err)
	require.NoError(t, fixture.service.Decide(t.Context(), request.Code, "approve", user.ID, "fresh-proof"))

	_, accessToken, _, err := fixture.service.Exchange(t.Context(), request.ID, deviceToken, "", "", testSessionDuration)
	var disabledError *common.UserDisabledError
	require.ErrorAs(t, err, &disabledError)
	require.Empty(t, accessToken)
	require.Equal(t, RequestStatusApproved, getRequestActorState(t, fixture.actors, request.ID).Status)
}

func TestApprovalRejectsMissingAndStaleReauthentication(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	fixture := newServiceFixture(t, db)

	request, _, err := fixture.service.Create(t.Context(), "", "requesting-agent")
	require.NoError(t, err)

	err = fixture.service.Decide(t.Context(), request.Code, "approve", "device-login-user", "")
	var reauthenticationError *common.ReauthenticationRequiredError
	require.ErrorAs(t, err, &reauthenticationError)

	fixture.reauth.expectedValue = "stale-proof"
	fixture.reauth.createdAt = time.Now().Add(-2 * time.Minute)
	err = fixture.service.Decide(t.Context(), request.Code, "approve", "device-login-user", "stale-proof")
	require.ErrorAs(t, err, &reauthenticationError)
	require.Equal(t, RequestStatusPending, getRequestActorState(t, fixture.actors, request.ID).Status)
}

func TestConcurrentExchangeAllowsOnlyOneSuccess(t *testing.T) {
	db := testutils.NewConcurrentDatabaseForTest(t)
	fixture := newServiceFixture(t, db)

	user := model.User{
		Base:     model.Base{ID: "single-use-device-login-user"},
		Username: "single-use-device-login-user",
	}
	require.NoError(t, db.Create(&user).Error)

	request, deviceToken, err := fixture.service.Create(t.Context(), "", "requesting-agent")
	require.NoError(t, err)
	require.NoError(t, fixture.service.Decide(t.Context(), request.Code, "approve", user.ID, "fresh-proof"))

	var waitGroup sync.WaitGroup
	type exchangeResult struct {
		token string
		err   error
	}
	results := make(chan exchangeResult, 2)
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			_, token, _, exchangeErr := fixture.service.Exchange(t.Context(), request.ID, deviceToken, "", "", testSessionDuration)
			results <- exchangeResult{token: token, err: exchangeErr}
		}()
	}
	waitGroup.Wait()
	close(results)

	var successfulTokens []string
	var invalidExchanges int
	for result := range results {
		if result.err == nil {
			successfulTokens = append(successfulTokens, result.token)
			continue
		}
		var invalidRequestError *common.DeviceLoginRequestInvalidOrExpiredError
		require.ErrorAs(t, result.err, &invalidRequestError)
		invalidExchanges++
	}
	require.Equal(t, []string{"device-login-access-token"}, successfulTokens)
	require.Equal(t, 1, invalidExchanges)
	require.Equal(t, 1, fixture.auditLog.entryCount())
	_, _, _, generated := fixture.signer.generatedToken()
	require.Equal(t, 1, generated)
	requireRequestActorStateDeleted(t, fixture.actors, request.ID)
}

func TestCreateCollisionPreservesOriginalActorState(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	fixture := newServiceFixture(t, db)

	request, deviceToken, err := fixture.service.Create(t.Context(), "192.0.2.1", "original-agent")
	require.NoError(t, err)
	original := getRequestActorState(t, fixture.actors, request.ID)

	result, err := fixture.service.invoke(t.Context(), request.ID, requestActorMethodCreate, requestActorCreateInput{
		Code:            request.Code,
		DeviceTokenHash: utils.CreateSha256Hash("different-token"),
		IPAddress:       "198.51.100.1",
		UserAgent:       "replacement-agent",
	})
	require.NoError(t, err)
	require.Equal(t, requestActorResultCollision, result.Code)
	require.Equal(t, original, getRequestActorState(t, fixture.actors, request.ID))
	require.NotEqual(t, deviceToken, original.DeviceTokenHash)
}

func TestRequestStateSurvivesActorHostRestart(t *testing.T) {
	db := testutils.NewConcurrentDatabaseForTest(t)
	deps := persistentTestDependencies(db)

	firstModule, stopFirst := startPersistentDeviceLoginHost(t, db, deps)
	request, deviceToken, err := firstModule.service.Create(t.Context(), "192.0.2.1", "persistent-agent")
	require.NoError(t, err)
	stopFirst()

	secondModule, stopSecond := startPersistentDeviceLoginHost(t, db, deps)
	defer stopSecond()
	info, err := secondModule.service.Inspect(t.Context(), request.Code)
	require.NoError(t, err)
	require.Equal(t, "persistent-agent", strings.TrimPrefix(info.Device, "Parsed "))
	require.NoError(t, secondModule.service.Decide(t.Context(), request.Code, "deny", "device-login-user", ""))
	_, _, status, err := secondModule.service.Exchange(t.Context(), request.ID, deviceToken, "", "", testSessionDuration)
	var deniedError *common.DeviceLoginDeniedError
	require.ErrorAs(t, err, &deniedError)
	require.Equal(t, RequestStatusDenied, status)
}

func TestCompletedExchangeIsInvalidAfterActorHostRestart(t *testing.T) {
	db := testutils.NewConcurrentDatabaseForTest(t)
	deps := persistentTestDependencies(db)

	user := model.User{
		Base:     model.Base{ID: "completed-exchange-user"},
		Username: "completed-exchange-user",
	}
	require.NoError(t, db.Create(&user).Error)

	firstModule, stopFirst := startPersistentDeviceLoginHost(t, db, deps)
	request, deviceToken, err := firstModule.service.Create(t.Context(), "", "persistent-agent")
	require.NoError(t, err)
	require.NoError(t, firstModule.service.Decide(t.Context(), request.Code, "approve", user.ID, "fresh-proof"))
	_, _, firstStatus, err := firstModule.service.Exchange(t.Context(), request.ID, deviceToken, "", "", testSessionDuration)
	require.NoError(t, err)
	require.Equal(t, RequestStatusApproved, firstStatus)
	stopFirst()

	secondModule, stopSecond := startPersistentDeviceLoginHost(t, db, deps)
	defer stopSecond()
	_, _, _, err = secondModule.service.Exchange(t.Context(), request.ID, deviceToken, "", "", testSessionDuration)
	assertInvalidRequestError(t, err)

	_, _, _, generated := deps.Signer.(*fakeTokenService).generatedToken()
	require.Equal(t, 1, generated)
	require.Equal(t, 1, deps.AuditLog.(*fakeAuditLogger).entryCount())
}

func TestPendingExchangeStopsWhenRequestIsCanceled(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	fixture := newServiceFixture(t, db)

	request, deviceToken, err := fixture.service.Create(t.Context(), "", "requesting-agent")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	started := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		close(started)
		_, _, _, exchangeErr := fixture.service.Exchange(ctx, request.ID, deviceToken, "", "", testSessionDuration)
		result <- exchangeErr
	}()

	<-started
	cancel()

	select {
	case exchangeErr := <-result:
		require.ErrorIs(t, exchangeErr, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("canceled exchange did not stop")
	}
}

func newServiceFixture(t *testing.T, db *gorm.DB) serviceFixture {
	t.Helper()
	signer := &fakeTokenService{}
	auditLog := &fakeAuditLogger{}
	reauth := &fakeReauthenticationTokenConsumer{expectedValue: "fresh-proof"}
	var module *Module
	host := testutils.NewActorHostForTest(t, func(t *testing.T, host *local.Host) {
		var err error
		module, err = New(Dependencies{
			DB:         db,
			Actors:     host,
			ActorIDKey: []byte(testActorIDKey),
			Signer:     signer,
			AuditLog:   auditLog,
			Reauth:     reauth,
		})
		require.NoError(t, err)
	})

	return serviceFixture{
		service:  module.service,
		actors:   host.Service(),
		signer:   signer,
		auditLog: auditLog,
		reauth:   reauth,
	}
}

func getRequestActorState(t *testing.T, actors *actor.Service, actorID string) requestActorState {
	t.Helper()
	var state requestActorState
	require.NoError(t, actors.GetState(t.Context(), requestActorType, actorID, &state))
	return state
}

func requireRequestActorStateDeleted(t *testing.T, actors *actor.Service, actorID string) {
	t.Helper()
	var state requestActorState
	require.ErrorIs(t, actors.GetState(t.Context(), requestActorType, actorID, &state), actor.ErrStateNotFound)
}

func assertInvalidRequestError(t *testing.T, err error) {
	t.Helper()
	var invalidError *common.DeviceLoginRequestInvalidOrExpiredError
	require.ErrorAs(t, err, &invalidError)
}

func persistentTestDependencies(db *gorm.DB) Dependencies {
	return Dependencies{
		DB:         db,
		ActorIDKey: []byte(testActorIDKey),
		Signer:     &fakeTokenService{},
		AuditLog:   &fakeAuditLogger{},
		Reauth:     &fakeReauthenticationTokenConsumer{expectedValue: "fresh-proof"},
	}
}

func startPersistentDeviceLoginHost(t *testing.T, db *gorm.DB, deps Dependencies) (*Module, func()) {
	t.Helper()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_, err = sqlDB.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	host, err := local.NewHost(
		local.WithAddress(freeLoopbackAddress(t)),
		local.WithRuntimePSKs([]byte("pocket-id-device-login-test-host-psk")),
		local.WithSQLiteProvider(local.SQLiteProviderOptions{DB: sqlDB}),
		local.WithShutdownGracePeriod(time.Second),
	)
	require.NoError(t, err)

	deps.Actors = host
	module, err := New(deps)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- host.Run(ctx)
	}()

	select {
	case <-host.Ready():
	case runErr := <-errCh:
		t.Fatalf("persistent actor host stopped before becoming ready: %v", runErr)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for persistent actor host")
	}

	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			cancel()
			runErr := <-errCh
			if runErr != nil && !errors.Is(runErr, context.Canceled) {
				require.NoError(t, runErr)
			}
		})
	}
	t.Cleanup(stop)
	return module, stop
}

func freeLoopbackAddress(t *testing.T) string {
	t.Helper()
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	return address
}
