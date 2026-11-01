import { m } from '#lib/paraglide/messages.js';

export type ScopeLevel = 'none' | 'read' | 'write';
export type ScopePreset = 'full' | 'read' | 'custom';

export type ScopeResource = {
	id: string;
	label: () => string;
	description: () => string;
	admin: boolean;
	read?: string;
	// A resource without a read scope uses its write scope for reading and changing alike
	write?: string;
};

// Mirrors the scopes of the backend catalog in internal/authz/scope.go that API keys and OAuth clients can hold
export const scopeResources: ScopeResource[] = [
	{
		id: 'account',
		label: m.scope_account,
		description: m.scope_account_description,
		admin: false,
		read: 'account:read',
		write: 'account:write'
	},
	{
		id: 'passkeys',
		label: m.passkeys,
		description: m.scope_passkeys_description,
		admin: false,
		write: 'account:passkeys'
	},
	{
		id: 'apps',
		label: m.my_apps,
		description: m.scope_apps_description,
		admin: false,
		write: 'account:apps'
	},
	{
		id: 'activity',
		label: m.activity,
		description: m.scope_activity_description,
		admin: false,
		read: 'account:audit-logs'
	},
	{
		id: 'api-keys',
		label: m.api_keys,
		description: m.scope_api_keys_description,
		admin: false,
		write: 'account:api-keys'
	},
	{
		id: 'users',
		label: m.users,
		description: m.scope_users_description,
		admin: true,
		read: 'users:read',
		write: 'users:write'
	},
	{
		id: 'groups',
		label: m.groups,
		description: m.scope_groups_description,
		admin: true,
		read: 'groups:read',
		write: 'groups:write'
	},
	{
		id: 'oidc-clients',
		label: m.oidc_clients,
		description: m.scope_oidc_clients_description,
		admin: true,
		read: 'oidc-clients:read',
		write: 'oidc-clients:write'
	},
	{
		id: 'claim-mappings',
		label: m.claim_mappings,
		description: m.scope_claim_mappings_description,
		admin: true,
		read: 'oidc-claim-mapping-policies:read',
		write: 'oidc-claim-mapping-policies:write'
	},
	{
		id: 'apis',
		label: m.apis,
		description: m.scope_apis_description,
		admin: true,
		read: 'apis:read',
		write: 'apis:write'
	},
	{
		id: 'config',
		label: m.configuration,
		description: m.scope_config_description,
		admin: true,
		read: 'config:read',
		write: 'config:write'
	},
	{
		id: 'audit-log',
		label: m.audit_log,
		description: m.scope_audit_log_description,
		admin: true,
		read: 'audit-logs:read'
	}
];

export const resourcesFor = (isAdmin: boolean) =>
	scopeResources.filter((resource) => isAdmin || !resource.admin);

export const grantableScopes = (isAdmin: boolean) =>
	resourcesFor(isAdmin).flatMap((r) => [r.read, r.write].filter((s): s is string => !!s));

export const readOnlyScopes = (isAdmin: boolean) =>
	resourcesFor(isAdmin).flatMap((r) => (r.read ? [r.read] : []));

const sameScopes = (a: string[], b: string[]) =>
	a.length === b.length && b.every((scope) => a.includes(scope));

export function presetOf(scopes: string[], isAdmin: boolean): ScopePreset {
	if (sameScopes(scopes, grantableScopes(isAdmin))) return 'full';
	if (sameScopes(scopes, readOnlyScopes(isAdmin))) return 'read';
	return 'custom';
}

export function levelOf(resource: ScopeResource, scopes: string[]): ScopeLevel {
	if (resource.write && scopes.includes(resource.write)) return 'write';
	if (resource.read && scopes.includes(resource.read)) return 'read';
	return 'none';
}

// Write includes read, so a write level stores both scopes
export function withLevel(resource: ScopeResource, scopes: string[], level: ScopeLevel) {
	const rest = scopes.filter((scope) => scope !== resource.read && scope !== resource.write);
	const added =
		level === 'none' ? [] : level === 'read' ? [resource.read] : [resource.read, resource.write];
	return [...rest, ...added.filter((s): s is string => !!s)];
}

export function presetLabel(scopes: string[], isAdmin: boolean) {
	const preset = presetOf(scopes, isAdmin);
	if (preset === 'full') return m.full_access();
	if (preset === 'read') return m.read_only();
	const count = resourcesFor(isAdmin).filter((r) => levelOf(r, scopes) !== 'none').length;
	return m.custom_scope_count({ count });
}

// Labels a single scope, such as a permission of the built-in Pocket ID API, with the same words the API key picker uses
export function scopeLabel(scope: string): { name: string; description?: string } {
	const resource = scopeResources.find((r) => r.read === scope || r.write === scope);
	if (!resource) return { name: scope };
	// A resource with a single scope needs no level next to its name
	const level =
		!resource.read || !resource.write ? '' : scope === resource.read ? m.read() : m.write();
	return {
		name: level ? `${resource.label()} · ${level}` : resource.label(),
		description: resource.description()
	};
}
