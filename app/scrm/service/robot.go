package service

import (
	"context"
	"errors"
	"go-admin/app/scrm"
	"go-admin/app/scrm/model"
	"go-admin/common/actions"
	"go-admin/common/gormscope"
	common "go-admin/common/models"
	"gorm.io/gorm"
)

type CreateRobotReq struct {
	Name   string `json:"name"`
	DeptID int

	common.ControlBy
}

type CreateRobotResp struct{}

func CreateRobot(ctx context.Context, req CreateRobotReq) error {
	var (
		err   error
		robot model.Robot
		count int64
	)
	err = scrm.GormDB.WithContext(ctx).Model(&robot).Where("name = ?", req.Name).Count(&count).Error
	if err != nil {
		scrm.Logger().WithContext(ctx).Error("db error", err.Error())
		return err
	}
	if count > 0 {
		return errors.New("机器人名称已存在")
	}

	m := model.Robot{
		Name:   req.Name,
		DeptID: req.DeptID,
		ControlBy: common.ControlBy{
			CreateBy: req.CreateBy,
		},
	}
	db := scrm.GormDB.WithContext(ctx).Create(&m)
	if db.Error != nil {
		scrm.Logger().WithContext(ctx).Error("db error", err.Error())
		return err
	}
	return nil
}

type DeleteRobotReq struct {
	ID int `form:"id"`
}

type DeleteRobotResp struct{}

func DeleteRobot(ctx context.Context, req DeleteRobotReq) error {
	db := scrm.GormDB.WithContext(ctx).
		Scopes(
			actions.DeptPermission(ctx, model.Robot{}.TableName()),
		).Delete(&model.Robot{}, req.ID)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error(ctx, "Error found in delete robot :", err.Error())
		return err
	}
	if db.RowsAffected == 0 {
		return errors.New("无权删除该数据")
	}
	return nil
}

type UpdateRobotReq struct {
	ID   int    `json:"id"`
	Name string `json:"name"`

	common.ControlBy
}

type UpdateRobotResp struct{}

func UpdateRobot(ctx context.Context, req UpdateRobotReq) error {
	m := model.Robot{}
	db := scrm.GormDB.WithContext(ctx).
		Scopes(
			actions.DeptPermission(ctx, m.TableName()),
		).
		Find(&m, req.ID)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("update robot error", err.Error())
		return err
	}
	if db.RowsAffected == 0 {
		return errors.New("无权更新该数据")
	}

	m.Name = req.Name
	db = scrm.GormDB.WithContext(ctx).
		Select("name").
		Updates(&m)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("update robot error", err.Error())
		return err
	}
	return nil
}

type SearchRobotsReq struct {
	RobotName string `json:"robotName"`
	Pagination
}

type RobotResponseItem struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Creator int    `json:"creator"`
}

func SearchRobots(ctx context.Context, req SearchRobotsReq) ([]RobotResponseItem, int64, error) {
	var (
		robots []*model.Robot
		count  int64
	)
	db := scrm.GormDB.WithContext(ctx).
		Scopes(
			//actions.DeptPermission(ctx, model.Robot{}.TableName()),
			gormscope.Paginate(&req.Pagination),
			SearchRobotsScope(req),
		).
		Find(&robots)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("db error", err.Error())
		return nil, 0, err
	}
	db = db.Limit(-1).Offset(-1).Count(&count)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("db error", err.Error())
		return nil, 0, err
	}
	items := make([]RobotResponseItem, len(robots))
	for i, robot := range robots {
		items[i] = RobotResponseItem{
			ID:      robot.ID,
			Name:    robot.Name,
			Creator: robot.CreateBy,
		}
	}
	return items, count, nil
}

func SearchRobotsScope(req SearchRobotsReq) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(req.RobotName) > 0 {
			db = db.Where("scrm_robot.name = ?", req.RobotName)
		}
		return db
	}
}

type GetRobotReq struct {
	ID int `json:"id" form:"id"`
}

type GetRobotRes struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	CreatorName string `json:"creatorName" gorm:"column:username"`
}

func GetRobot(ctx context.Context, req GetRobotReq) (GetRobotRes, error) {
	res := GetRobotRes{}
	err := scrm.GormDB.WithContext(ctx).Table("scrm_robot").
		Joins("left join sys_user on scrm_robot.create_by = sys_user.user_id").
		Select("scrm_robot.name,scrm_robot.id,sys_user.username").
		Where("scrm_robot.id = ?", req.ID).
		Scan(&res).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return GetRobotRes{}, errors.New("查看对象不存在或无权查看")
	}
	if err != nil {
		scrm.Logger().WithContext(ctx).Error("db error", err.Error())
		return GetRobotRes{}, err
	}
	return res, nil
}
