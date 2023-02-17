package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go-admin/app/scrm"
	"go-admin/app/scrm/service"
	"go-admin/common/actions"
	"net/http"
	"strconv"
)

func UploadOrderGroup(c *gin.Context) {
	var req service.UploadOrderGroupReq
	file, fh, err := c.Request.FormFile("file")
	if err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "文件解析异常")
		return
	}
	defer func() {
		if err = file.Close(); err != nil {
			scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		}
	}()
	projectIDStr := c.PostForm("projectId")
	if len(projectIDStr) == 0 {
		response.Error(c, 500, err, "参数为空")
		return
	}
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}

	p := actions.GetPermissionFromContext(c)
	if p.DeptId == 0 {
		response.Error(c, http.StatusUnauthorized, nil, "获取用户信息失败")
		return
	}
	req.ProjectID = projectID
	req.File = file
	req.Filename = fh.Filename
	req.SetCreateBy(user.GetUserId(c))
	req.DeptID = p.DeptId
	err = service.UploadOrderGroupAndCreateOrders(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, nil, err.Error())
		return
	}
	response.OK(c, service.CreateProjectResp{}, "上传成功")
}

func SearchOrderGroup(c *gin.Context) {
	var req service.SearchOrderGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	resp, total, err := service.SearchOrderGroup(c.Request.Context(), req)
	if err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "")
		return
	}
	response.PageOK(c, resp, int(total), req.GetPageIndex(), req.GetPageSize(), "查询成功")
}

func GetOrderList(c *gin.Context) {
	var req service.GetOrderListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	resp, total, err := service.GetOrderList(c.Request.Context(), req)
	if err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "")
		return
	}
	response.PageOK(c, resp, int(total), req.GetPageIndex(), req.GetPageSize(), "查询成功")
}

func GetOrderDetail(c *gin.Context) {
	req := service.GetOrderDetailReq{}
	if err := c.ShouldBindQuery(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	if req.ID <= 0 {
		response.Error(c, 200, nil, "id为空")
		return
	}
	order, err := service.GetOrderDetail(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, order, "获取成功")
}

func DeleteOrder(c *gin.Context) {
	req := service.DeleteOrderReq{}
	if err := c.ShouldBindQuery(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	if req.ID <= 0 {
		response.Error(c, 200, nil, "id为空")
		return
	}
	err := service.DeleteOrder(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, service.DeleteProjectResp{}, "删除成功")
}

func ExportOrders(c *gin.Context) {
	var req service.GetOrderListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	resp, err := service.ExportOrders(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, resp, "导出成功")
}
