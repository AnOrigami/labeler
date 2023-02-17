package router

import (
	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"go-admin/app/scrm/api"
)

func init() {
	routerCheckRole = append(routerCheckRole, registerProjectRouter)
}

func registerProjectRouter(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	r := v1.Group("")
	{
		r.POST("/api/v1/scrm/p/", api.CreateProject)
		r.POST("/api/v1/scrm/p/unlock", api.UnlockProjectSeat)
		r.GET("/api/v1/scrm/p/", api.GetProjectDetail)
		r.GET("/api/v1/scrm/p/s", api.GetSeatDetailOfProject)
		r.DELETE("/api/v1/scrm/p/", api.DeleteProject)
		r.POST("/api/v1/scrm/p/search", api.SearchProjects)
		r.PUT("/api/v1/scrm/p/running", api.RunProject)
		r.PUT("/api/v1/scrm/p/robot", api.SetProjectRobots)
		r.PUT("/api/v1/scrm/p/seat", api.SetProjectSeats)
		r.PUT("/api/v1/scrm/p/concurrency", api.SetProjectConcurrency)
	}
}
