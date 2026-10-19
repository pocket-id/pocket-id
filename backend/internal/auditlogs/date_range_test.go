package auditlogs

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
	"github.com/pocket-id/pocket-id/backend/internal/utils"
	testutils "github.com/pocket-id/pocket-id/backend/internal/utils/testing"
)

func TestListAuditLogsByDateRange(t *testing.T) {
	db := testutils.NewDatabaseForTest(t)
	s := newService(db, nil, signInLocationResolver{}, nil, browserTokenStub{})
	require.NoError(t, db.Create(&model.User{Base: model.Base{ID: "alice"}, Username: "alice"}).Error)

	day := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	for i, at := range []time.Time{day.Add(-time.Second), day, day.Add(12 * time.Hour), day.Add(24 * time.Hour)} {
		id := string(rune('a' + i))
		require.NoError(t, db.Create(&AuditLog{Base: model.Base{ID: id}, UserID: "alice", Event: EventSignIn}).Error)
		// Creating always stamps the current time, so the test time is set afterwards
		require.NoError(t, db.Model(&AuditLog{}).Where("id = ?", id).Update("created_at", datatype.DateTime(at)).Error)
	}

	ids := func(logs []AuditLog) []string {
		out := make([]string, len(logs))
		for i, l := range logs {
			out[i] = l.ID
		}
		return out
	}
	oneDay := DateRange{From: day, To: day.Add(24 * time.Hour)}

	all, _, err := s.ListAllAuditLogs(t.Context(), oneDay, utils.ListRequestOptions{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"b", "c"}, ids(all))

	own, _, err := s.ListAuditLogsForUser(t.Context(), "alice", DateRange{From: day.Add(time.Hour)}, utils.ListRequestOptions{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"c", "d"}, ids(own))
}

func TestParseDateRange(t *testing.T) {
	parse := func(query string) (DateRange, error) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequestWithContext(t.Context(), "GET", "/api/audit-logs?"+query, nil)
		return parseDateRange(c)
	}

	r, err := parse("from=2026-10-10T00:00:00%2B02:00")
	require.NoError(t, err)
	require.True(t, r.From.Equal(time.Date(2026, 10, 9, 22, 0, 0, 0, time.UTC)))
	require.True(t, r.To.IsZero())

	_, err = parse("to=yesterday")
	require.Error(t, err)
}
