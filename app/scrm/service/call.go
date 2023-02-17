package service

import (
	"context"
	"encoding/json"
	"errors"
	"go-admin/app/scrm"
	"go-admin/app/scrm/model"
	"go-admin/common/actions"
	"go-admin/common/gormscope"
	"go-admin/common/log"
	"go-admin/common/util"
	"go-admin/config"
	"gorm.io/gorm"
	"strings"
	"time"
)

var RoleMap = map[int]string{
	1: "人工坐席",
	2: "客户",
	3: "AI坐席",
}

type SearchCallHistoryReq struct {
	Start         string `json:"start"`
	End           string `json:"end"`
	ProjectID     int    `json:"projectId"`
	OrderID       int    `json:"orderId"`
	ModelLabelID  []int  `json:"modelLabelId"`
	CallLabelID   []int  `json:"callLabelId"`
	SeatLabelID   []int  `json:"seatLabelId"`
	HangupLabelID []int  `json:"hangupLabelId"`
	CallID        string `json:"callId"`
	Phone         string `json:"phone"`
	SeatID        int    `json:"seatId"`

	Pagination
}

type CallHistoryItem struct {
	ID        string    `json:"id"`
	Phone     string    `json:"phone"`
	OrderID   int       `json:"orderId"`
	Project   string    `json:"project"`
	CreatedAt time.Time `json:"createdAt"`
}

func SearchCallHistory(ctx context.Context, req SearchCallHistoryReq) ([]CallHistoryItem, int64, error) {
	var (
		calls []*model.Call
		count int64
		m     model.Call
	)
	db := scrm.GormDB.WithContext(ctx).
		Joins("Label").
		Joins("Order").
		Scopes(
			actions.DeptPermission(ctx, "Order"),
			gormscope.Paginate(&req.Pagination),
			gormscope.CreateDateTimeRange(req.Start, req.End, m.TableName()),
			SearchCallScope(req),
		).
		Preload("Order.Project").
		Order("created_at desc")
	db = db.Find(&calls)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("search call history error", err.Error())
		return nil, 0, err
	}
	log.LogAttr(ctx, log.Key("call.search.history.count").Int64(db.RowsAffected))
	db = db.Limit(-1).Offset(-1).Count(&count)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("count call error", err.Error())
		return nil, 0, err
	}
	items := make([]CallHistoryItem, len(calls))
	oidForLog := make([]int, len(calls))
	for i, v := range calls {
		var phone, project string
		if v.Order != nil {
			phone = util.HidePhone(v.Order.Phone)
			if v.Order.Project != nil {
				project = v.Order.Project.Name
			}
		}
		oidForLog[i] = v.OrderID
		items[i] = CallHistoryItem{
			ID:        v.ID,
			OrderID:   v.OrderID,
			Phone:     phone,
			Project:   project,
			CreatedAt: v.CreatedAt,
		}
	}
	log.LogAttr(ctx, log.Key("call.search.history.order.id").Int64(db.RowsAffected))
	return items, count, nil
}

type GetCallDetailReq struct {
	ID string `form:"id"`
}

type CallDetail struct {
	ID              string         `json:"id"`
	OrderID         int            `json:"orderId"`
	Phone           string         `json:"phone"`
	Sentences       []SentenceItem `json:"sentences"`
	Project         string         `json:"project"`
	AudioFile       string         `json:"audioFile"`
	ModelLabelName  string         `json:"modelLabelName"`
	SeatLabelName   string         `json:"seatLabelName"`
	CallLabelName   string         `json:"callLabelName"`
	HangupLabelName string         `json:"hangupLabelName"`
	Comment         string         `json:"comment"`
	Line            string         `json:"line"`
	SeatName        string         `json:"seatName"`
	SeatUserName    string         `json:"seatUserName"`
}

type SentenceItem struct {
	ID    int    `json:"id"`
	Role  int    `json:"role"`
	Index int    `json:"index"`
	Text  string `json:"text"`
}

type NameOfSeat struct {
	SeatID       int    `json:"seatID" gorm:"column:id"`
	SeatName     string `json:"seatName" gorm:"column:nickname"`
	SeatUserName string `json:"seatUserName" gorm:"column:username"`
}

