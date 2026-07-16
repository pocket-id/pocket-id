package usersignup

import (
	"strings"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// SignupToken is a single- or limited-use token that grants the ability to self-register
type SignupToken struct {
	model.Base

	Token       string            `json:"token"`
	ExpiresAt   datatype.DateTime `json:"expiresAt" sortable:"true"`
	UsageLimit  int               `json:"usageLimit" sortable:"true"`
	UsageCount  int               `json:"usageCount" sortable:"true"`
	EmailDomain *string           `json:"emailDomain"`
	UserGroups  []model.UserGroup `gorm:"many2many:signup_tokens_user_groups;"`
}

func (st *SignupToken) IsExpired() bool {
	return time.Time(st.ExpiresAt).Before(time.Now())
}

func (st *SignupToken) IsUsageLimitReached() bool {
	return st.UsageCount >= st.UsageLimit
}

func (st *SignupToken) IsValid() bool {
	return !st.IsExpired() && !st.IsUsageLimitReached()
}

// HasEmailDomainRestriction reports whether the token limits sign-ups to a specific email domain
func (st *SignupToken) HasEmailDomainRestriction() bool {
	return st.EmailDomain != nil && *st.EmailDomain != ""
}

// EmailMatchesDomain reports whether the given email address is allowed by the token's domain restriction
// It returns true when the token has no restriction
// The comparison is case-insensitive
func (st *SignupToken) EmailMatchesDomain(email string) bool {
	if !st.HasEmailDomainRestriction() {
		return true
	}

	at := strings.LastIndexByte(email, '@')
	if at < 0 {
		return false
	}

	return strings.EqualFold(email[at+1:], *st.EmailDomain)
}
