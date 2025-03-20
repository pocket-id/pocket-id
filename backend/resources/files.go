package resources

import "embed"

// Embedded file systems for the project

//go:embed email-templates images migrations fonts aaguids.json
var FS embed.FS
