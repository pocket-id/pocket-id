package resources

import "embed"

// Embedded file systems for the project

//go:embed email-templates/*.tmpl images default-images migrations fonts aaguids.json aaguid-icons
var FS embed.FS