func GetCallDetail(ctx context.Context, req GetCallDetailReq) (CallDetail, error) {
	var call model.Call
	err := scrm.GormDB.WithContext(ctx).
		Joins("Label").
		Joins("CallLabel").
		Joins("SeatLabel").
		Joins("HangupLabel").
		Joins("left join scrm_order on scrm_order.id=order_id").
		Scopes(
			actions.DeptPermission(ctx, "scrm_order"),
		).
		Preload("Order.Project").
		Preload("Sentences", func(db *gorm.DB) *gorm.DB {
			return db.Order("`index`")
		}).
		First(&call, "scrm_call.id = ?", req.ID).Error
	log.LogAttr(ctx, log.Key("call.get.call.id").String(req.ID))
	if err != nil {
		scrm.Logger().WithContext(ctx).Error("get call error", err.Error())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("查看对象不存在或无权查看")
		}
		return CallDetail{}, err
	}
	var projectName, phone string
	if call.Order != nil {
		phone = util.HidePhone(call.Order.Phone)
		if call.Order.Project != nil {
			projectName = call.Order.Project.Name
		}
	}
	var n NameOfSeat
	err = scrm.GormDB.WithContext(ctx).Table("scrm_call").
		Joins("left join scrm_seat on scrm_seat.id=scrm_call.seat_id").
		Joins("left join sys_user on sys_user.user_id=scrm_seat.user_id").
		Select("scrm_seat.nickname,sys_user.username").
		Where("scrm_call.id = ?", req.ID).
		Scan(&n).Error
	if err != nil {
		scrm.Logger().WithContext(ctx).Error("get call error", err.Error())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("查看对象不存在或无权查看")
			log.LogAttr(ctx, log.Key("call.detail.blank").Bool(true))
		}
		return CallDetail{}, err
	}
	return CallDetail{
		ID:        call.ID,
		OrderID:   call.OrderID,
		Phone:     phone,
		Project:   projectName,
		AudioFile: config.ExtConfig.AudioPrefix + call.AudioFile,
		ModelLabelName: func() string {
			if call.Label != nil {
				return call.Label.Name
			}
			return ""
		}(),
		CallLabelName: func() string {
			if call.CallLabel != nil {
				return call.CallLabel.Name
			}
			return ""
		}(),
		SeatLabelName: func() string {
			if call.SeatLabel != nil {
				return call.SeatLabel.Name
			}
			return ""
		}(),
		HangupLabelName: func() string {
			if call.HangupLabel != nil {
				return call.HangupLabel.Name
			}
			return ""
		}(),
		Comment: call.Comment,
		Sentences: util.Convert(
			call.Sentences,
			func(s model.Sentence) SentenceItem {
				return SentenceItem{
					ID:    s.ID,
					Role:  s.Role,
					Index: s.Index,
					Text:  s.Text,
				}
			}),
		Line:         call.Line.String,
		SeatName:     n.SeatName,
		SeatUserName: n.SeatUserName,
	}, nil
}

type UpdateCallReq = CallDetail

type UpdateCallResp struct {
}

func UpdateCall(ctx context.Context, req UpdateCallReq) (UpdateCallResp, error) {
	updates := map[string]interface{}{
		"comment": req.Comment,
	}
	if req.SeatLabelName != "" {
		var label model.Label
		db := scrm.GormDB.WithContext(ctx).
			Where("name=?", req.SeatLabelName).
			Find(&label)
		if err := db.Error; err != nil {
			scrm.Logger().WithContext(ctx).Error(err.Error())
			return UpdateCallResp{}, err
		} else if db.RowsAffected != 1 {
			return UpdateCallResp{}, errors.New("标签不存在")
		}
		updates["seat_label_id"] = label.ID
	} else {
		updates["seat_label_id"] = nil
	}
	{
		log.LogAttr(ctx, log.Key("call.update.id").String(req.ID))
		db := scrm.GormDB.WithContext(ctx).
			Model(&model.Call{}).
			Where("id=?", req.ID).
			Updates(updates)
		if err := db.Error; err != nil {
			scrm.Logger().WithContext(ctx).Error(err.Error())
			return UpdateCallResp{}, err
		}
	}
	return UpdateCallResp{}, nil
}

type ModelUpdateCallLabelReq = CallDetail

type ModelUpdateCallLabelResp struct {
}

func ModelUpdateCallLabel(ctx context.Context, req ModelUpdateCallLabelReq) (ModelUpdateCallLabelResp, error) {
	updates := map[string]interface{}{}
	if req.ModelLabelName != "" {
		var label model.Label
		db := scrm.GormDB.WithContext(ctx).
			Where("name=?", req.ModelLabelName).
			Find(&label)
		if err := db.Error; err != nil {
			scrm.Logger().WithContext(ctx).Error(err.Error())
			return ModelUpdateCallLabelResp{}, err
		} else if db.RowsAffected != 1 {
			return ModelUpdateCallLabelResp{}, errors.New("标签不存在")
		}
		switch label.Type {
		case LabelTypeModel:
			updates["label_id"] = label.ID
		case LabelTypeHangup:
			updates["hangup_label_id"] = label.ID
		}
	} else {
		updates["label_id"] = nil
	}
	{
		log.LogAttr(ctx, log.Key("call.model.update.label.id").String(req.ID))
		db := scrm.GormDB.WithContext(ctx).
			Model(&model.Call{}).
			Where("id=?", req.ID).
			Updates(updates)
		if err := db.Error; err != nil {
			scrm.Logger().WithContext(ctx).Error(err.Error())
			return ModelUpdateCallLabelResp{}, err
		}
	}
	return ModelUpdateCallLabelResp{}, nil
}

