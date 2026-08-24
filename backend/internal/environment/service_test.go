package environment

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

func TestShowSQLiteStorageWarning(t *testing.T) {
	tests := []struct {
		name                        string
		sqliteOnNetworkedFilesystem bool
		dismissed                   bool
		want                        bool
	}{
		{name: "warns when the database is on a networked filesystem", sqliteOnNetworkedFilesystem: true, want: true},
		{name: "does not warn when the database is on a local filesystem", sqliteOnNetworkedFilesystem: false, want: false},
		{name: "does not warn when the warning has been dismissed", sqliteOnNetworkedFilesystem: true, dismissed: true, want: false},
		{name: "does not warn when dismissed and the database is on a local filesystem", sqliteOnNetworkedFilesystem: false, dismissed: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prevConfig := common.EnvConfig
			t.Cleanup(func() {
				common.EnvConfig = prevConfig
			})
			common.EnvConfig.DismissSQLiteStorageWarning = common.DismissSQLiteStorageWarningConfig(tt.dismissed)

			svc := newService(Dependencies{SQLiteOnNetworkedFilesystem: tt.sqliteOnNetworkedFilesystem})
			assert.Equal(t, tt.want, svc.ShowSQLiteStorageWarning())
		})
	}
}
