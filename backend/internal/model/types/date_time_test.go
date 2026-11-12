package datatype

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDateTimeRejectsTimesThatCannotBeMarshalledBack(t *testing.T) {
	var date DateTime
	err := json.Unmarshal([]byte(`"9999-12-31T23:59:59-01:00"`), &date)
	require.Error(t, err)
}
