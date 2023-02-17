package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go-admin/app/scrm"
	"go-admin/app/scrm/service"
	"net/http"
)

func SearchCallHistory(c *gin.Context) {
	var req service.SearchCallHistoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	resp, total, err := service.SearchCallHistory(c.Request.Context(), req)
	if err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "")
		return
	}
	response.PageOK(c, resp, int(total), req.GetPageIndex(), req.GetPageSize(), "查询成功")
}

func GetCallDetail(c *gin.Context) {
	req := service.GetCallDetailReq{}
	if err := c.ShouldBindQuery(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	if len(req.ID) == 0 {
		response.Error(c, 200, nil, "id为空")
		return
	}
	call, err := service.GetCallDetail(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, call, "获取成功")
}

func UpdateCall(c *gin.Context) {
	var req service.UpdateCallReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, http.StatusInternalServerError, nil, "参数异常")
		return
	}
	resp, err := service.UpdateCall(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	response.OK(c, resp, "修改成功")
}

func ModelUpdateCallLabel(c *gin.Context) {
	var req service.ModelUpdateCallLabelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	resp, err := service.ModelUpdateCallLabel(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	response.OK(c, resp, "修改成功")
}

func ModelUpdateCallSwitchTime(c *gin.Context) {
	var req service.ModelUpdateCallSwitchTimeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	resp, err := service.ModelUpdateCallSwitchTime(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	response.OK(c, resp, "修改成功")
}

func ExportCallHistory(c *gin.Context) {
	var req service.SearchCallHistoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	resp, err := service.ExportCallHistory(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, resp, "导出成功")
}

func AsyncExportCallHistory(c *gin.Context) {
	var req service.SearchCallHistoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	err := service.AsyncExportCallHistory(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, service.AsyncExportCallHistoryResp{}, "添加导出任务成功")
}

func ReportModelCallHistory(c *gin.Context) {
	var req service.ReportModelCallHistoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
		return
	}
	if len(req.CallID) == 0 || req.ProjectID == 0 {
		scrm.Logger().WithContext(c.Request.Context()).Error("request params missing")
		response.Error(c, 500, nil, "部分参数为空")
		return
	}
	err := service.ReportModelCallHistory(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
		return
	}
	response.OK(c, service.ReportModelCallHistoryResp{}, "推送成功")
}
