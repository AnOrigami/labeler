package api

import (
	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go-admin/app/labeler/model"
	"go-admin/common/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func init() {
	routerCheckRole = append(routerCheckRole, schemaAuthRouter())
}

func schemaAuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/schema/", api.CreateSchema())
		g.PUT("/api/v1/labeler/schema/", api.UpdateSchema())
		g.DELETE("/api/v1/labeler/schema/", api.DeleteSchema())
		g.GET("/api/v1/labeler/schema/", api.GetSchema())
	}
}

func (api *LabelerAPI) CreateSchema() GinHandler {
	return func(c *gin.Context) {
		var req model.Schema
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		if len(req.Name) == 0 {
			response.Error(c, 500, nil, "规则名不能为空")
			return
		}

		resp, err := api.LabelerService.CreateSchema(c, req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "创建成功")
	}
}

func (api *LabelerAPI) UpdateSchema() GinHandler {
	return func(c *gin.Context) {
		var req model.Schema
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		if req.ID.IsZero() {
			response.Error(c, 500, nil, "id不能为空")
			return
		}
		if len(req.Name) == 0 {
			response.Error(c, 500, nil, "规则名不能为空")
			return
		}

		resp, err := api.LabelerService.UpdateSchema(c, req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "更新成功")
	}
}

func (api *LabelerAPI) DeleteSchema() GinHandler {
	return func(c *gin.Context) {
		id := c.Query("id")
		if len(id) == 0 {
			response.Error(c, 500, nil, "id不能为空")
			return
		}
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
		}

		if err := api.LabelerService.DeleteSchema(c, objectID); err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, nil, "删除成功")
	}
}

func (api *LabelerAPI) GetSchema() GinHandler {
	return func(c *gin.Context) {
		res, err := api.LabelerService.GetSchema(c)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, res, "查询成功")
	}
}
