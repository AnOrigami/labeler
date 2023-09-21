package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-admin/app/labeler/service"
	"go-admin/common/actions"
	"go-admin/common/log"
)

func init() {
	routerCheckRole = append(routerCheckRole, task4AuthRouter())
}

func task4AuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/t4/upload", api.UploadTask4())
		g.POST("/api/v1/labeler/t4/search", api.SearchTask4())
		g.POST("/api/v1/labeler/t4/alloc/labeler", api.Task4BatchAllocLabeler())
		g.POST("/api/v1/labeler/t4/reset", api.ResetTasks4())
		g.PUT("/api/v1/labeler/t4/", api.UpdateTask4())
		g.POST("/api/v1/labeler/t4/batch/status", api.BatchSetTask4Status())
		g.POST("/api/v1/labeler/t4/my", api.SearchMyTask4())
		g.DELETE("/api/v1/labeler/t4/", api.DeleteTask4())
		g.POST("/api/v1/labeler/t4/download", api.DownloadTask4())
		g.POST("/api/v1/labeler/t4/detail", api.GetTask4())
	}
}

func (api *LabelerAPI) UploadTask4() GinHandler {
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

		req := service.UploadTask4Req{
			Rows:      make([]service.Task4FileRow, 0),
			ProjectID: projectID,
		}
		for _, fh := range files {
			rows, err := ReadFileHeaderExcel(fh)
			if err != nil {
				log.Logger().WithContext(c.Request.Context()).Error(err.Error())
				response.Error(c, 500, err, "")
				return
			}
			prefix := strings.Split(fh.Filename, ".")[0]
			for i, row := range rows {
				if i == 0 {
					continue
				}
				req.Rows = append(req.Rows, service.Task4FileRow{
					Name: prefix + "-" + strconv.Itoa(i),
					Data: row,
				})
			}
		}
		resp, err := api.LabelerService.UploadTask4(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "上传成功")
	}
}

func (api *LabelerAPI) SearchTask4() GinHandler {
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

		resp, total, err := api.LabelerService.SearchTask4(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, req.PageIndex, req.PageSize, "查询成功")
	}
}

func (api *LabelerAPI) Task4BatchAllocLabeler() GinHandler {
	return func(c *gin.Context) {
		var req service.Task4BatchAllocLabelerReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "参数异常")
			return
		}
		if req.ProjectID.IsZero() {
			response.Error(c, 400, nil, "项目id不能为空")
			return
		}

		resp, err := api.LabelerService.Task4BatchAllocLabeler(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "分配成功")
	}
}

func (api *LabelerAPI) ResetTasks4() GinHandler {
	return func(c *gin.Context) {
		p := actions.GetPermissionFromContext(c)
		if p.DataScope != "1" && p.DataScope != "2" {
			response.Error(c, http.StatusUnauthorized, nil, "当前用户没有操作权限")
			return
		}
		var req service.ResetTasks4Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "参数异常")
			return
		}
		if req.ProjectID.IsZero() {
			response.Error(c, 400, nil, "project id不能为空")
			return
		}

		resp, err := api.LabelerService.ResetTasks4(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "重置成功")
	}
}

func (api *LabelerAPI) UpdateTask4() GinHandler {
	return func(c *gin.Context) {
		var req service.UpdateTask4Req
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
		resp, err := api.LabelerService.UpdateTask4(c, req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "更新成功")
	}
}

func (api *LabelerAPI) BatchSetTask4Status() GinHandler {
	return func(c *gin.Context) {
		var req service.BatchSetTask4StatusReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		p := actions.GetPermissionFromContext(c)
		req.UserID = strconv.Itoa(p.UserId)
		req.UserDataScope = p.DataScope
		resp, err := api.LabelerService.BatchSetTask4Status(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, service.ResponseMap[req.Status])
	}
}

func (api *LabelerAPI) SearchMyTask4() GinHandler {
	return func(c *gin.Context) {
		var req service.SearchMyTask4Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		p := actions.GetPermissionFromContext(c)
		req.UserID = strconv.Itoa(p.UserId)
		resp, total, err := api.LabelerService.SearchMyTask4(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, req.PageIndex, req.PageSize, "查询成功")
	}
}

func (api *LabelerAPI) DeleteTask4() GinHandler {
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

		if err := api.LabelerService.DeleteTask4(c.Request.Context(), objectID); err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, nil, "删除成功")
	}
}

func (api *LabelerAPI) DownloadTask4() GinHandler {
	return func(c *gin.Context) {
		var req service.DownloadTask4Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		if req.ProjectID.IsZero() {
			response.Error(c, 500, nil, "项目id不能为空")
			return
		}
		if len(req.Status) == 0 {
			response.Error(c, 500, nil, "状态不能为空")
			return
		}
		resp, err := api.LabelerService.DownloadTask4(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "下载成功")
	}
}

func (api *LabelerAPI) GetTask4() GinHandler {
	return func(c *gin.Context) {
		var req service.GetTask4Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		if req.WorkType != 0 && req.WorkType != 1 && req.WorkType != 2 {
			response.Error(c, 500, nil, "参数异常")
			return
		}
		if req.ID.IsZero() {
			response.Error(c, 500, nil, "id不能为空")
			return
		}
		p := actions.GetPermissionFromContext(c)
		resp, err := api.LabelerService.GetTask4(c.Request.Context(), req, p)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "获取成功")
	}
}
