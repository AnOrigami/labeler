package model

import "go-admin/common/models"

type User struct {
	UserId   int    `gorm:"primaryKey;autoIncrement;comment:编码"  json:"userId"`
	Username string `json:"username" gorm:"size:64;comment:用户名"`
	NickName string `json:"nickName" gorm:"size:128;comment:昵称"`
	RoleId   int    `json:"roleId" gorm:"size:20;comment:角色ID"`
	DeptId   int    `json:"deptId" gorm:"size:20;comment:部门"`
	//Dept     *Dept  `json:"dept" gorm:"references:dept_id"`

	models.ControlBy
	models.ModelTime
}

func (User) TableName() string {
	return "sys_user"
}
