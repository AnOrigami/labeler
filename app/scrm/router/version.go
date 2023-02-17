package router

import (
	"github.com/gin-gonic/gin"
	"go-admin/app/scrm/api"
)

func init() {
	routerNoAuth = append(routerNoAuth, registerVersionRouterNoAuth)
}

func registerVersionRouterNoAuth(g *gin.RouterGroup) {
	r := g.Group("")
	{
		r.GET("/api/v1/scrm/version", api.GetVersion)
	}
}
