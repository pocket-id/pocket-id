package model

import "time"

const KnownBrowserLifetime = 180 * 24 * time.Hour

// KnownBrowser recognizes a browser for notifications without granting authentication
// Only the hash of the cookie token is stored
type KnownBrowser struct {
	UserID    string `gorm:"primaryKey"`
	TokenHash string `gorm:"primaryKey"`
	ExpiresAt int64
}

// LoginTokens carries the separate authentication and browser recognition cookies
type LoginTokens struct {
	AccessToken       string
	KnownBrowserToken string
}

// SignInResult defers notification delivery until the login has committed
type SignInResult struct {
	AuditLog          AuditLog
	Created           bool
	Notify            bool
	KnownBrowserToken string
}
