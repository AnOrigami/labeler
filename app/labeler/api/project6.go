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
	routerCheckRole = append(routerCheckRole, project6AuthRouter())
}

func project6AuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/p6/", api.CreateProject6())
		g.PUT("/api/v1/labeler/p6/", api.UpdateProject6())
		g.DELETE("/api/v1/labeler/p6/", api.DeleteProject6())
		g.POST("/api/v1/labeler/p6/search", api.SearchProject6())
		g.GET("/api/v1/labeler/p6/count", api.Project6Count())
	}
}

func (api *LabelerAPI) CreateProject6() GinHandler {
	return func(c *gin.Context) {
		var project model.Project6
		if err := c.ShouldBindJSON(&project); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		if project.FolderID.IsZero() {
			response.Error(c, 500, nil, "文件夹ID不能为空")
			return
		}
		resp, err := api.LabelerService.CreateProject6(c.Request.Context(), project)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "创建成功")
	}
}

func (api *LabelerAPI) UpdateProject6() GinHandler {
	return func(c *gin.Context) {
		var req model.Project6
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		resp, err := api.LabelerService.UpdateProject6(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "修改成功")
	}
}

func (api *LabelerAPI) DeleteProject6() GinHandler {
	return func(c *gin.Context) {
		oid, err := QueryObjectID(c)
		if err != nil {
			response.Error(c, 400, err, "")
			return
		}
		resp, err := api.LabelerService.DeleteProject6(c.Request.Context(), service.DeleteProject6Req{ID: oid})
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "删除成功")
	}
}

func (api *LabelerAPI) SearchProject6() GinHandler {
	return func(c *gin.Context) {
		var req service.SearchProject6Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		resp, total, err := api.LabelerService.SearchProject6(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, 1, 10000, "")
	}
}

func (api *LabelerAPI) Project6Count() GinHandler {
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
		resp, err := api.LabelerService.Project6Count(c.Request.Context(), service.Project6CountReq{ID: oid})
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "")
	}
}
