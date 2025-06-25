package jwk

import (
	"github.com/lestrrat-go/jwx/v3/jwk"
	"gorm.io/gorm"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

type KeyProviderOpts struct {
	EnvConfig *common.EnvConfigSchema
	DB        *gorm.DB
	Kek       []byte
}

type KeyProvider interface {
	Init(opts KeyProviderOpts) error
	LoadKey() (jwk.Key, error)
	SaveKey(key jwk.Key) error
}
