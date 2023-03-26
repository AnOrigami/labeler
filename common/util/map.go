package util

func Map[I any, O any](req []I, fn func(v I) O) []O {
	res := make([]O, len(req))
	for i, v := range req {
		res[i] = fn(v)
	}
	return res
}
