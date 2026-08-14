# Agent-Compatible Account Completion Scope

This document defines a narrow development scope for making Pocket ID friendlier to First-Class Agents (FCAs) without expanding Pocket ID into an agent platform.

The intended product boundary is:


> Existing user lifecycle + existing bootstrap intent + agent-compatible completion method


Pocket ID should remain an identity management system. It should not become responsible for agent runtime policy, delegated task authorization, secret management, or application-specific permission enforcement.

## Problem

Pocket ID already supports the important identity primitives for a First-Class Agent:

- A durable user account.
- Administrative user creation.
- One-time login links.
- OIDC identity for downstream applications.
- Groups and claims that applications may consume.
- Existing session, credential, and audit concepts.

The gap is first-login completion.

The normal human onboarding flow assumes that the new user can complete an interactive human ceremony, such as registering a passkey or satisfying a native user-presence prompt.

An FCA such as Vex may be a durable organizational actor with a real account, but may run inside an agent runtime where native passkey registration is unavailable, invisible to automation, or inappropriate for unattended operation.

The goal is to let an administrator deliberately create the FCA user and issue the bootstrap link, then let the FCA complete account setup through a method compatible with agent runtimes.

## Desired Flow

The desired flow should stay as close as possible to human onboarding:


1. Administrator creates user.
2. Administrator provides username/email and one-time login URL.
3. Agent opens one-time login URL.
4. Agent completes account setup through an agent-compatible method.
5. The account behaves like a normal Pocket ID user for OIDC login and application access.


Example:

```text
Administrator creates:
  name: Vex
  email: vex@vanness.me

Administrator sends:
  username/email
  one-time login URL

Vex completes:
  agent-compatible first login

Applications see:
  normal Pocket ID user identity and claims
```

The one-time login URL is the administrator's intentional approval and handoff. The first version should not introduce a second agent-runtime approval gate.

## In Scope

### Agent-Compatible Completion Method

Add a first-login completion path suitable for managed agents.

The completion method should:

- Start from the existing one-time login/bootstrap flow where possible.
- Be usable by a non-human agent runtime.
- Produce normal Pocket ID account state.
- Integrate with existing session, credential, token, and audit paths.
- Avoid requiring native passkey user presence as the only completion option.

The exact mechanism is open for implementation design. Possible shapes include:

- A managed credential created during one-time-link completion.
- A device-style credential.
- A long-lived refresh-capable credential, if it fits Pocket ID's existing security model.
- A first-login setup path that allows a durable non-passkey credential for users marked as managed agents.

The mechanism should be judged by how cleanly it fits existing Pocket ID architecture, not by how agent-specific it appears.

### Authentication Path Selection

Authentication path should be represented by a binary agent-authentication selector on the user that only an authorized administrator can manage. Existing Pocket ID user identifiers remain authoritative; the selector is not another identifier.

- Agent authentication off selects passkey registration and authentication.
- Agent authentication on selects runtime-key registration and authentication.
- The identifier selects process, not account capabilities or the nature of the operator.
- Passkey-path identities cannot register or authenticate with runtime credentials.
- Runtime-path identities cannot register or authenticate with passkeys.
- Changing the agent-authentication selector is blocked while any passkey or active runtime credential exists on the account.
- Moving to the runtime-key path requires explicit removal of all passkeys before the identifier is set.
- Moving to the passkey path requires explicit revocation of all runtime credentials before the identifier is cleared.
- Authentication-path changes never delete or revoke credentials implicitly.

The backend must enforce the zero-credential transition precondition atomically. Administrative UI controls should explain the rule, but hiding or disabling a switch is not a security boundary.

The agent-authentication selector does not create a separate identity system and does not reduce or expand Pocket ID functionality or downstream application permissions. Groups, apps, claims, disabled state, administrator status, and application authorization retain their existing meanings.

The human or software operating an identity is bound to the identity's configured credential path. A human using a runtime-path identity does not gain a passkey option, and software using a passkey-path identity does not gain a runtime-key option.

### Functional Parity And Human Discernment

An identity using the runtime-key path has the same Pocket ID functionality as an identity using the passkey path when existing authorization permits it. An agent identity may be an administrator, provision users, manage groups, or configure OIDC clients when its normal Pocket ID roles permit those operations.

Human choice remains the real boundary. People choose which identities and administrators to provision, which roles and groups to assign, which clients to configure, and whether an endpoint system enables a client. Pocket ID should not add a human-only operation gate merely because an identity authenticates through the runtime-key path.

This scope does not require new user or administrator APIs for functions that are not already exposed. It does require that existing and future functions not be blocked solely because an identity uses the runtime-key path.

### Audit Events

The agent-compatible flow should emit to the existing audit and logging infrastructure.

Useful events include:

- One-time login link issued.
- Agent-compatible account completion started.
- Agent-compatible account completion succeeded.
- Managed credential issued.
- Managed credential revoked.


### Credential Visibility and Revocation

Any new credential created for the completion path should appear wherever comparable credentials or sessions are already visible.

An administrator should be able to revoke it using normal Pocket ID controls.

This is not a request for a new credential inventory system. It is a requirement that new credentials do not become invisible operational debt.

Revoking a runtime credential prevents future authentication with that credential. It does not add credential-linked invalidation for existing Pocket ID cookies, OAuth access tokens, refresh tokens, grants, or relying-party sessions. Those retain their ordinary lifecycle under per-client access-token lifetime, refresh-token inactivity, current authorization checks, logout, and other existing controls.

