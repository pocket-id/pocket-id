package utils

// PtrOrNil returns a pointer to v if v is not the zero value of its type,
// otherwise it returns nil.
func PtrOrNil[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}
