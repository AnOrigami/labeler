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
	routerCheckRole = append(routerCheckRole, project2AuthRouter())
}

func project2AuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/p2/", api.CreateProject2())
		g.PUT("/api/v1/labeler/p2/", api.UpdateProject2())
		g.DELETE("/api/v1/labeler/p2/", api.DeleteProject2())
		g.POST("/api/v1/labeler/p2/search", api.SearchProject2())
		g.GET("/api/v1/labeler/p2/count", api.Project2Count())
	}
}

func (api *LabelerAPI) CreateProject2() GinHandler {
	return func(c *gin.Context) {
		var project model.Project2
		if err := c.ShouldBindJSON(&project); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}

		resp, err := api.LabelerService.CreateProject2(c.Request.Context(), project)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "创建成功")
	}
}

func (api *LabelerAPI) UpdateProject2() GinHandler {
	return func(c *gin.Context) {
		var req model.Project2
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		resp, err := api.LabelerService.UpdateProject2(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "修改成功")
	}
}

func (api *LabelerAPI) DeleteProject2() GinHandler {
	return func(c *gin.Context) {
		oid, err := QueryObjectID(c)
		if err != nil {
			response.Error(c, 400, err, "")
			return
		}
		resp, err := api.LabelerService.DeleteProject2(c.Request.Context(), service.DeleteProject2Req{ID: oid})
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "删除成功")
	}
}

func (api *LabelerAPI) SearchProject2() GinHandler {
	return func(c *gin.Context) {
		var req service.SearchProject2Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		resp, total, err := api.LabelerService.SearchProject2(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, 1, 10000, "")
	}
}

func (api *LabelerAPI) Project2Count() GinHandler {
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
		resp, err := api.LabelerService.Project2Count(c.Request.Context(), service.Project2CountReq{ID: oid})
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "")
	}
}
