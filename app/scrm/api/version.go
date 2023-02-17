package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	_ "go-admin/app/scrm/service"
	"go-admin/common/global"
)

func GetVersion(c *gin.Context) {
	response.OK(c, GetVersionResp{Version: global.Version}, "获取成功")
}

type GetVersionResp struct {
	Version string `json:"version"`
}
