package api

import (
	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"

	"go-admin/app/labeler/model"
	"go-admin/app/labeler/service"
	"go-admin/common/log"
)

func init() {
	routerCheckRole = append(routerCheckRole, project3AuthRouter())
}

func project3AuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/p3/", api.CreateProject3())
		g.PUT("/api/v1/labeler/p3/", api.UpdateProject3())
		g.DELETE("/api/v1/labeler/p3/", api.DeleteProject3())
		g.POST("/api/v1/labeler/p3/search", api.SearchProject3())
		g.GET("/api/v1/labeler/p3/count", api.Project3Count())
	}
}

func (api *LabelerAPI) CreateProject3() GinHandler {
	return func(c *gin.Context) {
		var project model.Project3
		if err := c.ShouldBindJSON(&project); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		if project.FolderID.IsZero() {
			response.Error(c, 500, nil, "文件夹ID不能为空")
			return
		}
		resp, err := api.LabelerService.CreateProject3(c.Request.Context(), project)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "创建成功")
	}
}

func (api *LabelerAPI) UpdateProject3() GinHandler {
	return func(c *gin.Context) {
		var req model.Project3
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		resp, err := api.LabelerService.UpdateProject3(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "修改成功")
	}
}

func (api *LabelerAPI) DeleteProject3() GinHandler {
	return func(c *gin.Context) {
		oid, err := QueryObjectID(c)
		if err != nil {
			response.Error(c, 400, err, "")
			return
		}
		resp, err := api.LabelerService.DeleteProject3(c.Request.Context(), service.DeleteProject3Req{ID: oid})
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "删除成功")
	}
}

func (api *LabelerAPI) SearchProject3() GinHandler {
	return func(c *gin.Context) {
		var req service.SearchProject3Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		resp, total, err := api.LabelerService.SearchProject3(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, 1, 10000, "")
	}
}

func (api *LabelerAPI) Project3Count() GinHandler {
	return func(c *gin.Context) {
		oid, err := QueryObjectID(c)
		if err != nil {
			response.Error(c, 400, err, "参数异常")
			return
		}
		if oid.IsZero() {
			response.Error(c, 400, nil, "项目id不能为空")
			return
		}
		resp, err := api.LabelerService.Project3Count(c.Request.Context(), service.Project3CountReq{ID: oid})
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "")
	}
}
