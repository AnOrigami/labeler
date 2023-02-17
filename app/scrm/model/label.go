package model

import "go-admin/common/models"

const (
	LabelNameCallNormal    = "接通"
	LabelNameCallHoldOn    = "占线"
	LabelNameCallDeny      = "拒接"
	LabelNameCallNoAnswer  = "未接"
	LabelNameCallNotExists = "空号"
	LabelNameCallOther     = "其他未接通"
)

type Label struct {
	ID int `json:"id" gorm:"primaryKey;autoIncrement;"`

	Name  string `json:"name" gorm:"size:255;not null;unique;"`
	Order int    `json:"order" gorm:"not null;default:10000;"`
	Type  string `json:"type" gorm:"size:255;not null;"`

	models.ModelTime
}

func (*Label) TableName() string {
	return "scrm_label"
}
