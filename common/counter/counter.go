package counter

type Counter[T comparable] map[T]int

func (c Counter[T]) PopMax() (T, int) {
	var (
		maxKey T
		max    = -1
	)
	for key, n := range c {
		if n > max {
			maxKey, max = key, n
		}
	}
	delete(c, maxKey)
	return maxKey, max
}

func (c Counter[T]) Inc(key T, n int) {
	c[key] += n
}

func (c Counter[T]) IncIfExists(key T, n int) {
	if _, ok := c[key]; ok {
		c[key] += n
	}
}
