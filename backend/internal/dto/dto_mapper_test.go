package dto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pocket-id/pocket-id/backend/internal/model"
	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

type sourceStruct struct {
	AString            string
	AStringPtr         *string
	ABool              bool
	ABoolPtr           *bool
	ACustomDateTime    datatype.DateTime
	ACustomDateTimePtr *datatype.DateTime
	ANilStringPtr      *string
	ASlice             []string
	AMap               map[string]int
	AStruct            embeddedStruct
	AStructPtr         *embeddedStruct

	StringPtrToString      *string
	EmptyStringPtrToString *string
	NilStringPtrToString   *string
	IntToInt64             int
	AuditLogEventToString  model.AuditLogEvent
}

type destStruct struct {
	AString            string
	AStringPtr         *string
	ABool              bool
	ABoolPtr           *bool
	ACustomDateTime    datatype.DateTime
	ACustomDateTimePtr *datatype.DateTime
	ANilStringPtr      *string
	ASlice             []string
	AMap               map[string]int
	AStruct            embeddedStruct
	AStructPtr         *embeddedStruct

	StringPtrToString      string
	EmptyStringPtrToString string
	NilStringPtrToString   string
	IntToInt64             int64
	AuditLogEventToString  string
}

type embeddedStruct struct {
	Foo string
	Bar int64
}

func TestMapStruct(t *testing.T) {
	src := sourceStruct{
		AString:            "abcd",
		AStringPtr:         new("xyz"),
		ABool:              true,
		ABoolPtr:           new(false),
		ACustomDateTime:    datatype.DateTime(time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)),
		ACustomDateTimePtr: new(datatype.DateTime(time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC))),
		ANilStringPtr:      nil,
		ASlice:             []string{"a", "b", "c"},
		AMap: map[string]int{
			"a": 1,
			"b": 2,
		},
		AStruct: embeddedStruct{
			Foo: "bar",
			Bar: 42,
		},
		AStructPtr: &embeddedStruct{
			Foo: "quo",
			Bar: 111,
		},

		StringPtrToString:      new("foobar"),
		EmptyStringPtrToString: new(""),
		NilStringPtrToString:   nil,
		IntToInt64:             99,
		AuditLogEventToString:  model.AuditLogEventAccountCreated,
	}
	var dst destStruct
	err := MapStruct(src, &dst)
	require.NoError(t, err)

	assert.Equal(t, src.AString, dst.AString)
	_ = assert.NotNil(t, src.AStringPtr) &&
		assert.Equal(t, *src.AStringPtr, *dst.AStringPtr)
	assert.Equal(t, src.ABool, dst.ABool)
	_ = assert.NotNil(t, src.ABoolPtr) &&
		assert.Equal(t, *src.ABoolPtr, *dst.ABoolPtr)
	assert.Equal(t, src.ACustomDateTime, dst.ACustomDateTime)
	_ = assert.NotNil(t, src.ACustomDateTimePtr) &&
		assert.Equal(t, *src.ACustomDateTimePtr, *dst.ACustomDateTimePtr)
	assert.Nil(t, dst.ANilStringPtr)
	assert.Equal(t, src.ASlice, dst.ASlice)
	assert.Equal(t, src.AMap, dst.AMap)
	assert.Equal(t, "bar", dst.AStruct.Foo)
	assert.Equal(t, int64(42), dst.AStruct.Bar)
	_ = assert.NotNil(t, src.AStructPtr) &&
		assert.Equal(t, "quo", dst.AStructPtr.Foo) &&
		assert.Equal(t, int64(111), dst.AStructPtr.Bar)
	assert.Equal(t, "foobar", dst.StringPtrToString)
	assert.Empty(t, dst.EmptyStringPtrToString)
	assert.Empty(t, dst.NilStringPtrToString)
	assert.Equal(t, int64(99), dst.IntToInt64)
	assert.Equal(t, "ACCOUNT_CREATED", dst.AuditLogEventToString)
}

func TestMapStructList(t *testing.T) {
	sources := []sourceStruct{
		{
			AString:            "first",
			AStringPtr:         new("one"),
			ABool:              true,
			ABoolPtr:           new(false),
			ACustomDateTime:    datatype.DateTime(time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)),
			ACustomDateTimePtr: new(datatype.DateTime(time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC))),
			ASlice:             []string{"a", "b"},
			AMap: map[string]int{
				"a": 1,
				"b": 2,
			},
			AStruct: embeddedStruct{
				Foo: "first_struct",
				Bar: 10,
			},
			IntToInt64: 10,
		},
		{
			AString:            "second",
			AStringPtr:         new("two"),
			ABool:              false,
			ABoolPtr:           new(true),
			ACustomDateTime:    datatype.DateTime(time.Date(2026, 6, 7, 8, 9, 10, 0, time.UTC)),
			ACustomDateTimePtr: new(datatype.DateTime(time.Date(2023, 6, 7, 8, 9, 10, 0, time.UTC))),
			ASlice:             []string{"c", "d", "e"},
			AMap: map[string]int{
				"c": 3,
				"d": 4,
			},
			AStruct: embeddedStruct{
				Foo: "second_struct",
				Bar: 20,
			},
			IntToInt64: 20,
		},
	}

	var destinations []destStruct
	err := MapStructList(sources, &destinations)

	require.NoError(t, err)
	require.Len(t, destinations, 2)

	// Verify first element
	assert.Equal(t, "first", destinations[0].AString)
	assert.Equal(t, "one", *destinations[0].AStringPtr)
	assert.True(t, destinations[0].ABool)
	assert.False(t, *destinations[0].ABoolPtr)
	assert.Equal(t, datatype.DateTime(time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)), destinations[0].ACustomDateTime)
	assert.Equal(t, datatype.DateTime(time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)), *destinations[0].ACustomDateTimePtr)
	assert.Equal(t, []string{"a", "b"}, destinations[0].ASlice)
	assert.Equal(t, map[string]int{"a": 1, "b": 2}, destinations[0].AMap)
	assert.Equal(t, "first_struct", destinations[0].AStruct.Foo)
	assert.Equal(t, int64(10), destinations[0].AStruct.Bar)
	assert.Equal(t, int64(10), destinations[0].IntToInt64)

	// Verify second element
	assert.Equal(t, "second", destinations[1].AString)
	assert.Equal(t, "two", *destinations[1].AStringPtr)
	assert.False(t, destinations[1].ABool)
	assert.True(t, *destinations[1].ABoolPtr)
	assert.Equal(t, datatype.DateTime(time.Date(2026, 6, 7, 8, 9, 10, 0, time.UTC)), destinations[1].ACustomDateTime)
	assert.Equal(t, datatype.DateTime(time.Date(2023, 6, 7, 8, 9, 10, 0, time.UTC)), *destinations[1].ACustomDateTimePtr)
	assert.Equal(t, []string{"c", "d", "e"}, destinations[1].ASlice)
	assert.Equal(t, map[string]int{"c": 3, "d": 4}, destinations[1].AMap)
	assert.Equal(t, "second_struct", destinations[1].AStruct.Foo)
	assert.Equal(t, int64(20), destinations[1].AStruct.Bar)
	assert.Equal(t, int64(20), destinations[1].IntToInt64)
}

func TestMapStructList_EmptySource(t *testing.T) {
	var sources []sourceStruct
	var destinations []destStruct

	err := MapStructList(sources, &destinations)
	require.NoError(t, err)
	assert.Empty(t, destinations)
}
