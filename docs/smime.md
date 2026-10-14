# S/MIME email signing

In **Settings → Administration → Application configuration → E-Mail**, save
SMTP settings first, then upload a PKCS#12 (`.p12` or `.pfx`) file containing the
sender certificate, its private key, and optionally its issuer chain. Enter the
file's password, then enable **Sign outgoing emails**. The toggle is saved
immediately. Use the existing **Send test email** button to check delivery and
signature display in your mail client.

The certificate must be currently valid, permit email protection and digital
signatures, and include the configured SMTP sender address in its email subject
alternative names. RSA keys of 2048–8192 bits and ECDSA keys of at least 256 bits
are supported. Uploads are limited to a 1 MiB multipart request. Include any
intermediate issuer certificates needed by recipient clients. Certificate
bag order does not matter; the sender certificate is selected by its public key.
Trust is evaluated by the recipient; the displayed validity status refers to
certificate dates and is not a public-CA trust or revocation check.

The page displays certificate holder, issuer, email addresses, validity dates,
and SHA-256 fingerprint. Certificates expiring within 30 days are highlighted.
Replace an expiring certificate by uploading its replacement; signing remains
enabled. Removing a certificate also disables signing and clears its key.

When enabled, all outgoing system emails (test emails, email verification,
one-time access, login notifications, and API-key expiry notifications) use
SHA-256 detached CMS signatures in `multipart/signed`. Text and HTML MIME parts
remain readable by clients without S/MIME support. This feature signs messages;
it does not encrypt their contents or protect ordinary outer transport headers.

If the certificate expires, becomes unusable, or no longer matches the saved
sender address, sending fails before opening the SMTP connection. Messages are
not silently sent without a signature. Replace the certificate, restore the
sender address, or explicitly disable signing to restore delivery. A failed
send is not automatically queued by this feature.

Private keys are persisted using Pocket ID's `EncryptedString` encryption and
its existing `ENCRYPTION_KEY`. The upload password is only used in memory to
open the file and is not persisted. API responses expose certificate metadata,
never the private key. Read and write endpoints require `config:read` and
`config:write`, respectively. Writes are disabled when `UI_CONFIG_DISABLED` is
set. This version supports certificate management through the UI/API; it does
not add environment-based S/MIME certificate provisioning.

Database export/import preserves encrypted key material. Retain the matching
master encryption key with backups. `encryption-key-rotate` also re-encrypts the
S/MIME key. Schema migrations are provided for SQLite and PostgreSQL.

The local SMTP adapter is documented in
`backend/internal/email/smtptransport/README.md`. Go libraries perform PKCS#12
decoding and CMS signing; no external OpenSSL process is required in production.
Both DER and BER PKCS#12 containers are supported using the local SSLMate decoder
adaptation documented in `backend/third_party/go-pkcs12/README.pocket-id.md`.
The decoder verifies the MAC over the original authenticated bytes before parsing
the contained ASN.1 data, and bounds input size, nesting, and element count.
KDF iteration counts are limited to 1–1,000,000 per derivation to bound import
CPU work; higher-count files must be exported again with supported parameters.
Tests verify delivered MIME bytes, reject tampered content, cover both RSA and
ECDSA, and independently check signatures with OpenSSL when it is available.
