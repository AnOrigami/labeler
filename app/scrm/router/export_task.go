package router

import (
	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"go-admin/app/scrm/api"
)

func init() {
	routerCheckRole = append(routerCheckRole, registerExportRouter)
}

func registerExportRouter(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	r := v1.Group("")
	{
		r.POST("/api/v1/scrm/et/search", api.SearchExportTask)
		r.GET("/api/v1/scrm/et/file", api.ExportTaskFile)
	}
}
