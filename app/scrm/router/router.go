package router

import (
	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"go-admin/common/actions"
)

var (
	routerNoAuth    = make([]func(g *gin.RouterGroup), 0)
	routerCheckRole = make([]func(v1 *gin.RouterGroup, authMiddleware *jwtauth.GinJWTMiddleware), 0)
)

func InitSCRMRouter(r *gin.Engine, authMiddleware *jwtauth.GinJWTMiddleware) *gin.Engine {
	noAuth := r.Group("")
	for _, f := range routerNoAuth {
		f(noAuth)
	}
	group := r.Group("")
	group.Use(authMiddleware.MiddlewareFunc(), actions.PermissionAction())
	for _, f := range routerCheckRole {
		f(group, authMiddleware)
	}
	return r
}
