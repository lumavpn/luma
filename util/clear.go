//go:build go1.21

package util

func ClearArray[T ~[]E, E any](t T) {
	clear(t)
}

func ClearMap[T ~map[K]V, K comparable, V any](t T) {
	clear(t)
}
