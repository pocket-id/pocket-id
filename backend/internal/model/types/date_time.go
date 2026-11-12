package datatype

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"github.com/pocket-id/pocket-id/backend/internal/common"
)

// DateTime custom type for time.Time to store date as unix timestamp for sqlite and as date for postgres
type DateTime time.Time //nolint:recvcheck

func DateTimeFromString(str string) (DateTime, error) {
	t, err := time.Parse(time.RFC3339Nano, str)
	if err != nil {
		return DateTime{}, fmt.Errorf("failed to parse date string: %w", err)
	}

	return DateTime(t), nil
}

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
	var t time.Time
	err := t.UnmarshalJSON(b)
	if err != nil {
		return err
	}

	// MarshalJSON fails for years past 9999 in the time's zone, so a stored time must stay below that in any zone it's read back in
	if t.UTC().Year() >= 9999 {
		return errors.New("year must be before 9999")
	}

	*date = DateTime(t)
	return nil
}
