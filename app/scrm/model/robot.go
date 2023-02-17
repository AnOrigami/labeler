package model

import (
	"go-admin/common/models"
)

type Robot struct {
	ID   int    `json:"id" gorm:"primaryKey;autoIncrement;comment:主键编码"`
	Name string `gorm:"size:255;"`

	DeptID int `gorm:"not null;default:0;"`

	models.ModelTime
	models.ControlBy
}

func (r Robot) TableName() string {
	return "scrm_robot"
}
