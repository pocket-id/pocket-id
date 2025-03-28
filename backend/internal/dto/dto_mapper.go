package dto

import (
	"errors"
	"reflect"
	"time"

	datatype "github.com/pocket-id/pocket-id/backend/internal/model/types"
)

// MapStructList maps a list of source structs to a list of destination structs
func MapStructList[S any, D any](source []S, destination *[]D) error {
	*destination = make([]D, 0, len(source))

	for _, item := range source {
		var destItem D
		if err := MapStruct(item, &destItem); err != nil {
			return err
		}
		*destination = append(*destination, destItem)
	}
	return nil
}

// MapStruct maps a source struct to a destination struct
func MapStruct[S any, D any](source S, destination *D) error {
	// Ensure destination is a non-nil pointer
	destValue := reflect.ValueOf(destination)
	if destValue.Kind() != reflect.Ptr || destValue.IsNil() {
		return errors.New("destination must be a non-nil pointer to a struct")
	}

	// Ensure source is a struct
	sourceValue := reflect.ValueOf(source)
	if sourceValue.Kind() != reflect.Struct {
		return errors.New("source must be a struct")
	}

	return mapStructInternal(sourceValue, destValue.Elem())
}

func mapStructInternal(sourceVal reflect.Value, destVal reflect.Value) error {
	for i := 0; i < destVal.NumField(); i++ {
		destField := destVal.Field(i)
		destFieldType := destVal.Type().Field(i)

		if destFieldType.Anonymous {
			if err := mapStructInternal(sourceVal, destField); err != nil {
				return err
			}
			continue
		}

		sourceField := sourceVal.FieldByName(destFieldType.Name)

		if sourceField.IsValid() && destField.CanSet() {
			if err := mapField(sourceField, destField); err != nil {
				return err
			}
		}
	}
	return nil
}

func mapField(sourceField reflect.Value, destField reflect.Value) error {
	switch {
	case sourceField.Type() == destField.Type():
		destField.Set(sourceField)
	case sourceField.Kind() == reflect.Slice && destField.Kind() == reflect.Slice:
		return mapSlice(sourceField, destField)
	case sourceField.Kind() == reflect.Struct && destField.Kind() == reflect.Struct:
		return mapStructInternal(sourceField, destField)
	default:
		return mapSpecialTypes(sourceField, destField)
	}
	return nil
}

func mapSlice(sourceField reflect.Value, destField reflect.Value) error {
	if sourceField.Type().Elem() == destField.Type().Elem() {
		newSlice := reflect.MakeSlice(destField.Type(), sourceField.Len(), sourceField.Cap())
		for j := 0; j < sourceField.Len(); j++ {
			newSlice.Index(j).Set(sourceField.Index(j))
		}
		destField.Set(newSlice)
	} else if sourceField.Type().Elem().Kind() == reflect.Struct && destField.Type().Elem().Kind() == reflect.Struct {
		newSlice := reflect.MakeSlice(destField.Type(), sourceField.Len(), sourceField.Cap())
		for j := 0; j < sourceField.Len(); j++ {
			sourceElem := sourceField.Index(j)
			destElem := reflect.New(destField.Type().Elem()).Elem()
			if err := mapStructInternal(sourceElem, destElem); err != nil {
				return err
			}
			newSlice.Index(j).Set(destElem)
		}
		destField.Set(newSlice)
	}
	return nil
}

func mapSpecialTypes(sourceField reflect.Value, destField reflect.Value) error {
	if _, ok := sourceField.Interface().(datatype.DateTime); ok {
		if sourceField.Type() == reflect.TypeOf(datatype.DateTime{}) && destField.Type() == reflect.TypeOf(time.Time{}) {
			dateValue := sourceField.Interface().(datatype.DateTime)
			destField.Set(reflect.ValueOf(dateValue.ToTime()))
		}
	}
	return nil
}
