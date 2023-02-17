package router

import (
	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"go-admin/app/scrm/api"
)

func init() {
	routerCheckRole = append(routerCheckRole, registerWeComRouter)
}

func registerWeComRouter(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	r := v1.Group("")
	{
		r.POST("/api/v1/scrm/wc/bind", api.BindSeatAndWeCom)
		r.POST("/api/v1/scrm/wc/add", api.AddFriend)
		r.GET("/api/v1/scrm/wc/search", api.SearchBindStatus)
	}
}
