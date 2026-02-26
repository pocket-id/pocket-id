package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// JSONDuration is a type that allows marshalling/unmarshalling a Duration
type JSONDuration struct {
	time.Duration
}

func (d JSONDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *JSONDuration) UnmarshalJSON(b []byte) error {
	var v any
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		// If the value is a number, interpret it as a number of seconds
		d.Duration = time.Duration(value) * time.Second
		return nil
	case string:
		if v == "" {
			return nil
		}
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("invalid duration")
	}
}

func UnmarshalJSONFromDatabase(data any, value any) error {
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, data)
	case string:
		return json.Unmarshal([]byte(v), data)
	default:
		return fmt.Errorf("unsupported type: %T", value)
	}
}
