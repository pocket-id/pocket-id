package utils

import (
	"encoding/json"
	"errors"
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
