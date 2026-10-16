# Pocket ID PKCS#12 decoder

This directory contains [SSLMate/go-pkcs12 v0.7.4](https://github.com/SSLMate/go-pkcs12/tree/v0.7.4), including its upstream
BSD license, tests, and fixtures. The backend replaces the upstream module
with this local copy to accept BER PKCS#12 files without an OpenSSL process.

Local changes include `ber.go` and a normalization call in `unmarshal`
in `pkcs12.go`. Normalization handles definite and indefinite containers,
constructed OCTET STRINGs, and implicit encrypted-content OCTET STRINGs.
It rejects malformed data and limits input size, nesting, and element count.
Primitive OCTET STRING payloads are never recursively rewritten. In particular,
the original authenticated-safe bytes are verified by the upstream MAC code
before their ASN.1 framing is normalized for parsing. No MAC is removed,
recalculated, or bypassed.

`limits.go` bounds KDF iterations to 1–1,000,000 before password derivation in
`crypto.go` and `mac.go`, including legacy PBE, PBES2, classic MAC, and PBMAC1.
This rejects malformed counts and prevents a small upload from requesting
unbounded CPU work. The underlying cryptographic algorithms remain upstream code.

Run `go test software.sslmate.com/src/go-pkcs12/...` from `backend` for the
upstream and BER parser tests using the application's dependency versions.
The application's `internal/email` tests cover BER imports, password/MAC
failures, certificate chains, and signature verification. When updating the
upstream version, preserve the normalization hook and run both suites.
