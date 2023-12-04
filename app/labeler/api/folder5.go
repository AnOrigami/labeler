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
	routerCheckRole = append(routerCheckRole, folder5AuthRouter())
}

func folder5AuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/f5/", api.CreateFolder5())
		g.PUT("/api/v1/labeler/f5/", api.UpdateFolder5())
		g.DELETE("/api/v1/labeler/f5/", api.DeleteFolder5())
		g.GET("/api/v1/labeler/f5/", api.GetFolders5())
	}
}

func (api *LabelerAPI) CreateFolder5() GinHandler {
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

		resp, err := api.LabelerService.CreateFolder5(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "创建成功")
	}
}

func (api *LabelerAPI) UpdateFolder5() GinHandler {
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

		resp, err := api.LabelerService.UpdateFolder5(c.Request.Context(), req)
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "更新成功")
	}
}

func (api *LabelerAPI) DeleteFolder5() GinHandler {
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

		if err := api.LabelerService.DeleteFolder5(c.Request.Context(), objectID); err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, nil, "删除成功")
	}
}

func (api *LabelerAPI) GetFolders5() GinHandler {
	return func(c *gin.Context) {
		res, err := api.LabelerService.GetFolders5(c.Request.Context())
		if err != nil {
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, res, "查询成功")
	}
}