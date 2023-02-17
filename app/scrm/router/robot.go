package router

import (
	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"go-admin/app/scrm/api"
)

func init() {
	routerCheckRole = append(routerCheckRole, registerRobotRouter)
}

func registerRobotRouter(v1 *gin.RouterGroup, authMiddleware *jwt.GinJWTMiddleware) {
	//r := v1.Group("/p").Use(authMiddleware.MiddlewareFunc()).Use(middleware.AuthCheckRole()).Use(actions.PermissionAction())
	r := v1.Group("")
	{
		r.POST("/api/v1/scrm/r/", api.CreateRobot)
		r.DELETE("/api/v1/scrm/r/", api.DeleteRobot)
		r.PUT("/api/v1/scrm/r/", api.UpdateRobot)
		r.POST("/api/v1/scrm/r/search", api.SearchRobots)
		r.GET("/api/v1/scrm/r/detail", api.GetRobot)
	}
}
