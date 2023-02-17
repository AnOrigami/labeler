package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go-admin/app/scrm"
	"go-admin/app/scrm/service"
)

func SearchProjects(c *gin.Context) {
	var req service.SearchProjectsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "查询失败")
		return
	}
	resp, total, err := service.SearchProjects(c.Request.Context(), req)
	if err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "查询失败")
		return
	}
	response.PageOK(c, resp, int(total), req.GetPageIndex(), req.GetPageSize(), "查询成功")
}

func CreateProject(c *gin.Context) {
	var req service.CreateProjectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "创建失败")
		return
	}

	req.SetCreateBy(user.GetUserId(c))

	err := service.CreateProject(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, service.CreateProjectResp{}, "创建成功")
}

func DeleteProject(c *gin.Context) {
	req := service.DeleteProjectReq{}
	if err := c.ShouldBindQuery(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "删除失败")
		return
	}
	if req.ID <= 0 {
		response.Error(c, 200, nil, "id为空")
		return
	}

	req.SetUpdateBy(user.GetUserId(c))

	err := service.DeleteProject(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "删除失败")
		return
	}
	response.OK(c, service.DeleteProjectResp{}, "删除成功")
}

func GetProjectDetail(c *gin.Context) {
	req := service.GetProjectDetailReq{}
	if err := c.ShouldBindQuery(&req); err != nil {
		scrm.Logger().Error(err.Error())
		response.Error(c, 500, err, "匹配project detail的参数失败")
		return
	}
	if req.ID <= 0 {
		response.Error(c, 200, nil, "id为空")
		return
	}
	detail, err := service.GetProjectDetail(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, detail, "获取成功")
}

func GetSeatDetailOfProject(c *gin.Context) {
	req := service.GetProjectDetailReq{}
	if err := c.ShouldBindQuery(&req); err != nil {
		scrm.Logger().Error(err.Error())
		response.Error(c, 500, err, "匹配project detail的参数失败")
		return
	}
	if req.ID <= 0 {
		response.Error(c, 200, nil, "id为空")
		return
	}
	detail, err := service.GetSeatDetailOfProject(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, detail, "获取成功")
}

func UnlockProjectSeat(c *gin.Context) {
	req := service.UnlockProjectReq{}
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().Error(err.Error())
		response.Error(c, 500, err, "匹配参数失败")
		return
	}
	err := service.UnlockProjectSeat(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, gin.H{}, "解锁成功")
}

func RunProject(c *gin.Context) {
	var req service.RunProjectReq
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

	err := service.RunProject(c.Request.Context(), req)
	if err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, service.RunProjectResp{}, "切换成功")
}

func SetProjectSeats(c *gin.Context) {
	var req service.SetProjectSeatsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	if req.ID <= 0 {
		response.Error(c, 200, nil, "参数为空")
		return
	}

	req.SetUpdateBy(user.GetUserId(c))

	err := service.SetProjectSeats(c.Request.Context(), req)
	if err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, service.SetProjectSeatsResp{}, "设置成功")
}

func SetProjectRobots(c *gin.Context) {
	var req service.SetProjectRobotsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	if req.ID <= 0 || len(req.Robots) == 0 {
		response.Error(c, 200, nil, "参数为空")
		return
	}

	req.SetUpdateBy(user.GetUserId(c))

	err := service.SetProjectRobots(c.Request.Context(), req)
	if err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, service.SetProjectRobotsResp{}, "设置成功")
}

func SetProjectConcurrency(c *gin.Context) {
	var req service.SetProjectConcurrencyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "")
		return
	}
	resp, err := service.SetProjectConcurrency(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, resp, "成功")
}
