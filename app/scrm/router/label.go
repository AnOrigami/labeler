package router

import (
	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"go-admin/app/scrm/api"
)

func init() {
	routerCheckRole = append(routerCheckRole, registerLabelRouter)
}

func registerLabelRouter(g *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	r := g.Group("")
	{
		r.GET("/api/v1/scrm/label/", api.GetLabels)
	}
}
