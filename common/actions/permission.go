package actions

import (
	"context"
	"errors"
	"fmt"
	"go-admin/app/scrm"
	"go-admin/common/gormscope"
	"go-admin/common/util"

	"github.com/gin-gonic/gin"
	log "github.com/go-admin-team/go-admin-core/logger"
	"github.com/go-admin-team/go-admin-core/sdk/config"
	"github.com/go-admin-team/go-admin-core/sdk/pkg"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"gorm.io/gorm"
)

type DataPermission struct {
	DataScope string
	UserId    int
	DeptId    int
	RoleId    int
}

func PermissionAction() gin.HandlerFunc {
	return func(c *gin.Context) {
		db, err := pkg.GetOrm(c)
		if err != nil {
			log.Error(err)
			return
		}
		db = db.WithContext(c.Request.Context())

		msgID := pkg.GenerateMsgIDFromContext(c)
		var p = new(DataPermission)
		if userId := user.GetUserIdStr(c); userId != "" {
			p, err = newDataPermission(db, userId)
			if err != nil {
				log.Errorf("MsgID[%s] PermissionAction error: %s", msgID, err)
				response.Error(c, 500, err, "权限范围鉴定错误")
				c.Abort()
				return
			}
		}
		c.Set(PermissionKey, p)
		c.Next()
	}
}

func newDataPermission(tx *gorm.DB, userId interface{}) (*DataPermission, error) {
	var err error
	p := &DataPermission{}

	err = tx.Table("sys_user").
		Select("sys_user.user_id", "sys_role.role_id", "sys_user.dept_id", "sys_role.data_scope").
		Joins("left join sys_role on sys_role.role_id = sys_user.role_id").
		Where("sys_user.user_id = ?", userId).
		Scan(p).Error
	if err != nil {
		err = errors.New("获取用户数据出错 msg:" + err.Error())
		return nil, err
	}
	return p, nil
}

func Permission(tableName string, p *DataPermission) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if !config.ApplicationConfig.EnableDP {
			return db
		}
		switch p.DataScope {
		case "2":
			return db.Where(tableName+".create_by in (select sys_user.user_id from sys_role_dept left join sys_user on sys_user.dept_id=sys_role_dept.dept_id where sys_role_dept.role_id = ?)", p.RoleId)
		case "3":
			return db.Where(tableName+".create_by in (SELECT user_id from sys_user where dept_id = ? )", p.DeptId)
		case "4":
			return db.Where(tableName+".create_by in (SELECT user_id from sys_user where sys_user.dept_id in(select dept_id from sys_dept where dept_path like ? ))", "%/"+pkg.IntToString(p.DeptId)+"/%")
		case "5":
			return db.Where(tableName+".create_by = ?", p.UserId)
		default:
			return db
		}
	}
}

// GetPermissionFromContext 提供非action写法数据范围约束
func GetPermissionFromContext(ctx context.Context) *DataPermission {
	p := new(DataPermission)
	c := scrm.GinContext(ctx)
	if c != nil {
		if pm := c.Value(PermissionKey); pm != nil {
			util.Set(pm, &p)
		}
	}
	return p
}

func DeptPermission(ctx context.Context, table string) gormscope.Scope {
	return func(db *gorm.DB) *gorm.DB {
		deptID := GetPermissionFromContext(ctx).DeptId
		if deptID == 0 {
			return db
		}
		return db.Where(
			table+".dept_id in (select dept_id from sys_dept where sys_dept.dept_path like ?)",
			fmt.Sprintf("%%/%d/%%", deptID),
		)
	}
}

func DeptPermissionFromDeptId(deptId int, table string) gormscope.Scope {
	return func(db *gorm.DB) *gorm.DB {
		deptID := deptId
		if deptID == 0 {
			return db
		}
		return db.Where(
			table+".dept_id in (select dept_id from sys_dept where sys_dept.dept_path like ?)",
			fmt.Sprintf("%%/%d/%%", deptID),
		)
	}
}
