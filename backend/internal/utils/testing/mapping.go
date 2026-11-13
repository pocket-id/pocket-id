//go:build unit

// This file is only imported by unit tests

package testing

import (
	"reflect"
	"strings"
)

// UnmappedFields lists the fields of destination that dto.MapStruct can't fill from source, including those of nested structs
// It follows copier's rules: a field is filled from a source field of the same name in any case, or from a source method of the same name
func UnmappedFields(source, destination any) []string {
	return unmappedFields(reflect.TypeOf(source), reflect.TypeOf(destination), "")
}

func unmappedFields(source, destination reflect.Type, prefix string) []string {
	source, destination = elemType(source), elemType(destination)
	if source == destination || source.Kind() != reflect.Struct || destination.Kind() != reflect.Struct {
		return nil
	}

	var missing []string
	for _, field := range reflect.VisibleFields(destination) {
		if !field.IsExported() || field.Anonymous {
			continue
		}

		// Recurse into a matching field, since copier maps nested structs by name too
		sourceField, ok := source.FieldByNameFunc(func(name string) bool { return strings.EqualFold(name, field.Name) })
		if ok {
			missing = append(missing, unmappedFields(sourceField.Type, field.Type, prefix+field.Name+".")...)
			continue
		}

		if _, ok := reflect.PointerTo(source).MethodByName(field.Name); !ok {
			missing = append(missing, prefix+field.Name)
		}
	}
	return missing
}

// elemType unwraps pointers and slices to the type copier maps field by field
func elemType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	return t
}
