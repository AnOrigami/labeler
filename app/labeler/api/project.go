package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"go-admin/app/labeler/model"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func init() {
	routerCheckRole = append(routerCheckRole, projectAuthRouter())
}

func projectAuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/p/", api.CreateProject())
	}
}

func (api *LabelerAPI) CreateProject() GinHandler {
	return func(c *gin.Context) {
		var project model.Project
		fmt.Println(c.ShouldBindJSON(&project))
		fmt.Println(primitive.NewObjectID().Hex())
		api.LabelerService.CreateProject(c.Request.Context(), project)
	}
}
