package api

import (
	"encoding/json"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-admin/app/labeler/service"
	"go-admin/common/actions"
	"go-admin/common/log"
)

func init() {
	routerCheckRole = append(routerCheckRole, task6AuthRouter())
}

func task6AuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/t6/upload", api.UploadTask6())
		g.POST("/api/v1/labeler/t6/search", api.SearchTask6())
		g.POST("/api/v1/labeler/t6/alloc/labeler", api.Task6BatchAllocLabeler())
		g.POST("/api/v1/labeler/t6/reset", api.ResetTasks6())
		g.PUT("/api/v1/labeler/t6/", api.UpdateTask6())
		g.POST("/api/v1/labeler/t6/batch/status", api.BatchSetTask6Status())
		g.POST("/api/v1/labeler/t6/my", api.SearchMyTask6())
		g.DELETE("/api/v1/labeler/t6/", api.DeleteTask6())
		g.POST("/api/v1/labeler/t6/download", api.DownloadTask6())
		g.POST("/api/v1/labeler/t6/detail", api.GetTask6())
		g.POST("/api/v1/labeler/t6/mycount", api.SearchMyTask6Count())
	}
}

func (api *LabelerAPI) UploadTask6() GinHandler {
	return func(c *gin.Context) {
		mf, err := c.MultipartForm()
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		files := mf.File["files"]
		var tasks []bson.M
		var filenames []string
		var projectID primitive.ObjectID
		for _, fh := range files {

			filename := fh.Filename
			filenames = append(filenames, filename)
			file, _ := fh.Open()
			d, err := ioutil.ReadAll(file)
			if err != nil {
				log.Logger().WithContext(c.Request.Context()).Error(err.Error())
				response.Error(c, 500, err, "")
				return
			}
			defer func(file multipart.File) {
				err := file.Close()
				if err != nil {
					log.Logger().WithContext(c.Request.Context()).Error(err.Error())
					response.Error(c, 500, err, "")
					return
				}
			}(file)
			var filedata bson.M
			err = json.Unmarshal(d, &filedata)
			if err != nil {
				log.Logger().WithContext(c.Request.Context()).Error(err.Error())
				response.Error(c, 400, err, "")
				return
			}
			projectID, err = primitive.ObjectIDFromHex(c.Request.FormValue("projectId"))
			if err != nil {
				log.Logger().WithContext(c.Request.Context()).Error(err.Error())
				response.Error(c, 400, err, "")
				return
			}

			tasks = append(tasks, filedata)

		}
		req := service.UploadTask6Req{
			Tasks6:    tasks,
			ProjectID: projectID,
			Name:      filenames,
		}
		resp, err := api.LabelerService.UploadTask6(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "上传成功")
	}

}

func (api *LabelerAPI) SearchTask6() GinHandler {
	return func(c *gin.Context) {
		var req service.SearchTask6Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "参数异常")
			return
		}
		p := actions.GetPermissionFromContext(c)
		req.UserID = p.UserId
		req.DataScope = p.DataScope

		resp, total, err := api.LabelerService.SearchTask6(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, req.PageIndex, req.PageSize, "查询成功")
	}
}

func (api *LabelerAPI) Task6BatchAllocLabeler() GinHandler {
	return func(c *gin.Context) {
		var req service.Task6BatchAllocLabelerReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "参数异常")
			return
		}
		if req.ProjectID.IsZero() {
			response.Error(c, 400, nil, "项目id不能为空")
			return
		}

		resp, err := api.LabelerService.Task6BatchAllocLabeler(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "分配成功")
	}
}

func (api *LabelerAPI) ResetTasks6() GinHandler {
	return func(c *gin.Context) {
		p := actions.GetPermissionFromContext(c)
		if p.DataScope != "1" && p.DataScope != "2" {
			response.Error(c, http.StatusUnauthorized, nil, "当前用户没有操作权限")
			return
		}
		var req service.ResetTasks6Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "参数异常")
			return
		}
		if req.ProjectID.IsZero() {
			response.Error(c, 400, nil, "project id不能为空")
			return
		}
		if len(req.Statuses) == 0 {
			response.Error(c, 400, nil, "状态不能为空")
			return
		}
		if len(req.Persons) == 0 {
			response.Error(c, 400, nil, "人员不能为空")
			return
		}
		if req.ResetType != 0 && req.ResetType != 1 {
			response.Error(c, 400, nil, "重置类型错误")
			return
		}
		resp, err := api.LabelerService.ResetTasks6(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "重置成功")
	}
}

func (api *LabelerAPI) UpdateTask6() GinHandler {
	return func(c *gin.Context) {
		var req service.UpdateTask6Req
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
		resp, err := api.LabelerService.UpdateTask6(c, req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "更新成功")
	}
}

func (api *LabelerAPI) BatchSetTask6Status() GinHandler {
	return func(c *gin.Context) {
		var req service.BatchSetTask6StatusReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		if req.WorkType != 0 && req.WorkType != 1 && req.WorkType != 2 {
			response.Error(c, 400, nil, "类型参数异常")
			return
		}
		p := actions.GetPermissionFromContext(c)
		req.UserID = strconv.Itoa(p.UserId)
		req.UserDataScope = p.DataScope
		resp, err := api.LabelerService.BatchSetTask6Status(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, service.ResponseMap[req.Status])
	}
}

func (api *LabelerAPI) SearchMyTask6() GinHandler {
	return func(c *gin.Context) {
		var req service.SearchMyTask6Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		if req.TaskType != "审核" && req.TaskType != "标注" {
			response.Error(c, 500, nil, "任务类型错误")
			return
		}
		p := actions.GetPermissionFromContext(c)
		req.UserID = strconv.Itoa(p.UserId)
		resp, total, err := api.LabelerService.SearchMyTask6(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, req.PageIndex, req.PageSize, "查询成功")
	}
}

func (api *LabelerAPI) DeleteTask6() GinHandler {
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

		if err := api.LabelerService.DeleteTask6(c.Request.Context(), objectID); err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, nil, "删除成功")
	}
}

func (api *LabelerAPI) DownloadTask6() GinHandler {
	return func(c *gin.Context) {
		var req service.DownloadTask6Req
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
		resp, err := api.LabelerService.DownloadTask6(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "下载成功")
	}
}

func (api *LabelerAPI) GetTask6() GinHandler {
	return func(c *gin.Context) {
		var req service.GetTask6Req
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
		if len(req.Status) == 0 {
			req.Status = []string{"未分配", "待标注", "已提交", "待审核", "已审核", "审核不通过"}
		}
		p := actions.GetPermissionFromContext(c)
		resp, err := api.LabelerService.GetTask6(c.Request.Context(), req, p)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "获取成功")
	}
}

func (api *LabelerAPI) SearchMyTask6Count() GinHandler {
	return func(c *gin.Context) {
		var req service.SearchMyTask6CountReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		if req.TaskType != "审核" && req.TaskType != "标注" {
			response.Error(c, 500, nil, "任务类型错误")
			return
		}
		p := actions.GetPermissionFromContext(c)
		req.UserID = strconv.Itoa(p.UserId)
		resp, err := api.LabelerService.SearchMyTask6Count(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "查询成功")
	}
}
