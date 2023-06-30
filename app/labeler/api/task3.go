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
	routerCheckRole = append(routerCheckRole, task3AuthRouter())
}

func task3AuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/t3/upload", api.UploadTask3())
		g.POST("/api/v1/labeler/t3/search", api.SearchTask3())
		g.POST("/api/v1/labeler/t3/alloc/labeler", api.Task3BatchAllocLabeler())
		g.POST("/api/v1/labeler/t3/reset", api.ResetTasks3())
		g.PUT("/api/v1/labeler/t3/", api.UpdateTask3())
		g.POST("/api/v1/labeler/t3/batch/status", api.BatchSetTask3Status())
		g.POST("/api/v1/labeler/t3/my", api.SearchMyTask3())
		g.DELETE("/api/v1/labeler/t3/", api.DeleteTask3())
		g.POST("/api/v1/labeler/t3/download", api.DownloadTask3())
		g.GET("/api/v1/labeler/t3/", api.GetTask3())
	}
}

func (api *LabelerAPI) UploadTask3() GinHandler {
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

		req := service.UploadTask3Req{
			Rows:      make([]service.Task3FileRow, 0),
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
				req.Rows = append(req.Rows, service.Task3FileRow{
					Name: prefix + "-" + strconv.Itoa(i),
					Data: row,
				})
			}
		}
		resp, err := api.LabelerService.UploadTask3(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "上传成功")
	}
}

func (api *LabelerAPI) SearchTask3() GinHandler {
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

		resp, total, err := api.LabelerService.SearchTask3(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, req.PageIndex, req.PageSize, "查询成功")
	}
}

func (api *LabelerAPI) Task3BatchAllocLabeler() GinHandler {
	return func(c *gin.Context) {
		var req service.Task3BatchAllocLabelerReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "参数异常")
			return
		}
		if req.ProjectID.IsZero() {
			response.Error(c, 400, nil, "项目id不能为空")
			return
		}

		resp, err := api.LabelerService.Task3BatchAllocLabeler(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "分配成功")
	}
}

func (api *LabelerAPI) ResetTasks3() GinHandler {
	return func(c *gin.Context) {
		p := actions.GetPermissionFromContext(c)
		if p.DataScope != "1" && p.DataScope != "2" {
			response.Error(c, http.StatusUnauthorized, nil, "当前用户没有操作权限")
			return
		}
		var req service.ResetTasks3Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "参数异常")
			return
		}
		if req.ProjectID.IsZero() {
			response.Error(c, 400, nil, "project id不能为空")
			return
		}

		resp, err := api.LabelerService.ResetTasks3(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "重置成功")
	}
}

func (api *LabelerAPI) UpdateTask3() GinHandler {
	return func(c *gin.Context) {
		var req service.UpdateTask3Req
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
		resp, err := api.LabelerService.UpdateTask3(c, req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "更新成功")
	}
}

func (api *LabelerAPI) BatchSetTask3Status() GinHandler {
	return func(c *gin.Context) {
		var req service.BatchSetTask3StatusReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		p := actions.GetPermissionFromContext(c)
		req.UserID = strconv.Itoa(p.UserId)
		req.UserDataScope = p.DataScope
		resp, err := api.LabelerService.BatchSetTask3Status(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, service.ResponseMap[req.Status])
	}
}

func (api *LabelerAPI) SearchMyTask3() GinHandler {
	return func(c *gin.Context) {
		var req service.SearchMyTask3Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		p := actions.GetPermissionFromContext(c)
		req.UserID = strconv.Itoa(p.UserId)
		resp, total, err := api.LabelerService.SearchMyTask3(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, req.PageIndex, req.PageSize, "查询成功")
	}
}

func (api *LabelerAPI) DeleteTask3() GinHandler {
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

		if err := api.LabelerService.DeleteTask3(c.Request.Context(), objectID); err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, nil, "删除成功")
	}
}

func (api *LabelerAPI) DownloadTask3() GinHandler {
	return func(c *gin.Context) {
		var req service.DownloadTask3Req
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
		resp, err := api.LabelerService.DownloadTask3(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "下载成功")
	}
}

func (api *LabelerAPI) GetTask3() GinHandler {
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

		resp, err := api.LabelerService.GetTask3(c.Request.Context(), objectID)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "获取成功")
	}
}
