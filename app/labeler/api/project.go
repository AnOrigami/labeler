package api

import (
	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
)

func init() {
	routerCheckRole = append(routerCheckRole, projectAuthRouter())
}

func projectAuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/p/", api.CreateProject())
	}
}

func (api *LabelerAPI) CreateProject() GinHandler {
	return func(c *gin.Context) {

	}
}
