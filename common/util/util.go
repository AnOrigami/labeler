package util

func Set[T any](i any, target *T) {
	if v, ok := i.(T); ok {
		*target = v
	}
}

func HidePhone(phone string) (result string) {
	return phone
	//if len(phone) <= 4 {
	//	return phone
	//}
	//return strings.Repeat("*", (len(phone)-4)) + phone[len(phone)-4:]
}

func Convert[S any, T any](source []S, f func(s S) T) []T {
	result := make([]T, len(source))
	for i, s := range source {
		result[i] = f(s)
	}
	return result
}
