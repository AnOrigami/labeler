package api

import (
	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go-admin/app/labeler/model"
	"go-admin/common/log"
)

func init() {
	routerCheckRole = append(routerCheckRole, projectAuthRouter())
}

func projectAuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/p/", api.CreateProject())
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
