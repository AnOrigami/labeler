package database

import (
	"gorm.io/gorm"
	"strings"
)

type OrderColumn struct {
	ColumnName string
	Asc        bool
}

func Apply[I, O any](is []I, f func(I) O) []O {
	out := make([]O, len(is))
	for i := range is {
		out[i] = f(is[i])
	}
	return out
}

func Range(start, stop, step int) []int {
	res := make([]int, 0)
	for i := start; i < stop; i += step {
		res = append(res, i)
	}
	return res
}

/*
BatchQueryExcludeNull
目前得保证入参的数据库列不能有空,存在空值的行会被抛弃
batch要大于0
next返回值的长度为0,表示没有下一批了,如果继续执行next,会得到nil和error
*/
func BatchQueryExcludeNull[Table any](db *gorm.DB, batch int, cols []OrderColumn, valueFunc func(Table) []any) (
	next func() ([]Table, error),
) {
	orderQuery := strings.Join(Apply(cols, func(c OrderColumn) string {
		if c.Asc {
			return c.ColumnName
		} else {
			return c.ColumnName + " desc"
		}
	}), ",")
	whereQuery := "(" + strings.Join(Apply(Range(1, len(cols)+1, 1), func(i int) string {
		equalsPart := strings.Join(Apply(Range(1, i, 1), func(j int) string {
			return "(" + cols[j-1].ColumnName + "=?)"
		}), "and")
		lastSymbol := ">"
		if !cols[i-1].Asc {
			lastSymbol = "<"
		}
		lastPart := "(" + cols[i-1].ColumnName + lastSymbol + "?)"
		if equalsPart == "" {
			return lastPart
		} else {
			return "(" + equalsPart + ")and" + lastPart
		}
	}), "or") + ")"
	whereArgsGenerator := func(t Table) []any {
		values := valueFunc(t)
		args := make([]any, 0, (1+len(values))*len(values)/2)
		for i := 1; i < len(cols)+1; i++ {
			for j := 1; j < i+1; j++ {
				args = append(args, values[j-1])
			}
		}
		return args
	}
	var (
		receiver []Table
		first    = true
		nextArg  []any
		finished = false
	)
	db = db.Order(orderQuery).Limit(batch)
	next = func() ([]Table, error) {
		batchDB := db.Session(&gorm.Session{})
		if finished {
			return receiver, nil
		}
		if !first {
			batchDB = batchDB.Where(whereQuery, nextArg...)
		} else {
			first = false
			batchDB = batchDB.Where("(" + strings.Join(Apply(cols, func(i OrderColumn) string {
				return "(" + i.ColumnName + " is not null)"
			}), "and") + ")")
		}
		if err := batchDB.Find(&receiver).Error; err != nil {
			return nil, err
		}
		if len(receiver) == 0 {
			finished = true
		} else {
			nextArg = whereArgsGenerator(receiver[len(receiver)-1])
		}
		return receiver, nil
	}
	return
}
