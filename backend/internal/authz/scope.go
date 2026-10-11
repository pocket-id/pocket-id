package authz

// Scope is a permission that a principal must hold to call a route
// Keys follow the resource:action pattern and are valid RFC 6749 scope tokens so they can later appear in API key records and OAuth access tokens unchanged
type Scope string

// Account scopes cover the caller's own account and are held by every signed-in user
const (
	AccountRead           Scope = "account:read"
	AccountWrite          Scope = "account:write"
	AccountPasskeys       Scope = "account:passkeys"
	AccountAPIKeys        Scope = "account:api-keys"
	AccountApps           Scope = "account:apps"
	AccountAuditLogs      Scope = "account:audit-logs"
	AccountSession        Scope = "account:session"
	AccountPasskeysEnroll Scope = "account:passkeys:enroll"
	AccountAPIKeysCreate  Scope = "account:api-keys:create"
)

// Admin scopes cover other users' data and the instance configuration
const (
	UsersRead        Scope = "users:read"
	UsersWrite       Scope = "users:write"
	GroupsRead       Scope = "groups:read"
	GroupsWrite      Scope = "groups:write"
	OidcClientsRead  Scope = "oidc-clients:read"
	OidcClientsWrite Scope = "oidc-clients:write"
	APIsRead         Scope = "apis:read"
	APIsWrite        Scope = "apis:write"
	ConfigRead       Scope = "config:read"
	ConfigWrite      Scope = "config:write"
	AuditLogsRead    Scope = "audit-logs:read"
)

// Category groups scopes by whose data they reach
type Category int

const (
	// CategoryAccount scopes act on the caller's own account
	CategoryAccount Category = iota + 1
	// CategoryAdmin scopes act on other users or on the instance
	CategoryAdmin
)

// PrincipalKind identifies the kind of credential a principal authenticated with
// Kinds are bit flags so a scope can list every kind that may hold it
type PrincipalKind uint8

const (
	// KindSession is a browser session established by signing in to Pocket ID
	KindSession PrincipalKind = 1 << iota
	// KindAPIKey is a personal API key sent in the X-API-Key header
	KindAPIKey
	// KindOAuthUser is an OAuth access token issued to a client acting on behalf of a user
	KindOAuthUser
	// KindOAuthClient is an OAuth access token issued to a client acting as itself through the client credentials grant
	KindOAuthClient
)

// delegated lists the kinds that act for a user, which is every kind except a client acting as itself
const delegated = KindSession | KindAPIKey | KindOAuthUser

type definition struct {
	scope       Scope
	category    Category
	grantableTo PrincipalKind
}

// catalog is the complete list of scopes
// grantableTo restricts which credential kinds can ever hold a scope, independent of the user's role
// Session-only scopes guard actions that must never be reachable with a long-lived or third-party credential, such as enrolling passkeys or minting API keys
var catalog = []definition{
	{AccountRead, CategoryAccount, delegated},
	{AccountWrite, CategoryAccount, delegated},
	{AccountPasskeys, CategoryAccount, delegated},
	{AccountAPIKeys, CategoryAccount, delegated},
	{AccountApps, CategoryAccount, delegated},
	{AccountAuditLogs, CategoryAccount, delegated},
	{AccountSession, CategoryAccount, KindSession},
	{AccountPasskeysEnroll, CategoryAccount, KindSession},
	{AccountAPIKeysCreate, CategoryAccount, KindSession},

	{UsersRead, CategoryAdmin, delegated},
	{UsersWrite, CategoryAdmin, delegated},
	{GroupsRead, CategoryAdmin, delegated},
	{GroupsWrite, CategoryAdmin, delegated},
	{OidcClientsRead, CategoryAdmin, delegated},
	{OidcClientsWrite, CategoryAdmin, delegated},
	{APIsRead, CategoryAdmin, delegated},
	{APIsWrite, CategoryAdmin, delegated},
	{ConfigRead, CategoryAdmin, delegated},
	{ConfigWrite, CategoryAdmin, delegated},
	{AuditLogsRead, CategoryAdmin, delegated},
}

var definitions = indexCatalog(catalog)

func indexCatalog(entries []definition) map[Scope]definition {
	index := make(map[Scope]definition, len(entries))
	for _, entry := range entries {
		index[entry.scope] = entry
	}
	return index
}

// Known reports whether the scope is part of the catalog
func (s Scope) Known() bool {
	_, ok := definitions[s]
	return ok
}

// GrantableTo reports whether a principal of the given kind can ever hold the scope
func (s Scope) GrantableTo(kind PrincipalKind) bool {
	return definitions[s].grantableTo&kind != 0
}

// ScopeSet is an unordered set of scopes
type ScopeSet map[Scope]struct{}

// NewScopeSet creates a set containing the given scopes
func NewScopeSet(scopes ...Scope) ScopeSet {
	set := make(ScopeSet, len(scopes))
	for _, scope := range scopes {
		set[scope] = struct{}{}
	}
	return set
}

// Has reports whether the set contains the scope
func (s ScopeSet) Has(scope Scope) bool {
	_, ok := s[scope]
	return ok
}

// UserScopes returns the scopes a user holds when authenticated with a credential of the given kind
// The admin flag stands in for roles: admins hold every scope and other users hold the account scopes
func UserScopes(isAdmin bool, kind PrincipalKind) ScopeSet {
	set := make(ScopeSet, len(catalog))
	for _, entry := range catalog {
		if entry.grantableTo&kind == 0 {
			continue
		}
		if entry.category == CategoryAdmin && !isAdmin {
			continue
		}
		set[entry.scope] = struct{}{}
	}
	return set
}
