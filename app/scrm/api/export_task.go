package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go-admin/app/scrm"
	"go-admin/app/scrm/service"
)

func SearchExportTask(c *gin.Context) {
	var req service.SearchExportTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	resp, total, err := service.SearchExportTask(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.PageOK(c, resp, int(total), req.PageSize, req.PageIndex, "查询成功")
}

func ExportTaskFile(c *gin.Context) {
	var req service.ExportTaskFileReq
	if err := c.ShouldBindQuery(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	if req.ID == 0 {
		response.Error(c, 500, nil, "参数异常")
		return
	}
	resp, err := service.ExportTaskFile(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, resp, "导出成功")
}
