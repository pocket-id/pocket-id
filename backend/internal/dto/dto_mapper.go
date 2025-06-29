package dto

import (
	"fmt"

	"github.com/jinzhu/copier"
)

// MapStructList maps a list of source structs to a list of destination structs
func MapStructList[S any, D any](source []S, destination *[]D) (err error) {
	*destination = make([]D, len(source))

	for i, item := range source {
		err = MapStruct(item, &((*destination)[i]))
		if err != nil {
			return fmt.Errorf("failed to map field %d: %w", i, err)
		}
	}
	return nil
}

// MapStruct maps a source struct to a destination struct
func MapStruct(source any, destination any) error {
	return copier.CopyWithOption(destination, source, copier.Option{
		DeepCopy: true,
	})
}
