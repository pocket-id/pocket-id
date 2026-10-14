This SMTP adapter is derived from `github.com/italypaleale/go-kit/emailer/smtp`
v1.2.5 (commit `12828ecaccd71e38dfe4d37833c14e00c00e6209`). Its upstream MIT license
is retained in `LICENSE.txt`.

The adapter adds an optional `InitOpts.Transform` callback. It receives the final
RFC 5322 message bytes before an SMTP connection is opened. A callback error
aborts delivery. Without a callback, MIME generation and transport retain the
upstream behavior. Public email address/message types remain those of go-kit.

This keeps Pocket ID buildable without depending on an unpublished library fork.
The callback can be proposed upstream later; once a released go-kit version
provides it, this adapter can be removed. Preserve upstream SMTP fixes while it
is retained. Upstream transport tests are included, alongside Pocket ID tests
that verify S/MIME on actual SMTP wire messages.
