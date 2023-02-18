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
	routerCheckRole = append(routerCheckRole, projectAuthRouter())
}

func projectAuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/p/", api.CreateProject())
		g.PUT("/api/v1/labeler/p/", api.UpdateProject())
		g.DELETE("/api/v1/labeler/p/", api.DeleteProject())
		g.POST("/api/v1/labeler/p/search", api.SearchProject())
		g.GET("/api/v1/labeler/p/detail", api.ProjectDetail())
	}
}

func (api *LabelerAPI) CreateProject() GinHandler {
	return func(c *gin.Context) {
		var project model.Project
		if err := c.ShouldBindJSON(&project); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}

		resp, err := api.LabelerService.CreateProject(c.Request.Context(), project)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "")
	}
}

func (api *LabelerAPI) SearchProject() GinHandler {
	return func(c *gin.Context) {
		var req service.SearchProjectReq
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		resp, total, err := api.LabelerService.SearchProject(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, 1, 10000, "")
	}
}

func (api *LabelerAPI) ProjectDetail() GinHandler {
	return func(c *gin.Context) {
		oid, err := QueryObjectID(c)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		resp, err := api.LabelerService.ProjectDetail(c.Request.Context(), service.ProjectDetailReq{ID: oid})
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "")
	}
}

func (api *LabelerAPI) UpdateProject() GinHandler {
	return func(c *gin.Context) {
		var req model.Project
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		resp, err := api.LabelerService.UpdateProject(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "")
	}
}

func (api *LabelerAPI) DeleteProject() GinHandler {
	return func(c *gin.Context) {
		oid, err := QueryObjectID(c)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}
		resp, err := api.LabelerService.DeleteProject(c.Request.Context(), service.DeleteProjectReq{ID: oid})
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "")
	}
}
