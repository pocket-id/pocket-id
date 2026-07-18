package devicelogin

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
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
}

func (f *fakeTokenService) GenerateAccessToken(user model.User, authenticationMethod string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.userID = user.ID
	f.authenticationMethod = authenticationMethod
	return "device-login-access-token", nil
}

func (f *fakeTokenService) generatedToken() (string, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.userID, f.authenticationMethod
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

func TestRequestLifecycle(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	deviceLoginService, signer, auditLog := newServiceForTest(db)

	user := model.User{
		Base:     model.Base{ID: "device-login-user"},
		Username: "device-login-user",
	}
	require.NoError(t, db.Create(&user).Error)

	request, deviceToken, err := deviceLoginService.Create(t.Context(), "192.0.2.10", "Mozilla/5.0 Chrome/125.0.0.0")
	require.NoError(t, err)
	require.Regexp(t, `^P[ABCDEFGHJKMNPQRSTUVWXYZ23456789]{7}$`, request.Code)
	require.Equal(t, utils.CreateSha256Hash(deviceToken), request.DeviceTokenHash)
	require.NotEqual(t, deviceToken, request.DeviceTokenHash)
	require.Equal(t, RequestStatusPending, request.Status)

	info, err := deviceLoginService.Inspect(t.Context(), strings.ToLower(request.Code))
	require.NoError(t, err)
	require.Equal(t, request.Code, info.UserCode)
	require.Equal(t, "192.0.2.10", info.IPAddress)
	require.Equal(t, "Parsed Mozilla/5.0 Chrome/125.0.0.0", info.Device)

	err = deviceLoginService.Decide(t.Context(), strings.ToLower(request.Code), "approve", user.ID, "fresh-proof")
	require.NoError(t, err)

	exchangedUser, accessToken, status, err := deviceLoginService.Exchange(t.Context(), request.ID, deviceToken, "198.51.100.20", "target-agent")
	require.NoError(t, err)
	require.Equal(t, RequestStatusApproved, status)
	require.Equal(t, user.ID, exchangedUser.ID)
	require.Equal(t, "device-login-access-token", accessToken)

	signedUserID, authenticationMethod := signer.generatedToken()
	require.Equal(t, user.ID, signedUserID)
	require.Equal(t, authenticationMethodOneTimePassword, authenticationMethod)

	var requestCount int64
	require.NoError(t, db.Model(&Request{}).Where("id = ?", request.ID).Count(&requestCount).Error)
	require.Zero(t, requestCount)

	entry := auditLog.lastEntry()
	require.Equal(t, model.AuditLogEventRemoteSignIn, entry.event)
	require.Equal(t, "198.51.100.20", entry.ipAddress)
	require.Equal(t, "target-agent", entry.userAgent)
	require.Equal(t, user.ID, entry.userID)
}

func TestPendingAndDeniedRequests(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	deviceLoginService, _, _ := newServiceForTest(db)

	request, deviceToken, err := deviceLoginService.Create(t.Context(), "", "requesting-agent")
	require.NoError(t, err)

	user, accessToken, status, err := deviceLoginService.Exchange(t.Context(), request.ID, deviceToken, "", "")
	require.NoError(t, err)
	require.Equal(t, RequestStatusPending, status)
	require.Empty(t, user.ID)
	require.Empty(t, accessToken)

	err = deviceLoginService.Decide(t.Context(), request.Code, "deny", "device-login-user", "")
	require.NoError(t, err)

	user, accessToken, status, err = deviceLoginService.Exchange(t.Context(), request.ID, deviceToken, "", "")
	var deniedError *common.DeviceLoginDeniedError
	require.ErrorAs(t, err, &deniedError)
	require.Equal(t, RequestStatusDenied, status)
	require.Empty(t, user.ID)
	require.Empty(t, accessToken)
}

func TestRejectsInvalidAndExpiredRequests(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	deviceLoginService, _, _ := newServiceForTest(db)

	request, deviceToken, err := deviceLoginService.Create(t.Context(), "", "requesting-agent")
	require.NoError(t, err)

	_, _, _, err = deviceLoginService.Exchange(t.Context(), request.ID, "wrong-token", "", "")
	var invalidError *common.DeviceLoginRequestInvalidOrExpiredError
	require.ErrorAs(t, err, &invalidError)

	require.NoError(t, db.Model(&request).Update("expires_at", datatype.DateTime(time.Now().Add(-time.Minute))).Error)
	_, err = deviceLoginService.Inspect(t.Context(), request.Code)
	require.ErrorAs(t, err, &invalidError)
	err = deviceLoginService.Decide(t.Context(), request.Code, "deny", "device-login-user", "")
	require.ErrorAs(t, err, &invalidError)
	_, _, _, err = deviceLoginService.Exchange(t.Context(), request.ID, deviceToken, "", "")
	require.ErrorAs(t, err, &invalidError)

}

func TestRejectsDisabledUserAtExchange(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	deviceLoginService, _, _ := newServiceForTest(db)

	user := model.User{
		Base:     model.Base{ID: "disabled-device-login-user"},
		Username: "disabled-device-login-user",
		Disabled: true,
	}
	require.NoError(t, db.Create(&user).Error)

	request, deviceToken, err := deviceLoginService.Create(t.Context(), "", "requesting-agent")
	require.NoError(t, err)
	require.NoError(t, deviceLoginService.Decide(t.Context(), request.Code, "approve", user.ID, "fresh-proof"))

	_, accessToken, _, err := deviceLoginService.Exchange(t.Context(), request.ID, deviceToken, "", "")
	var disabledError *common.UserDisabledError
	require.ErrorAs(t, err, &disabledError)
	require.Empty(t, accessToken)

	var remainingRequest Request
	require.NoError(t, db.First(&remainingRequest, "id = ?", request.ID).Error)
}

func TestApprovalRejectsMissingAndStaleReauthentication(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	deviceLoginService, _, _ := newServiceForTest(db)

	request, _, err := deviceLoginService.Create(t.Context(), "", "requesting-agent")
	require.NoError(t, err)

	err = deviceLoginService.Decide(t.Context(), request.Code, "approve", "device-login-user", "")
	var reauthenticationError *common.ReauthenticationRequiredError
	require.ErrorAs(t, err, &reauthenticationError)

	deviceLoginService.reauth = &fakeReauthenticationTokenConsumer{
		expectedValue: "stale-proof",
		createdAt:     time.Now().Add(-2 * time.Minute),
	}
	err = deviceLoginService.Decide(t.Context(), request.Code, "approve", "device-login-user", "stale-proof")
	require.ErrorAs(t, err, &reauthenticationError)

	var persistedRequest Request
	require.NoError(t, db.First(&persistedRequest, "id = ?", request.ID).Error)
	require.Equal(t, RequestStatusPending, persistedRequest.Status)
}

func TestExchangeIsSingleUse(t *testing.T) {
	db := testutils.NewConcurrentDatabaseForTest(t)
	deviceLoginService, _, _ := newServiceForTest(db)

	user := model.User{
		Base:     model.Base{ID: "single-use-device-login-user"},
		Username: "single-use-device-login-user",
	}
	require.NoError(t, db.Create(&user).Error)

	request, deviceToken, err := deviceLoginService.Create(t.Context(), "", "requesting-agent")
	require.NoError(t, err)
	require.NoError(t, deviceLoginService.Decide(t.Context(), request.Code, "approve", user.ID, "fresh-proof"))

	var waitGroup sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			_, _, _, exchangeErr := deviceLoginService.Exchange(t.Context(), request.ID, deviceToken, "", "")
			results <- exchangeErr
		}()
	}
	waitGroup.Wait()
	close(results)

	var successes int
	for result := range results {
		if result == nil {
			successes++
		}
	}
	require.Equal(t, 1, successes)
}

func newServiceForTest(db *gorm.DB) (*Service, *fakeTokenService, *fakeAuditLogger) {
	signer := &fakeTokenService{}
	auditLog := &fakeAuditLogger{}
	deviceLoginService := newService(Dependencies{
		DB:       db,
		Signer:   signer,
		AuditLog: auditLog,
		Reauth:   &fakeReauthenticationTokenConsumer{expectedValue: "fresh-proof"},
	})
	return deviceLoginService, signer, auditLog
}
