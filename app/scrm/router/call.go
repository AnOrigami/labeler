package router

import (
	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"go-admin/app/scrm/api"
)

func init() {
	routerCheckRole = append(routerCheckRole, registerCallRouter)
	routerNoAuth = append(routerNoAuth, registerReportCallRouter)
}

func registerCallRouter(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	r := v1.Group("")
	{
		r.POST("/api/v1/scrm/ch/search", api.SearchCallHistory)
		r.GET("/api/v1/scrm/ch/detail", api.GetCallDetail)
		r.PUT("/api/v1/scrm/ch/", api.UpdateCall)
		r.POST("/api/v1/scrm/ch/export", api.ExportCallHistory)
		r.POST("/api/v1/scrm/ch/et", api.AsyncExportCallHistory)
	}
}

func registerReportCallRouter(v1 *gin.RouterGroup) {
	v1.POST("/api/v1/scrm/ch/m/report", api.ReportModelCallHistory)
	v1.POST("/api/v1/scrm/m/ch/report", api.ReportModelCallHistory)
	v1.PUT("/api/v1/scrm/m/ch/label", api.ModelUpdateCallLabel)
	v1.PUT("/api/v1/scrm/m/ch/switch", api.ModelUpdateCallSwitchTime)
}
