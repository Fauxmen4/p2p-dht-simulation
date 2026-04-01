package utils

import "math/rand"

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// RandString generates random string of length n.
// It is used to generate (key, value) data in different workload scenarios.
// TODO: replace (or add) (key, value) data of much bigger size along with ordinary datasets to make scenarios more realistic.
func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// RandomElements returns n random elements from the given slice.
// Inputs are not modified.
func RandomElements[T any](slice []T, n int) []T {
	if len(slice) <= n {
		return slice
	}

	if n == 0 {
		return []T{}
	}

	shuffled := make([]T, len(slice))
	copy(shuffled, slice)

	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:n]
}
