package api

import (
	"encoding/json"
	"io/ioutil"
	"mime/multipart"

	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-admin/app/labeler/model"
	"go-admin/app/labeler/service"
	"go-admin/common/log"
)

func init() {
	routerCheckRole = append(routerCheckRole, task5AuthRouter())
}

func task5AuthRouter() RouterCheckRole {
	return func(g *gin.RouterGroup, api *LabelerAPI, authMiddleware *jwt.GinJWTMiddleware) {
		g.POST("/api/v1/labeler/t5/upload", api.UploadTask5())
	}
}
func (api *LabelerAPI) UploadTask5() GinHandler {
	return func(c *gin.Context) {
		mf, err := c.MultipartForm()
		if err != nil {
			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 400, err, "")
			return
		}
		files := mf.File["files"]
		var tasks []model.Task5
		var filenames []string
		var projectID primitive.ObjectID
		for _, fh := range files {

			filename := fh.Filename
			filenames = append(filenames, filename)
			file, _ := fh.Open()
			d, err := ioutil.ReadAll(file)
			if err != nil {
				log.Logger().WithContext(c.Request.Context()).Error(err.Error())
				response.Error(c, 500, err, "")
				return
			}
			defer func(file multipart.File) {
				err := file.Close()
				if err != nil {
					log.Logger().WithContext(c.Request.Context()).Error(err.Error())
					response.Error(c, 500, err, "")
					return
				}
			}(file)
			var filedata model.Task5
			err = json.Unmarshal(d, &filedata)
			if err != nil {
				log.Logger().WithContext(c.Request.Context()).Error(err.Error())
				response.Error(c, 400, err, "")
				return
			}
			projectID, err = primitive.ObjectIDFromHex(c.Request.FormValue("projectId"))
			if err != nil {
				log.Logger().WithContext(c.Request.Context()).Error(err.Error())
				response.Error(c, 400, err, "")
				return
			}

			tasks = append(tasks, filedata)

		}
		req := service.UploadTask5Req{
			Task5:     tasks,
			ProjectID: projectID,
			Name:      filenames,
		}
		resp, err := api.LabelerService.UploadTask5(c.Request.Context(), req)
		if err != nil {

			log.Logger().WithContext(c.Request.Context()).Error(err.Error())
			response.Error(c, 500, err, "")
			return
		}

		response.OK(c, resp, "上传成功")
	}

}
