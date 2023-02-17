package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"go-admin/app/scrm"
	"go-admin/app/scrm/service"
)

func BindSeatAndWeCom(c *gin.Context) {
	var req service.BindSeatAndWeComReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
	}
	err := service.BindSeatAndWeCom(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
	} else {
		response.OK(c, "", "执行成功")
	}
}

func AddFriend(c *gin.Context) {
	var req service.AddFriendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "参数异常")
	}
	err := service.AddFriend(c.Request.Context(), req)
	if err != nil {
		response.Error(c, 500, err, "")
	} else {
		response.OK(c, "", "添加成功")
	}
}

func SearchBindStatus(c *gin.Context) {
	WeComName, err := service.SearchBindStatus(c.Request.Context())
	if err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "查询绑定情况出错")
	} else {
		response.OK(c, WeComName, "查询成功")
	}
}
