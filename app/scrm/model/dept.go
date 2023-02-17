package model

type Dept struct {
	DeptId   int    `json:"deptId" gorm:"primaryKey;autoIncrement;"` //部门编码
	ParentId int    `json:"parentId" gorm:""`                        //上级部门
	DeptPath string `json:"deptPath" gorm:"size:255;"`               //
	DeptName string `json:"deptName"  gorm:"size:128;"`              //部门名称
	Leader   string `json:"leader" gorm:"size:128;"`                 //负责人
}

func (Dept) TableName() string {
	return "sys_dept"
}
