# Suggested title

🚀 Feature: First Class Agent Identies

## Feature description

Add an AI Agent dedicated onboarding and authentication flow separate from the native passkey ceremony.

Pocket ID already provides the surrounding identity lifecycle: administrators can create users, issue one-time login links, assign groups and administrator status, and expose the resulting identity to applications through OIDC. The remaining gap is durable authentication for a `user` identity which is an agent only. 

The proposed user flow is:

```text
Administrator creates a normal Pocket ID user
  -> turns on Agent authentication 
  -> issues the existing one-time login code
  -> agent uses the code and registers a runtime public key
  -> agent proves possession of the private key for future logins
  -> Pocket ID issues its normal sessions and OIDC tokens
```

The administrator user form should expose **Agent authentication** as a binary on/off selector. It is not a separate agent identifier: the user's existing Pocket ID username and ID remain authoritative.

- Off: the user follows the existing passkey registration and authentication path.
- On: the user follows the runtime-key registration and authentication path.

The selector changes the authentication process only. It must not create a reduced identity class, change groups or claims, limit existing user or administrator functions, or prevent an agent-authenticated user from being an administrator. The boundary of user management and explicit permissios depends on human choice as it does in the real world.  

## Pitch

Managed software agents are increasingly durable organizational actors. They may need their own identity, group memberships, audit history, and access to OIDC applications, but many run headlessly and cannot complete Pocket ID's human-oriented passkey ceremony.

Using a human's Pocket ID session for an agent weakens identity separation and auditability. Giving the agent an opaque long-lived bearer secret would work technically, but makes the durable credential effectively a password. A registered asymmetric runtime key provides a closer analogue to a passkey: the private key remains with the runtime, Pocket ID stores only the public key, and the credential can be named, audited, and revoked.

A practical first implementation can remain narrow and reuse Pocket ID's existing account, bootstrap, session, OIDC, authorization, and audit concepts.

### Simplified implementation

1. Add an administrator-managed boolean to the user model, exposed in the UI as **Agent authentication** and defaulting to off.
2. Add a runtime credential associated with the normal user. Store a name, credential ID, Ed25519 public key, creation time, last-used time, optional expiration, and revocation time. Never accept or store the private key.
3. Allow the existing administrator-issued one-time login token to begin runtime credential registration for a user whose selector is on.
4. Use short-lived, single-use, operation-bound challenges for registration, login, and fresh reauthentication. The runtime signs the exact challenge with its private key.
5. After successful proof, issue the same normal Pocket ID session used by existing browser and OIDC flows. Runtime login uses the existing username plus credential ID to locate the credential before proof.
6. Expose non-secret runtime credential metadata through the existing account and administrator patterns, including rename and revoke operations.
7. Emit distinct audit events for registration, authentication, sign-in, rename, and revocation.
8. Report proof-of-possession authentication as `amr: ["pop"]`; do not claim passkey, hardware-key, or human-presence assurance.

Conceptually, the API can use paired start/finish operations:

```text
POST /api/runtime-credentials/register/start
POST /api/runtime-credentials/register/finish

POST /api/runtime-credentials/login/start
POST /api/runtime-credentials/login/finish

POST /api/runtime-credentials/reauthenticate/start
POST /api/runtime-credentials/reauthenticate/finish
```

The exact route names are not important to the proposal. The important properties are one-time bootstrap authority, local private-key custody, proof of possession, replay-resistant challenges, normal Pocket ID session issuance, and operational revocability.

### Authentication-path safeguards

- A passkey-path user cannot register or authenticate with a runtime credential.
- A runtime-key user cannot register or authenticate with a passkey.
- Changing the selector is blocked while any passkey or active runtime credential exists on the account.
- Changing paths never deletes or revokes credentials implicitly.
- These rules should be enforced in the service layer and database, not only by disabling UI controls.
- Existing usernames, user IDs, groups, claims, disabled state, and administrator status retain their current meanings.

This also prevents a human from silently adding a passkey to an agent-authenticated administrator account and using it as an alternate authentication route. A human may operate the account through the configured runtime-key process, but does not gain a separate passkey path.

### Revocation behavior

Revoking a runtime credential should prevent future proof-of-possession authentication with that credential. It should not introduce a new universal mechanism for invalidating already-issued Pocket ID cookies, OAuth access tokens, refresh tokens, grants, or relying-party sessions. Those continue to follow Pocket ID's existing logout, authorization, and per-client token-lifetime controls.

### Functional boundary

This feature should not add agent-specific authorization policy. Anything the user may already do through Pocket ID remains permissible when authenticated through the runtime-key path, including administrator functions when `isAdmin` is enabled.

Human discernment remains the boundary: people decide which users to create, whether to enable agent authentication, which roles and groups to assign, which OIDC clients to configure, and whether downstream systems enable those clients.

The proposal does not require Pocket ID to become:

- An agent runtime or workflow engine.
- A delegation or sub-agent authorization system.
- A manager of third-party application secrets.
- A policy layer for actions inside relying applications.
- A credential-linked global session revocation service.

### Acceptance criteria

- An administrator can create a user with Agent authentication on and issue the existing one-time login link.
- A client with no prior Pocket ID session can generate a keypair locally, register the public key through the link, prove possession, and receive a normal Pocket ID session.
- The same client can later authenticate using the existing username, credential ID, and a signed one-time challenge.
- The private key is never transmitted to or stored by Pocket ID.
- Bootstrap and authentication challenges expire and cannot be replayed.
- Passkey and runtime credential paths are mutually exclusive.
- Selector changes are rejected while credentials for either path remain present.
- Users and administrators can see appropriate non-secret credential metadata and revoke the runtime credential.
- Revocation denies a new runtime login while leaving already-issued session and token lifecycles unchanged.
- A runtime-authenticated user receives the same subject and configured OIDC claims as the same Pocket ID user would through the passkey path.
- Existing authorization applies unchanged, including the ability for a runtime-authenticated identity to be a Pocket ID administrator.
- SQLite and PostgreSQL migrations and tests cover the new user selector, credential data, challenges, constraints, and rollback behavior.

### Example use case

An organization runs Pocket ID with a supervisor agent as an administrator and identity gatekeeper for other managed agents. The supervisor can provision new agent users and configure OIDC clients using the same administrator capabilities as any other authorized user. Humans retain control of the endpoint systems and decide whether to enable those clients there. Pocket ID provides identity and authentication; it does not decide what actions an authenticated agent may perform inside each application.
