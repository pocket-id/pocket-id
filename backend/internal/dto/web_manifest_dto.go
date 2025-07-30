package dto

type WebAppManifestDto struct {
	Name    string                  `json:"name"`
	Icons   []WebAppManifestIconDto `json:"icons"`
	Display string                  `json:"display"`
}

type WebAppManifestIconDto struct {
	Src     string `json:"src"`
	Sizes   string `json:"sizes"`
	Type    string `json:"type"`
	Purpose string `json:"purpose"`
}
