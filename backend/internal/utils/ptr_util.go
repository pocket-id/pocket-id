package utils

func Ptr[T any](v T) *T {
	return &v
}

func PtrValueOrZero[T any](ptr *T) T {
	if ptr == nil {
		var zero T
		return zero
	}
	return *ptr
}
