package model

import (
	"database/sql"
	"go-admin/common/models"
)

type Seat struct {
	ID         int    `gorm:"primaryKey;autoIncrement;comment:主键编码"`
	Nickname   string `gorm:"size:255"`
	UserID     int
	User       *User
	Projects   []Project      `gorm:"many2many:scrm_project_seat"`
	DeptID     int            `gorm:"not null;default:0;"`
	Line       sql.NullString `gorm:"size:20;unique;"`
	LineGroup  sql.NullString `gorm:"size:20;"`
	WeCom      string         `gorm:"size:127;"`
	WeComRobot string         `gorm:"size:127;"`
	models.ModelTime
	models.ControlBy
}

func (s Seat) TableName() string {
	return "scrm_seat"
}