type ModelUpdateCallSwitchTimeReq struct {
	ID string `json:"id"`
}

type ModelUpdateCallSwitchTimeResp struct {
}

func ModelUpdateCallSwitchTime(ctx context.Context, req ModelUpdateCallSwitchTimeReq) (ModelUpdateCallLabelResp, error) {
	db := scrm.GormDB.WithContext(ctx).
		Model(&model.Call{}).
		Where("id=?", req.ID).
		Update("SwitchSeatTime", time.Now())
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error(err.Error())
		return ModelUpdateCallLabelResp{}, err
	}
	return ModelUpdateCallLabelResp{}, nil
}

type AsyncExportCallHistoryResp struct{}

func AsyncExportCallHistory(ctx context.Context, req SearchCallHistoryReq) error {
	deptID := actions.GetPermissionFromContext(ctx).DeptId
	args, err := json.Marshal(req)
	if err != nil {
		scrm.Logger().WithContext(ctx).Error("json marshal error when async export call history")
		return err
	}
	var task = model.ExportTask{
		Args:   string(args),
		DeptID: deptID,
		Status: ExportTaskStatusWaiting,
		Type:   ExportCallTask{}.GetTaskType(),
	}
	err = scrm.GormDB.WithContext(ctx).Create(&task).Error
	if err != nil {
		scrm.Logger().WithContext(ctx).Error("error when operating database async export call history")
	}
	return err
}

type ExportResp struct {
	File     *string `json:"file"`
	Filename string  `json:"filename"`
}

func ExportCallHistory(ctx context.Context, req SearchCallHistoryReq) (ExportResp, error) {
	var (
		calls []*model.Call
		m     model.Call
		ns    []NameOfSeat
	)
	db := scrm.GormDB.WithContext(ctx).
		Preload("Label").
		Preload("SeatLabel").
		Preload("CallLabel").
		Preload("HangupLabel").
		Preload("Sentences", func(db *gorm.DB) *gorm.DB {
			return db.Order("`index`")
		}).
		Joins("Order").
		Preload("Order.Project").
		//Joins("left join scrm_order on scrm_order.id=order_id").
		Scopes(
			actions.DeptPermission(ctx, "Order"),
			gormscope.CreateDateTimeRange(req.Start, req.End, m.TableName()),
			SearchCallScope(req),
		)
	if req.ProjectID > 0 {
		db = db.Where("Order.project_id = ?", req.ProjectID)
	}
	db = db.Find(&calls)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("search call history error", err.Error())
		return ExportResp{}, err
	}
	seatIDs := make([]int, 0)
	for _, v := range calls {
		if v.SeatID > 0 {
			seatIDs = append(seatIDs, v.SeatID)
		}
	}
	seatMap := make(map[int]NameOfSeat, len(seatIDs))
	if len(seatIDs) > 0 {
		err := scrm.GormDB.WithContext(ctx).
			Table("scrm_seat").
			Joins("left join sys_user on sys_user.user_id = scrm_seat.user_id").
			Select("scrm_seat.id,scrm_seat.nickname,sys_user.username").
			Where("scrm_seat.id IN ?", seatIDs).
			Scan(&ns).Error
		if err != nil {
			scrm.Logger().WithContext(ctx).Error("get username and nickname err", err.Error())
			return ExportResp{}, err
		}
	}
	for _, v := range ns {
		seatMap[v.SeatID] = v
	}
	columns := []string{"序号", "通话编号", "工单编号", "电话号码", "项目编号", "创建时间",
		"通话记录", "模型标签", "通话标签", "坐席标签", "挂断标签", "备注", "呼出给客户时刻", "客户接起时刻",
		"转换时刻", "呼出给坐席时刻", "坐席接起时刻", "挂断时刻", "客户响铃时长", "坐席响铃时长",
		"机器人通话时长", "坐席通话时长", "客户等待转接时长", "综合通话时长", "坐席用户名", "坐席名", "线路"}
	data, filename, err := util.CreateExcelFile(
		callToSlice(calls, seatMap),
		columns,
		"通话记录",
	)
	if err != nil {
		return ExportResp{}, err
	}
	return ExportResp{data, filename}, nil
}

