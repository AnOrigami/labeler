package gormscope

import (
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Scope = func(db *gorm.DB) *gorm.DB

func SelectForUpdate() Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
}

func Join(name string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Joins(name)
	}
}

type Paginator interface {
	GetPageIndex() int
	GetPageSize() int
}

func Paginate(p Paginator) Scope {
	return func(db *gorm.DB) *gorm.DB {
		offset := (p.GetPageIndex() - 1) * p.GetPageSize()
		if offset < 0 {
			offset = 0
		}
		return db.Offset(offset).Limit(p.GetPageSize())
	}
}

func CreateDateRange(start, end, tableName string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		if len(start) > 0 {
			db = db.Where(fmt.Sprintf("date(%s.created_at)>=?", tableName), start)
		}
		if len(end) > 0 {
			db = db.Where(fmt.Sprintf("date(%s.created_at)<=?", tableName), end)
		}
		return db
	}
}

func CreateDateTimeRange(start, end, tableName string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		if len(start) > 0 {
			db = db.Where(fmt.Sprintf("%s.created_at>=?", tableName), start)
		}
		if len(end) > 0 {
			db = db.Where(fmt.Sprintf("%s.created_at<=?", tableName), end)
		}
		return db
	}
}
