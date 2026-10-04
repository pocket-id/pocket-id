package logopreset

type logoPresetDto struct {
	Name        string  `json:"name"`
	Reference   string  `json:"reference"`
	LogoURL     string  `json:"logoUrl"`
	DarkLogoURL *string `json:"darkLogoUrl"`
}