A short access-token lifetime limits the access token but does not itself invalidate a refresh token. Security-sensitive clients must configure both access and refresh lifetimes according to their acceptable residual window.

## Out of Scope

The following are explicitly out of scope for this Pocket ID change.

### Agent Runtime Policy

Pocket ID scope should not differ from existing functionality related to any other account. 

Examples that are out of scope:

- Whether an agent may spawn a sub-agent.
- Whether a sub-agent may read a document.
- Whether a tool call requires human review.
- Whether an agent may send an email.

Those decisions belong in the agent runtime, gateway, application, or workflow system.

### Delegation Between Agents


Pocket ID should not implement FCA-to-SCA or FCA-to-sub-agent delegation as part of this feature.

This includes:

- Delegated task scopes.
- Ephemeral sub-agent capabilities.
- Actor chains such as `Vex via summary-worker-123`.
- Policy enforcement for delegated workers.

Those are important concepts, but they are separate from enabling a First-Class Agent to complete its own account setup.

### Secret Management

Pocket ID should not become the secret manager for agent operational credentials.

Out of scope:

- Storing third-party application API keys for agents.
- Managing MCP server credentials.
- Writing secrets into OpenBao, Keychain, Vault, or runtime config.
- Rotating downstream app credentials.

Pocket ID may issue and manage its own credentials as part of its identity function. It should not manage unrelated application secrets.

### Application-Specific Permissions

Pocket ID should not attempt to enforce permissions inside downstream applications.

Examples:

- Outline collection permissions.
- Gitea repository permissions.
- Email send permissions.
- Purchasing limits.

Pocket ID may emit identity claims and groups. The receiving application remains responsible for interpreting those claims and enforcing its own permissions.

### Fine-Grained OAuth Delegation

Fine-grained delegated OAuth scopes are out of scope unless they already fit directly into Pocket ID's existing OAuth/API model.

This feature should not require Pocket ID to implement a broad delegation framework, token exchange system, or resource-specific policy engine.

## Non-Goals

This work should not:

- Create a separate agent identity product.
- Replace passkeys on the default credential path.
- Weaken the default passkey-path security model.
- Bypass administrator intent.
- Add broad application authorization features.
- Add an MCP platform.
- Add a workflow engine.
- Add a secret broker.
- Turn every spawned worker into a user account.

## Design Constraints

### Preserve Existing Pipelines

The implementation should mesh with existing Pocket ID concepts wherever practical:

- User creation.
- One-time links.
- Sessions.
- Credentials.
- OIDC claims.
- Groups.
- Audit logs.
- Admin controls.

New behavior should feel like an additional account-completion method, not a separate system bolted onto Pocket ID.

### Administrator Intent Is the Gate

Creating the user and issuing the one-time login URL is the administrator's approval.

The first implementation should not require a second administrator approval to bind the account to the agent runtime unless existing Pocket ID architecture requires it for security.

### Normal User And Administrator Semantics After Completion

After completion, the FCA should be a normal Pocket ID user from the perspective of applications.

For example, when Vex logs into Outline via OIDC, Outline should see a normal user identity:

```text
name: Vex
email: vex@vanness.me
groups: emitted by Pocket ID according to normal configuration
```

The downstream application should not need to know whether Vex completed account setup with a passkey or an agent-compatible method.

Pocket ID itself must likewise preserve normal authorization semantics. If Vex is assigned administrator status, Vex may use existing administrator functions; the runtime-key path is not a reason to deny them.

### No Assumption of Downstream Scope Reduction

An agent-compatible Pocket ID login does not imply reduced per-task access in downstream applications.

If Vex has access to a wiki collection, an application token or session for Vex usually carries Vex's application permissions. Reducing that scope is the responsibility of the application, an agent runtime, or a gateway.

## Suggested First Milestone

The first milestone should be deliberately small:

```text
An administrator can create an FCA user and issue a one-time login link.
The FCA can use that link to complete account setup without native passkey registration.
The resulting account can authenticate to an OIDC application as a normal user.
The issued credential/session is visible and revocable through Pocket ID's existing administrative model.
```

Example acceptance test:

```text
Given an administrator creates Vex <vex@vanness.me>
And the administrator issues a one-time login URL
When Vex opens the URL from an agent runtime
And completes the agent-compatible setup flow
Then Vex can authenticate to an OIDC application
And the application receives normal Vex identity claims
And an administrator can see and revoke the credential created by the setup flow
And existing tokens and sessions retain their configured ordinary lifecycle
```

## Relationship to Later Work

This scope enables First-Class Agents. It does not solve all agent authorization.

Later work and another project may explore:

- Delegated capabilities from FCAs to SCAs.
- Ephemeral sub-agent task capabilities.
- OAuth token exchange or actor-token semantics.
- Agent runtime policy enforcement.
- Application gateways that enforce delegated scope before calling downstream APIs.

Those are real problems, but they should not be prerequisites for the first Pocket ID feature. The first feature should simply let a deliberately provisioned First-Class Agent complete account onboarding cleanly.

## Implementation Guide

The concrete v1 key, challenge, API, authentication-path, lifecycle, and rotation contract is documented in [Agent Runtime Credentials](agent-runtime-credentials.md).
