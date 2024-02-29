package api

import (
	"encoding/json"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-admin/app/labeler/model"
	"go-admin/app/labeler/service"
	"go-admin/common/actions"
	"go-admin/common/log"
)

func init() {
	routerCheckRole = append(routerCheckRole, task5AuthRouter())
}

func task5AuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/t5/upload", api.UploadTask5())
		g.POST("/api/v1/labeler/t5/search", api.SearchTask5())
		g.POST("/api/v1/labeler/t5/alloc", api.AllocOneTask5())
		g.POST("/api/v1/labeler/t5/reset", api.ResetTasks5())
		g.PUT("/api/v1/labeler/t5/", api.UpdateTask5())
		g.POST("/api/v1/labeler/t5/batch/status", api.BatchSetTask5Status())
		g.POST("/api/v1/labeler/t5/my", api.SearchMyTask5())
		g.DELETE("/api/v1/labeler/t5/", api.DeleteTask5())
		g.POST("/api/v1/labeler/t5/download", api.DownloadTask5())
		g.POST("/api/v1/labeler/t5/detail", api.GetTask5())
		g.POST("/api/v1/labeler/t5/mycount", api.SearchMyTask5Count())
		g.POST("/api/v1/labeler/t5/action", api.GetActionTags())
		g.POST("/api/v1/labeler/t5/downloadscore", api.DownloadScore())
		g.POST("/api/v1/labeler/t5/downloadworkload", api.DownloadWorkload())
		g.POST("/api/v1/labeler/t5/proportionalScoring", api.ProportionalScoring())
	}
}

func (api *LabelerAPI) UploadTask5() GinHandler {
	return func(c *gin.Context) {
		mf, err := c.MultipartForm()
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		files := mf.File["files"]
		var tasks []model.Task5
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
			var filedata model.Task5
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
		req := service.UploadTask5Req{
			Tasks5:    tasks,
			ProjectID: projectID,
			Name:      filenames,
		}
		resp, err := api.LabelerService.UploadTask5(c.Request.Context(), req)
		if err != nil {

			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			if strings.Contains(err.Error(), "重复") {
				response.Error(c, 409, err, "")
				return
			}
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "上传成功")
	}

}

func (api *LabelerAPI) SearchTask5() GinHandler {
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

		resp, total, err := api.LabelerService.SearchTask5(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, req.PageIndex, req.PageSize, "查询成功")
	}
}

func (api *LabelerAPI) AllocOneTask5() GinHandler {
	return func(c *gin.Context) {
		var req service.AllocOneTaskReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "参数异常")
			return
		}
		if req.ProjectID.IsZero() {
			response.Error(c, 400, nil, "项目id不能为空")
			return
		}

		p := actions.GetPermissionFromContext(c)
		req.UserId = strconv.Itoa(p.UserId)

		resp, err := api.LabelerService.AllocOneTask5(c.Request.Context(), req)
		if err != nil {
			if err.Error() == "存在未标注任务" {
				response.OK(c, resp, "分配成功: 分配待标注任务")
			} else {
				log.Logger().WithContext(c.Request.Context()).Error(err.Error())
				response.Error(c, 500, err, "")
			}
			return
		}
		response.OK(c, resp, "分配成功")
	}
}

func (api *LabelerAPI) ResetTasks5() GinHandler {
	return func(c *gin.Context) {
		p := actions.GetPermissionFromContext(c)
		if p.DataScope != "1" && p.DataScope != "2" {
			response.Error(c, http.StatusUnauthorized, nil, "当前用户没有操作权限")
			return
		}
		var req service.ResetTasks5Req
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
		resp, err := api.LabelerService.ResetTasks5(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "重置成功")
	}
}

func (api *LabelerAPI) UpdateTask5() GinHandler {
	return func(c *gin.Context) {
		var req service.UpdateTask5Req
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
		//req.UserID = "2"
		req.UserDataScope = p.DataScope
		resp, err := api.LabelerService.UpdateTask5(c, req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "更新成功")
	}
}

func (api *LabelerAPI) BatchSetTask5Status() GinHandler {
	return func(c *gin.Context) {
		var req service.BatchSetTask5StatusReq
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
		resp, err := api.LabelerService.BatchSetTask5Status(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, service.ResponseMap[req.Status])
	}
}

func (api *LabelerAPI) SearchMyTask5() GinHandler {
	return func(c *gin.Context) {
		var req service.SearchMyTask5Req
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
		resp, total, err := api.LabelerService.SearchMyTask5(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, req.PageIndex, req.PageSize, "查询成功")
	}
}

func (api *LabelerAPI) DeleteTask5() GinHandler {
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

		if err := api.LabelerService.DeleteTask5(c.Request.Context(), objectID); err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, nil, "删除成功")
	}
}

func (api *LabelerAPI) DownloadTask5() GinHandler {
	return func(c *gin.Context) {
		var req service.DownloadTask5Req
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
		resp, err := api.LabelerService.DownloadTask5(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "下载成功")
	}
}

func (api *LabelerAPI) GetTask5() GinHandler {
	return func(c *gin.Context) {
		var req service.GetTask5Req
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
		resp, err := api.LabelerService.GetTask5(c.Request.Context(), req, p)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "获取成功")
	}
}

func (api *LabelerAPI) SearchMyTask5Count() GinHandler {
	return func(c *gin.Context) {
		var req service.SearchMyTask5CountReq
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
		resp, err := api.LabelerService.SearchMyTask5Count(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "查询成功")
	}
}

func (api *LabelerAPI) GetActionTags() GinHandler {
	return func(c *gin.Context) {
		response.OK(c, service.ActionTags, "查询成功")
	}
}

func (api *LabelerAPI) DownloadScore() GinHandler {
	return func(c *gin.Context) {
		var req service.DownloadScoreReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		resp, err := api.LabelerService.DownloadScore(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "查询成功")
	}
}

func (api *LabelerAPI) DownloadWorkload() GinHandler {
	return func(c *gin.Context) {
		var req service.DownloadWorkloadReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		resp, err := api.LabelerService.DownloadWorkload(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "查询成功")
	}
}

func (api *LabelerAPI) ProportionalScoring() GinHandler {
	return func(c *gin.Context) {
		var req service.ProportionalScoringReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		resp, err := api.LabelerService.ProportionalScoring(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "成功")
	}
}
