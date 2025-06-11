package utils

func Ptr[K any](v K) *K {
	return &v
}
