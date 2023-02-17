package model

import (
	"go-admin/common/models"
)

type ExportTask struct {
	ID       int    `json:"id" gorm:"primaryKey;autoIncrement;comment:主键编码"`
	DeptID   int    `json:"deptId"`
	Type     string `json:"type"`
	Args     string `json:"args"`
	Status   string `json:"status" gorm:"comment:当前查询状态"`
	FileName string `json:"fileName"`

	models.ModelTime
}

func (ExportTask) TableName() string {
	return "scrm_export_task"
}
