package api

import (
	"time"

	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-admin/app/labeler/model"
	"go-admin/common/log"
	"go-admin/common/util"
)

func init() {
	routerCheckRole = append(routerCheckRole, folder3AuthRouter())
}

func folder3AuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/f3/", api.CreateFolder3())
		g.PUT("/api/v1/labeler/f3/", api.UpdateFolder3())
		g.DELETE("/api/v1/labeler/f3/", api.DeleteFolder3())
		g.GET("/api/v1/labeler/f3/", api.GetFolders3())
	}
}

func (api *LabelerAPI) CreateFolder3() GinHandler {
	return func(c *gin.Context) {
		var req model.Folder
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "参数异常")
			return
		}
		if len(req.Name) == 0 {
			response.Error(c, 500, nil, "文件夹名不能为空")
			return
		}
		req.CreateTime = util.Datetime(time.Now())

		resp, err := api.LabelerService.CreateFolder3(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "创建成功")
	}
}

func (api *LabelerAPI) UpdateFolder3() GinHandler {
	return func(c *gin.Context) {
		var req model.Folder
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
			response.Error(c, 500, nil, "文件夹名不能为空")
			return
		}

		resp, err := api.LabelerService.UpdateFolder3(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "更新成功")
	}
}

func (api *LabelerAPI) DeleteFolder3() GinHandler {
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

		if err := api.LabelerService.DeleteFolder3(c.Request.Context(), objectID); err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, nil, "删除成功")
	}
}

func (api *LabelerAPI) GetFolders3() GinHandler {
	return func(c *gin.Context) {
		res, err := api.LabelerService.GetFolders3(c.Request.Context())
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, res, "查询成功")
	}
}
