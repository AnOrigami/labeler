package model

import "go-admin/common/models"

type Order struct {
	ID           int    `json:"id" gorm:"primaryKey;autoIncrement;comment:主键编码"`
	Code         string `json:"code" gorm:"size:255"`
	Phone        string `json:"phone" gorm:"type:longtext;"` // 暂时和call中的类型保持一致
	Status       string `json:"status" gorm:"size:255"`
	Sex          string `json:"sex" gorm:"size:20"`
	ProjectID    int    `json:"projectId"`
	Project      *Project
	DeptID       int    `json:"deptId" gorm:"not null;"`
	OrderGroupID int    `json:"orderGroupId" gorm:"not null;"`
	Calls        []Call `gorm:"foreignKey:OrderID"`

	models.ModelTime
	models.ControlBy
}

func (Order) TableName() string {
	return "scrm_order"
}

type OrderGroup struct {
	ID        int     `json:"id" gorm:"primaryKey;autoIncrement;comment:主键编码"`
	Filename  string  `json:"filename" gorm:"size:255;not null;default:'';"`
	Count     int     `json:"count" gorm:"not null;"`
	ProjectID int     `json:"projectId" gorm:"not null;"`
	DeptID    int     `json:"deptId" gorm:"not null;"`
	Orders    []Order `json:"orders" gorm:"foreignKey:order_group_id"`

	models.ModelTime
	models.ControlBy
}

func (OrderGroup) TableName() string {
	return "scrm_order_group"
}
