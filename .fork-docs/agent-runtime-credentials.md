# Agent Runtime Credentials

Pocket ID supports an administrator-selected runtime-key authentication path for a user whose `isAgent` selector is on. The selector changes only the authentication process: the user's existing username remains the sign-in identifier, and the user retains the same groups, claims, administrator status, OIDC clients, and authorization behavior as any other Pocket ID user.

The runtime generates an Ed25519 keypair locally and never sends the private key to Pocket ID. Pocket ID stores only the public key, verifies signed one-time challenges, and issues its normal session cookie with `amr: ["pop"]`. `pop` is the IANA-registered generic proof-of-possession authentication-method reference; it does not assert human presence or whether key custody is implemented in software or hardware.

## Administrator Preparation

1. Create or edit the user through the existing administrator user API or UI.
2. Turn on **Agent authentication** for the user. No additional identifier is created; the existing unique username is used for repeat login.
3. Create a one-time access link for that user through the existing administrator control.
4. Deliver the link to the intended runtime over an appropriate secure channel.

The selector is off by default. Pocket ID blocks changing it while the account has any passkey or any active runtime credential. Remove passkeys or revoke the active runtime credential first; path changes never delete credentials implicitly.

## Encoding And Signature Contract

- Algorithm: Ed25519
- Public key: the 32 raw public-key bytes encoded as unpadded base64url
- Signature: the 64 raw Ed25519 signature bytes encoded as unpadded base64url
- Challenge: the exact unpadded-base64url-decoded bytes returned by Pocket ID
- Challenge lifetime: 60 seconds
- Challenge use: one successful registration, login, or reauthentication

The challenge bytes begin with an operation-specific prefix and a newline, followed by 32 random bytes:

```text
pocket-id-runtime-credential/v1/register\n<32 random bytes>
pocket-id-runtime-credential/v1/login\n<32 random bytes>
pocket-id-runtime-credential/v1/reauthenticate\n<32 random bytes>
```

Clients must sign the decoded response bytes exactly. They must not reconstruct the prefix, decode the random suffix separately, hash the challenge before signing, or use padded base64.

## Complete Bootstrap

Extract the token from the administrator-issued `/lc/<token>` URL. Do not exchange that link through the ordinary one-time-login endpoint; registration consumes the same one-time authority directly.

Generate the Ed25519 keypair locally, retain the private key in the runtime's protected credential store, and start registration:

```http
POST /api/runtime-credentials/register/start
Content-Type: application/json

{
  "token": "<one-time-token>",
  "name": "Vex OpenClaw Mac mini",
  "algorithm": "Ed25519",
  "publicKey": "<unpadded-base64url-public-key>"
}
```

The response contains only a short-lived proof session:

```json
{
  "sessionId": "<registration-session-id>",
  "challenge": "<unpadded-base64url-challenge>",
  "expiresAt": "<timestamp>"
}
```

Sign the decoded `challenge` with the retained private key and finish registration:

```http
POST /api/runtime-credentials/register/finish
Content-Type: application/json

{
  "sessionId": "<registration-session-id>",
  "signature": "<unpadded-base64url-ed25519-signature>"
}
```

Pocket ID returns the normal user DTO plus non-secret credential metadata and sets the normal Pocket ID access-token cookie. The private key is neither accepted nor returned. A completed registration consumes the one-time link; a second active runtime credential requires revoking the current credential and completing a new administrator-issued link.

## Repeat Authentication

Start login with the user's existing Pocket ID username and stored credential ID:

```http
POST /api/runtime-credentials/login/start
Content-Type: application/json

{
  "username": "vex",
  "credentialId": "<credential-id>"
}
```

Sign the returned decoded challenge bytes, then finish:

```http
POST /api/runtime-credentials/login/finish
Content-Type: application/json

{
  "sessionId": "<login-session-id>",
  "signature": "<unpadded-base64url-ed25519-signature>"
}
```

The response is the normal user DTO and sets the same access-token cookie used by Pocket ID's browser and OIDC flows. A headless client must retain and present that cookie when it continues to `/authorize` or calls cookie-authenticated Pocket ID APIs.

Invalid usernames, credential IDs, expired or revoked credentials, disabled users, expired sessions, and invalid signatures return the same generic runtime-credential failure. Clients should start a new challenge rather than retry an expired session.

## Fresh Reauthentication

When an existing Pocket ID interaction requires fresh proof, an authenticated runtime can use:

```http
POST /api/runtime-credentials/reauthenticate/start
Content-Type: application/json
Cookie: <normal-pocket-id-session-cookie>

{
  "credentialId": "<credential-id>"
}
```

After signing the returned challenge, finish with the same `{ "sessionId", "signature" }` shape at `POST /api/runtime-credentials/reauthenticate/finish`. Pocket ID sets its ordinary short-lived reauthentication cookie. Runtime proof satisfies the same freshness requirement as a fresh passkey ceremony but remains method-specific proof and never claims human presence.

## Credential Management And Rotation

An authenticated user can list, rename, and revoke its own runtime credential:

```text
GET    /api/runtime-credentials
PATCH  /api/runtime-credentials/<credential-id>  { "name": "<new-name>" }
DELETE /api/runtime-credentials/<credential-id>
```

An authenticated administrator can list and revoke a user's credential:

```text
GET    /api/users/<user-id>/runtime-credentials
DELETE /api/users/<user-id>/runtime-credentials/<credential-id>
```

Listing returns the credential ID, name, algorithm, creation time, last-used time, optional expiration, and revocation time. It never returns public-key bytes, challenges, or signatures.

Revocation immediately prevents future registration-key proofs. It does not introduce credential-linked invalidation for existing Pocket ID sessions, access tokens, refresh tokens, grants, or application-local sessions; those continue under each client's configured lifetimes and Pocket ID's existing controls. Rotation is therefore explicit: revoke, issue a new one-time link, generate a new local keypair, and register the replacement.

## Authentication-Path Boundaries

- A user with `isAgent` off uses passkeys and cannot register or authenticate with a runtime credential.
- A user with `isAgent` on uses runtime credentials and cannot register or authenticate with a passkey.
- The database and service layer both enforce credential-family exclusivity.
- Credential path never changes authorization. A runtime-key user may be a Pocket ID administrator when its normal `isAdmin` setting permits it.
- Pocket ID does not decide which downstream actions are appropriate for an agent. Humans retain discernment through identity provisioning, group and administrator assignment, OIDC client configuration, and the relying application's decision to enable that client.

## Standards References

- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html) defines `amr` as an optional array of authentication-method identifiers.
- [IANA Authentication Method Reference Values](https://www.iana.org/assignments/authentication-method-reference-values/authentication-method-reference-values.xhtml) registers `pop` as proof of possession of a key, distinct from `swk`, `hwk`, `phr`, and human-presence signals.
- [RFC 8176](https://www.rfc-editor.org/rfc/rfc8176.html) explains the authentication-method reference registry and the separation between authentication method evidence and authorization policy.
