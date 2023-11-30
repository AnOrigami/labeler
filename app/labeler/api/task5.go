package api

import (
	"encoding/json"
	"io/ioutil"
	"mime/multipart"

	"github.com/gin-gonic/gin"
	jwt "github.com/go-admin-team/go-admin-core/sdk/pkg/jwtauth"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"

	"go-admin/app/labeler/model"
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
		var task5map []map[string][]model.Task5

		var fs []string
		var filedatamap map[string][]model.Task5
		for _, fh := range files {

			filename := fh.Filename
			fs = append(fs, filename)
			file, _ := fh.Open()
			d, _ := ioutil.ReadAll(file)
			defer func(file multipart.File) {
				err := file.Close()
				if err != nil {

				}
			}(file)
			_ = json.Unmarshal(d, &filedatamap)

			task5map = append(task5map, filedatamap)

		}
		var task5 []model.Task5
		for _, v := range task5map {
			for _, v2 := range v {
				for _, v3 := range v2 {
					task5 = append(task5, v3)
				}
			}
		}

		response.OK(c, task5, "上传成功")
	}

}