func callToSlice(calls []*model.Call, seatMap map[int]NameOfSeat) [][]interface{} {
	var res [][]interface{}
	for index, item := range calls {
		var phone, project string
		if item.Order != nil {
			phone = util.HidePhone(item.Order.Phone)
			if item.Order.Project != nil {
				project = item.Order.Project.Name
			}
		}
		s := []interface{}{
			index + 1,
			item.ID,
			item.OrderID,
			phone,
			project,
			item.CreatedAt.Format(util.TimeLayoutDatetimeN),
			strings.Join(util.Convert(
				item.Sentences,
				func(s model.Sentence) string {
					return RoleMap[s.Role] + ": " + s.Text
				}), "\n"),
			func() string {
				if item.Label != nil {
					return item.Label.Name
				}
				return ""
			}(),
			func() string {
				if item.CallLabel != nil {
					return item.CallLabel.Name
				}
				return ""
			}(),
			func() string {
				if item.SeatLabel != nil {
					return item.SeatLabel.Name
				}
				return ""
			}(),
			func() string {
				if item.HangupLabel != nil {
					return item.HangupLabel.Name
				}
				return ""
			}(),
			item.Comment,
			func() string {
				if item.DialUpCustomTime.Valid {
					return item.DialUpCustomTime.Time.Format(util.TimeLayoutDatetimeN)
				}
				return ""
			}(),
			func() string {
				if item.CustomAnswerTime.Valid {
					return item.CustomAnswerTime.Time.Format(util.TimeLayoutDatetimeN)
				}
				return ""
			}(),
			func() string {
				if item.SwitchSeatTime.Valid {
					return item.SwitchSeatTime.Time.Format(util.TimeLayoutDatetimeN)
				}
				return ""
			}(),
			func() string {
				if item.DialUpSeatTime.Valid {
					return item.DialUpSeatTime.Time.Format(util.TimeLayoutDatetimeN)
				}
				return ""
			}(),
			func() string {
				if item.SeatAnswerTime.Valid {
					return item.SeatAnswerTime.Time.Format(util.TimeLayoutDatetimeN)
				}
				return ""
			}(),
			func() string {
				if item.HangUpTime.Valid {
					return item.HangUpTime.Time.Format(util.TimeLayoutDatetimeN)
				}
				return ""
			}(),
			item.CustomRingingDuration,
			item.SeatRingingDuration,
			item.AICallDuration,
			item.SeatCallDuration,
			item.SwitchingDuration,
			item.TotalCallDuration,
			//DurationFormat(item.CustomRingingDuration),
			//DurationFormat(item.SeatRingingDuration),
			//DurationFormat(item.AICallDuration),
			//DurationFormat(item.SeatCallDuration),
			//DurationFormat(item.SwitchingDuration),
			//DurationFormat(item.TotalCallDuration),
			seatMap[item.SeatID].SeatUserName,
			seatMap[item.SeatID].SeatName,
			item.Line.String,
		}
		res = append(res, s)
	}
	return res
}

func DurationFormat(d int64) string {
	return (time.Duration(d) * time.Second).String()
}

type ReportModelCallHistoryReq struct {
	CallID    string `json:"callId"`
	ProjectID int    `json:"projectId"`
	Role      int    `json:"role"`
	Index     int    `json:"index"`
	Text      string `json:"text"`
}

type ReportModelCallHistoryResp struct{}

func ReportModelCallHistory(ctx context.Context, req ReportModelCallHistoryReq) error {
	sentence := model.Sentence{
		CallID: req.CallID,
		Role:   req.Role,
		Index:  req.Index,
		Text:   req.Text,
	}
	res, err := json.Marshal(sentence)
	if err != nil {
		scrm.Logger().WithContext(ctx).Error("json marshal error", err.Error())
		return err
	}
	if err := scrm.RedisClient.RPush(ctx, config.ExtConfig.CacheSentence.LocalRedisKey, res).Err(); err != nil {
		scrm.Logger().WithContext(ctx).Error("operating redis error ", err.Error())
		return err
	}
	return nil
}

func SearchCallScope(req SearchCallHistoryReq) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if req.ProjectID > 0 {
			db = db.Where("Order.project_id = ?", req.ProjectID)
		}
		if req.OrderID > 0 {
			db = db.Where("order_id = ?", req.OrderID)
		}
		if len(req.CallID) > 0 {
			db = db.Where("scrm_call.id = ?", req.CallID)
		}
		if len(req.Phone) > 0 {
			db = db.Where("scrm_call.phone = ?", req.Phone)
		}
		if len(req.ModelLabelID) > 0 {
			db = db.Where("label_id in ?", req.ModelLabelID)
		}
		if len(req.CallLabelID) > 0 {
			db = db.Where("call_label_id in ?", req.CallLabelID)
		}
		if len(req.SeatLabelID) > 0 {
			db = db.Where("seat_label_id in ?", req.SeatLabelID)
		}
		if len(req.HangupLabelID) > 0 {
			db = db.Where("hangup_label_id in ?", req.HangupLabelID)
		}
		if req.SeatID > 0 {
			db = db.Where("scrm_call.seat_id = ?", req.SeatID)
		}
		return db
	}
}
