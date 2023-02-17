package router

import (
	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"go-admin/app/scrm/api"
)

func init() {
	routerCheckRole = append(routerCheckRole, registerOrderRouter)
}

func registerOrderRouter(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	r := v1.Group("")
	{
		r.POST("/api/v1/scrm/os/upload", api.UploadOrderGroup)
		r.POST("/api/v1/scrm/os/search", api.SearchOrderGroup)
		r.POST("/api/v1/scrm/o/", api.GetOrderList)
		r.GET("/api/v1/scrm/o/detail", api.GetOrderDetail)
		r.DELETE("/api/v1/scrm/o/", api.DeleteOrder)
		r.POST("/api/v1/scrm/o/export", api.ExportOrders)
	}
}
