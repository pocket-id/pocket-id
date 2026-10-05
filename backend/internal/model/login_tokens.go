package model

// LoginTokens carries the separate authentication and browser recognition cookies
type LoginTokens struct {
	AccessToken       string
	KnownBrowserToken string
}
