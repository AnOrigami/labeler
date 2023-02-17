package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go-admin/app/scrm"
	"go-admin/app/scrm/service"
	"go-admin/common/actions"
)

func CreateRobot(c *gin.Context) {
	var req service.CreateRobotReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}

	req.SetCreateBy(user.GetUserId(c))
	p := actions.GetPermissionFromContext(c)
	req.DeptID = p.DeptId

	err := service.CreateRobot(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, service.CreateRobotResp{}, "创建成功")
}

func DeleteRobot(c *gin.Context) {
	req := service.DeleteRobotReq{}
	if err := c.ShouldBindQuery(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	if req.ID <= 0 {
		response.Error(c, 200, nil, "id为空")
		return
	}
	err := service.DeleteRobot(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, service.DeleteRobotResp{}, "删除成功")
}

func UpdateRobot(c *gin.Context) {
	var req service.UpdateRobotReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	if req.ID <= 0 {
		response.Error(c, 200, nil, "id为空")
		return
	}

	req.SetUpdateBy(user.GetUserId(c))

	err := service.UpdateRobot(c.Request.Context(), req)
	if err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, service.RunProjectResp{}, "更新成功")
}

func SearchRobots(c *gin.Context) {
	var req service.SearchRobotsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	resp, total, err := service.SearchRobots(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.PageOK(c, resp, int(total), req.GetPageIndex(), req.GetPageSize(), "查询成功")
}

func GetRobot(c *gin.Context) {
	req := service.GetRobotReq{}
	if err := c.ShouldBindQuery(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	if req.ID <= 0 {
		response.Error(c, 200, nil, "id为空")
		return
	}
	robot, err := service.GetRobot(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, robot, "获取成功")
}
