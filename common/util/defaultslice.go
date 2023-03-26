package util

type DefaultSlice[T any] []T

func (s DefaultSlice[T]) At(index int) (res T) {
	if index < len(s) {
		res = s[index]
	}
	return
}
