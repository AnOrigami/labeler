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
	routerCheckRole = append(routerCheckRole, taskAuthRouter())
}

func taskAuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/t/upload", api.UploadTask())
	}
}

func (api *LabelerAPI) UploadTask() GinHandler {
	return func(c *gin.Context) {
		mf, err := c.MultipartForm()
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		files := mf.File["files"]
		projectID, err := primitive.ObjectIDFromHex(c.Request.FormValue("projectId"))
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		tasks := make([]model.Task, len(files))
		for i, fh := range files {
			document, err := ReadFileHeader(fh)
			if err != nil {
				log.Logger().WithContext(c.Request.Context()).Error(err.Error())
				response.Error(c, 500, err, "")
				return
			}
			tasks[i] = model.Task{
				ID:        primitive.NewObjectID(),
				ProjectID: projectID,
				Name:      fh.Filename,
				Status:    model.TaskStatusLabeling,
				Document:  document,
				Contents:  []model.Content{},
			}
		}
		resp, err := api.LabelerService.UploadTask(c.Request.Context(), tasks)
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}
		response.OK(c, resp, "")
	}
}
