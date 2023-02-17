package model

import "go-admin/common/models"

type Project struct {
	ID         int     `json:"id" gorm:"primaryKey;autoIncrement;comment:主键编码"`
	Name       string  `json:"name" gorm:"size:255;not null;default:'';"`
	DeptID     int     `json:"deptId" gorm:"not null;"`
	Dept       *Dept   `json:"dept"`
	RobotID    *int    `json:"robotId"`
	Robot      *Robot  `json:"robot"`
	SpareSeatC float64 `gorm:"not null;"`
	BusySeatC  float64 `gorm:"not null;"`
	Running    bool    `json:"running" gorm:"not null;default:false;"`
	Seats      []Seat  `json:"seats" gorm:"many2many:scrm_project_seat"`

	models.ModelTime
	models.ControlBy
}

func (p Project) TableName() string {
	return "scrm_project"
}
