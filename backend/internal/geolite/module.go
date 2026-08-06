package geolite

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

type Dependencies struct {
	HTTPClient *http.Client

	// DBPath is where the GeoLite2 City database is kept
	// It is local to each replica: the database is a cache of a public artifact rather than state, so it doesn't need to be shared, though pointing several replicas at the same mount works too
	DBPath string
	// DownloadURL is the URL the GeoLite2 City database is downloaded from
	// When it contains a "%s" placeholder, it is replaced with LicenseKey
	DownloadURL string
	// LicenseKey is the MaxMind license key
	LicenseKey string
}

type Module struct {
	service   *Service
	refresher *refresher
}

func New(ctx context.Context, deps Dependencies) (*Module, error) {
	if deps.DBPath == "" {
		return nil, errors.New("the GeoLite database path is empty")
	}

	log := slog.With(slog.String("scope", "geolite"))
	service := newService(log, deps.DBPath)

	// Map the database that is already on disk, if any, so lookups work before the first refresh
	// A database that can't be read isn't fatal: lookups just return no location until a readable one shows up
	err := service.load(ctx)
	if err != nil {
		log.WarnContext(ctx, "Failed to load the GeoLite2 City database", slog.String("path", deps.DBPath), slog.Any("error", err))
	}

	disabled := deps.LicenseKey == "" && deps.DownloadURL == common.MaxMindGeoLiteCityUrl
	if disabled {
		// Warn the user, and disable the periodic refresh
		// The database can still be supplied by hand at DBPath, which is what air-gapped deployments do
		log.Warn("MAXMIND_LICENSE_KEY environment variable is empty: the GeoLite2 City database won't be updated")
	}

	return &Module{
		service: service,
		refresher: &refresher{
			log:         log,
			service:     service,
			httpClient:  deps.HTTPClient,
			downloadURL: deps.DownloadURL,
			licenseKey:  deps.LicenseKey,
			disabled:    disabled,
		},
	}, nil
}

// GetLocationByIP returns the country and city of the given IP address
func (m *Module) GetLocationByIP(ctx context.Context, ipAddress string) (country string, city string, err error) {
	return m.service.GetLocationByIP(ctx, ipAddress)
}

// Run keeps the GeoLite2 City database up-to-date until the context is canceled
// It satisfies servicerunner.Service
func (m *Module) Run(ctx context.Context) error {
	return m.refresher.Run(ctx)
}
