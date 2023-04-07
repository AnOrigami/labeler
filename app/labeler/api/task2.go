package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-admin/app/admin/models"
	"go-admin/app/labeler/service"
	"go-admin/app/scrm"
	"go-admin/common/actions"
	"go-admin/common/log"
)

func init() {
	routerCheckRole = append(routerCheckRole, task2AuthRouter())
}

func task2AuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/t2/upload", api.UploadTask2())
		g.POST("/api/v1/labeler/t2/search", api.SearchTask2())
		g.POST("/api/v1/labeler/t2/alloc/labeler", api.Task2BatchAllocLabeler())
		g.POST("/api/v1/labeler/t2/alloc/checker", api.Task2BatchAllocChecker())
		g.POST("/api/v1/labeler/t2/reset", api.ResetTasks2())
		g.PUT("/api/v1/labeler/t2/", api.UpdateTask2())
		g.POST("/api/v1/labeler/t2/batch/status", api.BatchSetTask2Status())
		g.POST("/api/v1/labeler/t2/my", api.SearchMyTask2())
		g.DELETE("/api/v1/labeler/t2/delete", api.DeleteTask2())
	}
}

func (api *LabelerAPI) UploadTask2() GinHandler {
	return func(c *gin.Context) {
		mf, err := c.MultipartForm()
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		files := mf.File["files"]
		projectID, err := primitive.ObjectIDFromHex(c.Request.FormValue("projectId"))
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		req := service.UploadTask2Req{
			Rows:      make([][]string, 0),
			ProjectID: projectID,
		}
		for _, fh := range files {
			rows, err := ReadFileHeaderExcel(fh)
			if err != nil {
				log.Logger().WithContext(c.Request.Context()).Error(err.Error())
				response.Error(c, 500, err, "")
				return
			}
			req.Rows = append(req.Rows, rows...)
		}
		resp, err := api.LabelerService.UploadTask2(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "")
	}
}

func (api *LabelerAPI) SearchTask2() GinHandler {
	return func(c *gin.Context) {
		var req service.SearchTask2Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "参数异常")
			return
		}
		p := actions.GetPermissionFromContext(c)
		req.UserID = p.UserId
		req.DataScope = p.DataScope

		resp, total, err := api.LabelerService.SearchTask2(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, req.PageIndex, req.PageSize, "查询成功")
	}
}

func (api *LabelerAPI) Task2BatchAllocLabeler() GinHandler {
	return func(c *gin.Context) {
		var req service.Task2BatchAllocLabelerReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "参数异常")
			return
		}
		if req.ProjectID.IsZero() {
			response.Error(c, 400, nil, "项目id不能为空")
			return
		}

		resp, err := api.LabelerService.Task2BatchAllocLabeler(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "分配成功")
	}
}

func (api *LabelerAPI) Task2BatchAllocChecker() GinHandler {
	return func(c *gin.Context) {
		roleID := user.GetRoleId(c)
		var role models.SysRole
		db := scrm.GormDB.WithContext(c.Request.Context()).First(&role, roleID)
		if err := db.Error; err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "db error")
			return
		}
		if !role.Admin {
			response.Error(c, 500, nil, "当前用户无分配审核任务权限")
			return
		}
		var req service.Task2BatchAllocCheckerReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		resp, err := api.LabelerService.Task2BatchAllocChecker(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "分配成功")
	}
}

func (api *LabelerAPI) ResetTasks2() GinHandler {
	return func(c *gin.Context) {
		p := actions.GetPermissionFromContext(c)
		if p.DataScope != "1" && p.DataScope != "2" {
			response.Error(c, http.StatusUnauthorized, nil, "当前用户没有操作权限")
			return
		}
		var req service.ResetTasks2Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "参数异常")
			return
		}
		if req.ProjectID.IsZero() {
			response.Error(c, 400, nil, "project id不能为空")
			return
		}

		resp, err := api.LabelerService.ResetTasks2(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "重置成功")
	}
}

func (api *LabelerAPI) UpdateTask2() GinHandler {
	return func(c *gin.Context) {
		var req service.UpdateTask2Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "参数异常")
			return
		}
		if req.ID.IsZero() {
			response.Error(c, 400, nil, "id不能为空")
			return
		}

		p := actions.GetPermissionFromContext(c)
		req.UserID = strconv.Itoa(p.UserId)
		req.UserDataScope = p.DataScope
		resp, err := api.LabelerService.UpdateTask2(c, req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "更新成功")
	}
}

func (api *LabelerAPI) BatchSetTask2Status() GinHandler {
	return func(c *gin.Context) {
		var req service.BatchSetTask2StatusReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		p := actions.GetPermissionFromContext(c)
		req.UserID = strconv.Itoa(p.UserId)
		req.UserDataScope = p.DataScope
		resp, err := api.LabelerService.BatchSetTask2Status(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "更新成功")
	}
}

func (api *LabelerAPI) SearchMyTask2() GinHandler {
	return func(c *gin.Context) {
		var req service.SearchMyTask2Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		p := actions.GetPermissionFromContext(c)
		req.UserID = strconv.Itoa(p.UserId)
		resp, total, err := api.LabelerService.SearchMyTask2(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, req.PageIndex, req.PageSize, "查询成功")
	}
}

func (api *LabelerAPI) DeleteTask2() GinHandler {
	return func(c *gin.Context) {
		id := c.Query("id")
		if len(id) == 0 {
			response.Error(c, 500, nil, "id不能为空")
			return
		}
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
		}

		if err := api.LabelerService.DeleteTask2(c.Request.Context(), objectID); err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, nil, "删除成功")
	}
}
