package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go-admin/app/scrm"
	"go-admin/app/scrm/service"
	"net/http"
)

func ExportNotConsumedPhone(c *gin.Context) {
	var req service.ExportNotConsumedPhoneReq
	form, err := c.MultipartForm()
	if err != nil {
		scrm.Logger().Error(err.Error())
		response.Error(c, 500, err, "文件解析异常")
		return
	}
	files := form.File["file"]
	req.Files = append(req.Files, files...)
	if len(files) < 2 {
		response.Error(c, 500, err, "缺少文件参数")
		return
	}

	resp, err := service.ExportNotConsumedPhone(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, nil, "")
		return
	}
	response.OK(c, resp, "导出成功")
}
