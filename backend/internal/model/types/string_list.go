package datatype

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/pocket-id/pocket-id/backend/internal/utils"
)

type StringList []string //nolint:recvcheck

func (s *StringList) Scan(value any) error {
	return utils.UnmarshalJSONFromDatabase(s, value)
}

func (s StringList) Value() (driver.Value, error) {
	return json.Marshal(s)
}
