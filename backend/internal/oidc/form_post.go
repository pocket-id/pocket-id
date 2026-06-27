package oidc

import (
	"crypto/sha256"
	"encoding/base64"
	"html/template"
)

// formPostAutoSubmitScript submits the response_mode=form_post page back to the client as soon as it loads
// Pocket ID's Content-Security-Policy forbids 'unsafe-inline' scripts and inline event handlers, which is exactly what fosite's default form_post template relies on (<body onload="...">), so that page silently never submits and strands the user on a blank page
// We instead deliver the auto-submit as a regular inline <script> element and allow-list exactly this body via its SHA-256 hash (formPostScriptCSPHash); the script body and the hash must stay byte-for-byte identical, which form_post_test.go enforces
const formPostAutoSubmitScript = `document.forms[0].submit()`

// formPostScriptCSPHash is the CSP script-src source that allow-lists formPostAutoSubmitScript, e.g. "'sha256-...'"
var formPostScriptCSPHash = cspHashOf(formPostAutoSubmitScript)

// formPostTemplate replaces fosite's DefaultFormPostTemplate
// It behaves identically except the auto-submit runs from an allow-listed <script> element instead of a CSP-blocked inline onload handler, and a <noscript> button lets the user continue if scripts are disabled
var formPostTemplate = template.Must(template.New("form_post").Parse(
	`<!DOCTYPE html>
<html>
<head><title>Submit This Form</title></head>
<body>
<form method="post" action="{{ .RedirURL }}">
{{- range $key, $values := .Parameters }}
{{- range $value := $values }}
<input type="hidden" name="{{ $key }}" value="{{ $value }}"/>
{{- end }}
{{- end }}
<noscript><button type="submit">Continue</button></noscript>
</form>
<script>` + formPostAutoSubmitScript + `</script>
</body>
</html>`))

// cspHashOf returns the CSP hash-source expression ("'sha256-...'") for an inline script body
func cspHashOf(script string) string {
	sum := sha256.Sum256([]byte(script))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}
