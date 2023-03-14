package api

import (
	"context"
	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth/user"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go-admin/app/admin/models"
	"go-admin/app/labeler/model"
	"go-admin/app/labeler/service"
	"go-admin/app/scrm"
	"go-admin/common/actions"
	"go-admin/common/log"
	"go-admin/common/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
	"time"
)

func init() {
	routerCheckRole = append(routerCheckRole, taskAuthRouter())
}

func taskAuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/t/upload", api.UploadTask())
		g.PUT("/api/v1/labeler/t/label", api.LabelTask())
		g.POST("/api/v1/labeler/t/search", api.SearchTask())
		g.GET("/api/v1/labeler/t/", api.GetTask())
		g.POST("/api/v1/labeler/t/allocate", api.AllocateTasks())
		g.POST("/api/v1/labeler/t/reset", api.ResetTasks())
		g.POST("/api/v1/labeler/parse", api.ModelParse())
		g.POST("/api/v1/labeler/t/checkallocate", api.AllocateCheckTasks())
	}
}

func (api *LabelerAPI) UploadTask() GinHandler {
	return func(c *gin.Context) {
		mf, err := c.MultipartForm()
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		files := mf.File["files"]
		projectID, err := primitive.ObjectIDFromHex(c.Request.FormValue("projectId"))
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		tasks := make([]model.Task, len(files))
		now := util.Datetime(time.Now())
		for i, fh := range files {
			document, err := ReadFileHeader(fh)
			if err != nil {
				log.Logger().WithContext(c.Request.Context()).Error(err.Error())
				response.Error(c, 500, err, "")
				return
			}
			tasks[i] = model.Task{
				ID:         primitive.NewObjectID(),
				ProjectID:  projectID,
				Name:       fh.Filename,
				Status:     model.TaskStatusAllocate,
				Document:   document,
				Contents:   []model.Content{},
				Activities: []model.Activity{},
				UpdateTime: now,
			}
		}
		resp, err := api.LabelerService.UploadTask(c.Request.Context(), tasks)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "")
	}
}

func (api *LabelerAPI) LabelTask() GinHandler {
	return func(c *gin.Context) {
		var req model.Task
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		if req.ID.IsZero() {
			response.Error(c, 500, nil, "id不能为空")
			return
		}
		userID := user.GetUserId(c)
		resp, err := api.LabelerService.LabelTask(c, req, userID)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "更新成功")
	}
}

func (api *LabelerAPI) SearchTask() GinHandler {
	return func(c *gin.Context) {
		var req service.SearchTaskReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		p := actions.GetPermissionFromContext(c)
		req.UserID = p.UserId
		req.DataScope = p.DataScope

		resp, total, err := api.LabelerService.SearchTask(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, req.PageIndex, req.PageSize, "查询成功")
	}
}

func (api *LabelerAPI) GetTask() GinHandler {
	return func(c *gin.Context) {
		oid, err := QueryObjectID(c)
		if err != nil {
			response.Error(c, 500, err, "参数异常")
			return
		}
		resp, err := api.LabelerService.GetTask(c.Request.Context(), oid)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "获取成功")
	}
}

func (api *LabelerAPI) AllocateTasks() GinHandler {
	return func(c *gin.Context) {
		var req service.AllocateTasksReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		if req.ProjectID.IsZero() {
			response.Error(c, 500, nil, "项目id不能为空")
			return
		}

		if err := api.LabelerService.AllocateTasks(c.Request.Context(), req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, nil, "分配成功")
	}
}

func (api *LabelerAPI) ModelParse() GinHandler {
	return func(c *gin.Context) {
		var req service.ModelParseReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		if req.ModelURL == "" {
			response.Error(c, 500, nil, "modelURL为空")
			return
		}

		resp, err := api.LabelerService.ModelParse(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "获取成功")
	}
}

func (api *LabelerAPI) ResetTasks() GinHandler {
	return func(c *gin.Context) {
		p := actions.GetPermissionFromContext(c)
		if p.DataScope != "1" && p.DataScope != "2" {
			response.Error(c, http.StatusUnauthorized, nil, "当前用户没有操作权限")
			return
		}
		var req service.ResetTasksReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		if req.ProjectID.IsZero() {
			response.Error(c, 500, nil, "project id不能为空")
			return
		}

		if err := api.LabelerService.ResetTasks(c.Request.Context(), req); err != nil {
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, nil, "重置成功")
	}
}

func (api *LabelerAPI) AllocateCheckTasks() GinHandler {
	return func(c *gin.Context) {
		roleID := user.GetRoleId(c)
		var role models.SysRole
		db := scrm.GormDB.WithContext(context.Background()).First(&role, roleID)
		if err := db.Error; err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "db error")
			return
		}
		if !role.Admin {
			response.Error(c, 500, nil, "当前用户无分配审核任务权限")
			return
		}
		var req service.AllocateCheckTasksReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		if err := api.LabelerService.AllocateCheckTasks(c.Request.Context(), req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, nil, "分配成功")
	}
}
