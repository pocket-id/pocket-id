package resources

import "embed"

// Embedded file systems for the project
// Note that aaguids.json is not embedded, but it's used to source-generate a Go file

//go:embed email-templates images migrations fonts
var FS embed.FS
