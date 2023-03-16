package api

import (
	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go-admin/common/log"
)

func init() {
	routerCheckRole = append(routerCheckRole, userAuthRouter())
}

func userAuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.GET("/api/v1/labeler/user", api.GetUserList())
	}
}

func (api *LabelerAPI) GetUserList() GinHandler {
	return func(c *gin.Context) {
		resp, total, err := api.LabelerService.GetUserList(c.Request.Context())
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.PageOK(c, resp, total, 1, 10000, "获取成功")
	}
}
