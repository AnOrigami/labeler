package api

import (
	"github.com/gin-gonic/gin"
	"github.com/go-admin-team/go-admin-core/sdk/pkg/response"
	"github.com/gorilla/websocket"
	"go-admin/app/scrm"
	"go-admin/app/scrm/service"
	"go-admin/common/actions"
	"go-admin/common/log"
	"net/http"
	"strconv"
	"time"
)

func CreateSeat(c *gin.Context) {
	var req service.CreateSeatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error("ShouldBindJSON ERROR: ", err.Error())
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	err := service.CreateSeat(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	response.OK(c, gin.H{}, "创建坐席成功")
}

func GetSeatList(c *gin.Context) {
	var req service.GetSeatListReq
	p := actions.GetPermissionFromContext(c)
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, http.StatusInternalServerError, err, "参数异常")
		return
	}
	resp, total, err := service.GetSeatList(c.Request.Context(), p, req)
	if err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, http.StatusInternalServerError, err, "查询失败")
		return
	}
	response.PageOK(c, resp, int(total), req.GetPageIndex(), req.GetPageSize(), "查询成功")
}

func DelSeat(c *gin.Context) {
	id := c.Query("id")
	if len(id) == 0 {
		scrm.Logger().WithContext(c.Request.Context()).Error("Query id ERROR")
		response.Error(c, http.StatusInternalServerError, nil, "Query ERROR")
		return
	}
	iid, err := strconv.Atoi(id)
	if err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error("Atoi ERROR: ", err.Error())
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	err = service.DelSeat(c.Request.Context(), iid)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	response.OK(c, gin.H{}, "删除坐席成功")
}

func SetSeat(c *gin.Context) {
	var req service.UpdateSeatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error("ShouldBindJSON ERROR: ", err.Error())
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	err := service.UpdateSeat(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	response.OK(c, gin.H{}, "编辑坐席成功")
}

func SeatHandleWs(c *gin.Context) {
	ctx := c.Request.Context()
	conn, err := (&websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}).Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error("seat ws upgrade: ", err.Error())
		return
	}
	connID := c.GetHeader("Sec-WebSocket-Key")
	p := actions.GetPermissionFromContext(ctx)
	if connID == "" || p == nil || p.UserId == 0 {
		scrm.Logger().WithContext(ctx).Error("Unauthorized")
		_ = conn.SetWriteDeadline(time.Now().Add(service.WriteWait))
		err := conn.WriteJSON(service.NewMessage("error", service.ErrorData{Error: "Unauthorized"}))
		if err != nil {
			scrm.Logger().WithContext(ctx).Error("WriteJSON ERROR: ", err.Error())
		}
		_ = conn.Close()
		return
	}
	userID := strconv.Itoa(p.UserId)
	client := service.DefaultSeatHub.MakeClient(conn, userID, connID)
	scrm.Logger().WithContext(ctx).Infof("seat:%s ws connected", userID)
	go client.ReceiveLoop(log.WithNoCancel(ctx))
	go client.SendLoop(log.WithNoCancel(ctx))
}

func LockSeat(c *gin.Context) {
	var req service.LockSeatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error("ShouldBindJSON ERROR: ", err.Error())
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	res, err := service.LockSeat(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	response.OK(c, res, "锁定坐席结果")
}

func GetSeatListOfProject(c *gin.Context) {
	projectID, err := strconv.Atoi(c.Query("projectId"))
	if err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error("Atoi Error: ", err.Error())
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	res, err := service.GetSeatListOfProject(c.Request.Context(), projectID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	response.OK(c, res, "查询项目中签入的坐席状态成功")
}

func UnlockSeat(c *gin.Context) {
	var req service.LockSeatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error("ShouldBindJSON ERROR: ", err.Error())
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	res, err := service.UnlockSeat(c.Request.Context(), req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err, "")
		return
	}
	response.OK(c, res, "锁定坐席结果")
}

func SearchProjectsOfSeat(c *gin.Context) {
	var req service.SearchProjectsOfSeatReq
	p := actions.GetPermissionFromContext(c)
	req.SeatID = p.UserId
	resp, err := service.SearchProjectsOfSeat(c.Request.Context(), req)
	if err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "查询失败")
		return
	}
	response.OK(c, resp, "查询成功")
}

func SetSeatPreReady(c *gin.Context) {
	var req service.SetSeatPreReadyReq
	p := actions.GetPermissionFromContext(c)
	req.SeatID = p.UserId
	if err := service.SetSeatPreReady(c.Request.Context(), req); err != nil {
		scrm.Logger().WithContext(c.Request.Context()).Error(err.Error())
		response.Error(c, 500, err, "设置失败")
		return
	}
	response.OK(c, gin.H{}, "设置成功")
}
