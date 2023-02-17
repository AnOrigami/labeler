package util

// 基于map实现的集合

type Collect map[any]struct{}

func MakeCollect() Collect {
	return map[any]struct{}{}
}

func (c Collect) Add(item any) {
	c[item] = struct{}{}
}

func (c Collect) Delete(item any) {
	delete(c, item)
}

func (c Collect) Size() int {
	return len(c)
}

func (c Collect) Loop(fun func(item any) any) []any {
	res := make([]any, 0, c.Size())
	for i := range c {
		res = append(res, fun(i))
	}
	return res
}

func (c Collect) Exist(item any) bool {
	_, exist := c[item]
	return exist
}

func (c Collect) Export() []any {
	res := make([]any, 0, c.Size())
	for i := range c {
		res = append(res, i)
	}
	return res
}

/* shell

由于没有泛型方法，可以通过一下shell手动泛型


yourType="int"

echo "

type CollectT${yourType} map[${yourType}]struct{}

func MakeCollectT${yourType}() CollectT${yourType} {
	return map[${yourType}]struct{}{}
}

func (c CollectT${yourType}) Add(item ${yourType}) {
	c[item] = exist
}

func (c CollectT${yourType}) Delete(item ${yourType}) {
	delete(c, item)
}

func (c CollectT${yourType}) Size() int {
	return len(c)
}

func (c CollectT${yourType}) Loop(fun func(item ${yourType}) ${yourType}) []${yourType} {
	res := make([]${yourType}, 0, c.Size())
	for i := range c {
		res = append(res, fun(i))
	}
	return res
}

func (c CollectT${yourType}) Exist(item ${yourType}) bool {
	_, exist := c[item]
	return exist
}

func (c CollectT${yourType}) Export() []${yourType} {
	res := make([]${yourType}, 0, c.Size())
	for i := range c {
		res = append(res, i)
	}
	return res
}


*/

type CollectTint map[int]struct{}

func MakeCollectTint() CollectTint {
	return map[int]struct{}{}
}

func (c CollectTint) Add(item int) {
	c[item] = struct{}{}
}

func (c CollectTint) Delete(item int) {
	delete(c, item)
}

func (c CollectTint) Size() int {
	return len(c)
}

func (c CollectTint) Loop(fun func(item int) int) []int {
	res := make([]int, 0, c.Size())
	for i := range c {
		res = append(res, fun(i))
	}
	return res
}

func (c CollectTint) Exist(item int) bool {
	_, exist := c[item]
	return exist
}

func (c CollectTint) Export() []int {
	res := make([]int, 0, c.Size())
	for i := range c {
		res = append(res, i)
	}
	return res
}
