package router

import (
	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"go-admin/app/scrm/api"
)

func init() {
	routerNoAuth = append(routerNoAuth, registerSeatRouterNoAuth)
	routerCheckRole = append(routerCheckRole, registerSeatRouter)
}

func registerSeatRouterNoAuth(g *gin.RouterGroup) {
	r := g.Group("")
	{
		r.POST("/api/v1/scrm/s/m/lock", api.LockSeat)
		r.POST("/api/v1/scrm/s/m/unlock", api.UnlockSeat)
		r.GET("/api/v1/scrm/s/m/status", api.GetSeatListOfProject)
		r.POST("/api/v1/scrm/m/s/lock", api.LockSeat)
		r.POST("/api/v1/scrm/m/s/unlock", api.UnlockSeat)
		r.GET("/api/v1/scrm/m/s/status", api.GetSeatListOfProject)
		r.PUT("/api/v1/scrm/m/s/label", api.ModelUpdateCallLabel)
	}
}

func registerSeatRouter(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	r := v1.Group("")
	{
		r.POST("/api/v1/scrm/s/", api.CreateSeat)
		r.PUT("/api/v1/scrm/s/", api.SetSeat)
		r.DELETE("/api/v1/scrm/s/", api.DelSeat)
		r.POST("/api/v1/scrm/s/search", api.GetSeatList)
		r.GET("/api/v1/scrm/s/ws", api.SeatHandleWs)
		r.GET("/api/v1/scrm/s/project", api.SearchProjectsOfSeat)
		r.PUT("/api/v1/scrm/s/preready", api.SetSeatPreReady)
	}
}
