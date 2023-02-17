package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go-admin/app/scrm/service"
	"net/http"
)

func GetLabels(c *gin.Context) {
	resp, err := service.GetLabels(c.Request.Context(), service.GetLabelsReq{})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, nil, "查询数据库失败")
		return
	}
	response.OK(c, resp, "成功")
}
