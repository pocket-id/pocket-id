package usersignup

import (
	"testing"
)

func strPtr(s string) *string {
	return &s
}

func TestSignupTokenEmailMatchesDomain(t *testing.T) {
	tests := []struct {
		name        string
		emailDomain *string
		email       string
		want        bool
	}{
		{name: "no restriction allows any email", emailDomain: nil, email: "user@anything.com", want: true},
		{name: "empty restriction allows any email", emailDomain: strPtr(""), email: "user@anything.com", want: true},
		{name: "matching domain", emailDomain: strPtr("example.com"), email: "user@example.com", want: true},
		{name: "matching domain case-insensitive", emailDomain: strPtr("example.com"), email: "User@Example.COM", want: true},
		{name: "non-matching domain", emailDomain: strPtr("example.com"), email: "user@other.com", want: false},
		{name: "subdomain does not match", emailDomain: strPtr("example.com"), email: "user@mail.example.com", want: false},
		{name: "domain suffix does not match", emailDomain: strPtr("example.com"), email: "user@notexample.com", want: false},
		{name: "missing @ with restriction", emailDomain: strPtr("example.com"), email: "userexample.com", want: false},
		{name: "empty email with restriction", emailDomain: strPtr("example.com"), email: "", want: false},
		{name: "plus addressing still matches", emailDomain: strPtr("example.com"), email: "user+tag@example.com", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			st := &SignupToken{EmailDomain: tc.emailDomain}
			got := st.EmailMatchesDomain(tc.email)
			if got != tc.want {
				t.Errorf("EmailMatchesDomain(%q) with domain %v = %v, want %v", tc.email, tc.emailDomain, got, tc.want)
			}
		})
	}
}
