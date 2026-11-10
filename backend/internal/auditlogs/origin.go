package auditlogs

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/authz"
	"github.com/pocket-id/pocket-id/backend/internal/mcp"
)

// Origin is who performed an audited action and where their request came from
type Origin struct {
	Principal authz.Principal
	IPAddress string
	UserAgent string

	// Actor is set for actions that no user or client performs, such as the LDAP sync's
	Actor ActorType
	// MCP marks an action taken with an MCP tool
	MCP bool
}

var (
	// LDAPSync is the origin of changes the LDAP sync makes
	LDAPSync = Origin{Actor: ActorLDAP}
	// System is the origin of changes Pocket ID makes on its own, such as a cleanup job
	System = Origin{Actor: ActorSystem}
)

func OriginFrom(c *gin.Context) Origin {
	return Origin{
		Principal: authz.PrincipalFrom(c),
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
}

func OriginFromMCP(call mcp.Call) Origin {
	return Origin{
		Principal: call.Principal,
		IPAddress: call.IPAddress,
		UserAgent: call.UserAgent,
		MCP:       true,
	}
}

var credentialKinds = map[authz.PrincipalKind]string{
	authz.KindSession:     "session",
	authz.KindAPIKey:      "api_key",
	authz.KindOAuthUser:   "oauth",
	authz.KindOAuthClient: "oauth",
}

// Record creates an entry about userID for an action that origin performed
// An empty userID records an action that concerns no user, such as creating a client
// A nil tx writes the entry on its own, for callers that keep no database transaction
func (s *service) Record(ctx context.Context, event Event, origin Origin, userID string, data Data, tx *gorm.DB) (AuditLog, bool) {
	if tx == nil {
		tx = s.db
	}
	if data == nil {
		data = Data{}
	}
	entry := AuditLog{Event: event, UserAgent: origin.UserAgent, UserID: userID, Data: data}
	setActor(ctx, tx, origin, &entry)
	addCredential(ctx, tx, origin, data)
	return s.insert(ctx, entry, origin.IPAddress, tx)
}

// RecordCLI writes an entry for an action taken with the command line, which runs no service and has no client IP
func RecordCLI(ctx context.Context, db *gorm.DB, event Event, userID string, data Data) {
	s := &service{db: db, ipLocator: noLocation{}}
	s.Record(ctx, event, Origin{Actor: ActorCLI}, userID, data, nil)
}

type noLocation struct{}

func (noLocation) GetLocationByIP(context.Context, string) (string, string, error) {
	return "", "", nil
}

func setActor(ctx context.Context, tx *gorm.DB, origin Origin, entry *AuditLog) {
	principal := origin.Principal
	switch {
	case origin.Actor != "":
		entry.ActorType = origin.Actor
	case principal.Kind == authz.KindOAuthClient:
		entry.ActorType, entry.ActorID = ActorClient, principal.ClientID
		entry.ActorName = lookupName(ctx, tx, "oidc_clients", "name", principal.ClientID)
	case principal.UserID != "":
		entry.ActorType, entry.ActorID = ActorUser, principal.UserID
		entry.ActorName = lookupName(ctx, tx, "users", "username", principal.UserID)
	}
}

// addCredential records what the actor authenticated with, naming the API key or the app that holds the OAuth token
// A client acting as itself is already the actor, so its name isn't repeated
func addCredential(ctx context.Context, tx *gorm.DB, origin Origin, data Data) {
	principal := origin.Principal
	via := credentialKinds[principal.Kind]
	if origin.MCP {
		via = "mcp"
	}
	if via != "" {
		data["via"] = via
	}

	switch principal.Kind {
	case authz.KindAPIKey:
		data["viaName"] = lookupName(ctx, tx, "api_keys", "name", principal.APIKeyID)
	case authz.KindOAuthUser:
		data["viaName"] = lookupName(ctx, tx, "oidc_clients", "name", principal.ClientID)
	case authz.KindSession, authz.KindOAuthClient:
	}
}

// lookupName logs instead of failing, so a name that can't be loaded never blocks the audited action
func lookupName(ctx context.Context, tx *gorm.DB, table, column, id string) string {
	var name string
	err := tx.WithContext(ctx).Table(table).Select(column).Where("id = ?", id).Limit(1).Scan(&name).Error
	if err != nil {
		slog.ErrorContext(ctx, "Failed to load a name for the audit log", slog.String("table", table), slog.Any("error", err))
	}
	return name
}
