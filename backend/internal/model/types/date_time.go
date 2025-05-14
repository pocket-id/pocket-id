package datatype

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// DateTime custom type for time.Time to store date as unix timestamp for sqlite and as date for postgres
type DateTime time.Time //nolint:recvcheck

func (date *DateTime) Scan(value any) (err error) {
	switch v := value.(type) {
	case time.Time:
		*date = DateTime(v)
	case int64:
		*date = DateTime(time.Unix(v, 0))
	default:
		return fmt.Errorf("unexpected type for DateTime: %T", value)
	}
	return nil
}

func (date DateTime) Value() (driver.Value, error) {
	if common.EnvConfig.DbProvider == common.DbProviderSqlite {
		return time.Time(date).Unix(), nil
	} else {
		return time.Time(date), nil
	}
}

func (date DateTime) UTC() time.Time {
	return time.Time(date).UTC()
}

func (date DateTime) ToTime() time.Time {
	return time.Time(date)
}

// GormDataType gorm common data type
func (date DateTime) GormDataType() string {
	return "date"
}

func (date DateTime) GobEncode() ([]byte, error) {
	return time.Time(date).GobEncode()
}

func (date *DateTime) GobDecode(b []byte) error {
	return (*time.Time)(date).GobDecode(b)
}

func (date DateTime) MarshalJSON() ([]byte, error) {
	return time.Time(date).MarshalJSON()
}

func (date *DateTime) UnmarshalJSON(b []byte) error {
	return (*time.Time)(date).UnmarshalJSON(b)
}
