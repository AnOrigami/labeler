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
	routerCheckRole = append(routerCheckRole, project5AuthRouter())
}

func project5AuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/p5/", api.CreateProject5())
		g.PUT("/api/v1/labeler/p5/", api.UpdateProject5())
		g.DELETE("/api/v1/labeler/p5/", api.DeleteProject5())
		g.POST("/api/v1/labeler/p5/search", api.SearchProject5())
		g.GET("/api/v1/labeler/p5/count", api.Project5Count())
	}
}

func (api *LabelerAPI) CreateProject5() GinHandler {
	return func(c *gin.Context) {
		var project model.Project5
		if err := c.ShouldBindJSON(&project); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		if project.FolderID.IsZero() {
			response.Error(c, 500, nil, "文件夹ID不能为空")
			return
		}
		resp, err := api.LabelerService.CreateProject5(c.Request.Context(), project)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "创建成功")
	}
}

func (api *LabelerAPI) UpdateProject5() GinHandler {
	return func(c *gin.Context) {
		var req model.Project5
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		resp, err := api.LabelerService.UpdateProject5(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "修改成功")
	}
}

func (api *LabelerAPI) DeleteProject5() GinHandler {
	return func(c *gin.Context) {
		oid, err := QueryObjectID(c)
		if err != nil {
			response.Error(c, 400, err, "")
			return
		}
		resp, err := api.LabelerService.DeleteProject5(c.Request.Context(), service.DeleteProject5Req{ID: oid})
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "删除成功")
	}
}

func (api *LabelerAPI) SearchProject5() GinHandler {
	return func(c *gin.Context) {
		var req service.SearchProject5Req
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		resp, total, err := api.LabelerService.SearchProject5(c.Request.Context(), req)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, 1, 10000, "")
	}
}

func (api *LabelerAPI) Project5Count() GinHandler {
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
		resp, err := api.LabelerService.Project5Count(c.Request.Context(), service.Project5CountReq{ID: oid})
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "")
	}
}
