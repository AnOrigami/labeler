package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"go-admin/app/scrm"
	"go-admin/app/scrm/model"
	"go-admin/common/database"
	"go-admin/common/log"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"time"
)

type CTIManager struct {
	Ctx                  context.Context
	MaxRobotCon          int64
	MaxCTIQueueLen       int64
	Threshold            int64
	PushOrderKey         string
	PullCDRKey           string
	PullCallerChannelKey string
	GormDB               *gorm.DB
	CTIRDB               *redis.Client
	LocalRDB             *redis.Client
}

type PushRequest struct {
	Params    PushParams    `json:"params"`
	Variables PushVariables `json:"variables"`
}

type PushParams struct {
	Number string `json:"number"`
}

type PushVariables struct {
	OriginationUUID     string `json:"origination_uuid"`
	AbsoluteCodecString string `json:"absolute_codec_string"`
}

type TimestampString string

func (ts *TimestampString) SqlNullTime() sql.NullTime {
	if ts == nil {
		return sql.NullTime{}
	}
	timestamp, err := strconv.ParseInt(string(*ts), 10, 64)
	if timestamp == 0 || err != nil {
		return sql.NullTime{}
	}
	tm := time.Unix(timestamp/1000000, (timestamp%1000000)*1000)
	return sql.NullTime{
		Time:  tm,
		Valid: true,
	}
}

type Callflow struct {
	ProfileIndex  string `json:"profile_index"`
	CallerProfile struct {
		Username          string `json:"username"`
		Dialplan          string `json:"dialplan"`
		CallerIDName      string `json:"caller_id_name"`
		Ani               string `json:"ani"`
		Aniii             string `json:"aniii"`
		CallerIDNumber    string `json:"caller_id_number"`
		NetworkAddr       string `json:"network_addr"`
		Rdnis             string `json:"rdnis"`
		DestinationNumber string `json:"destination_number"`
		UUID              string `json:"uuid"`
		Source            string `json:"source"`
		Context           string `json:"context"`
		ChanName          string `json:"chan_name"`
	} `json:"caller_profile"`
	Times struct {
		CreatedTime        *TimestampString `json:"created_time"`
		ProfileCreatedTime *TimestampString `json:"profile_created_time"`
		ProgressTime       *TimestampString `json:"progress_time"`
		ProgressMediaTime  *TimestampString `json:"progress_media_time"`
		AnsweredTime       *TimestampString `json:"answered_time"`
		BridgedTime        *TimestampString `json:"bridged_time"`
		LastHoldTime       *TimestampString `json:"last_hold_time"`
		HoldAccumTime      *TimestampString `json:"hold_accum_time"`
		HangupTime         *TimestampString `json:"hangup_time"`
		ResurrectTime      *TimestampString `json:"resurrect_time"`
		TransferTime       *TimestampString `json:"transfer_time"`
	} `json:"times"`
}

type CDRDetailVariables struct {
	CTIDialNumber string `json:"cti_dial_number"`
	CallSource    string `json:"call_source"` // 1:queuedialer? 2:robot?
	Line          string `json:"cti_line_name"`
	LineGroup     string `json:"cti_line_group_name"`
}

type CDRDetail struct {
	Variables CDRDetailVariables `json:"variables"`
	Callflow  []Callflow         `json:"callflow"`
}

type CDRDetailJSON CDRDetail

func (d *CDRDetailJSON) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	//os.WriteFile("cdr-2.json", []byte(s), 0644)
	return json.Unmarshal([]byte(s), (*CDRDetail)(d))
}

// 1-create
// 1-answer
// 1-hangup
// 2-create
// 2-answer
// 客户响铃时长 1-hangup - 1-create(1 answer) || 1-answer - 1-create(1 no answer)
// 机器人通话时长 1-hangup - 1-answer(if no 2) || 2-answer - 1-answer(if 2)
// 坐席通话时长 2-hangup - 2-answer
// 坐席响铃时长
// 客户等待转接时长 2-answer - 2-create?
// 综合通话时长 1-hangup - 1-answer

type CDR struct {
	UUID        string        `json:"uuid"`
	BridgeUUID  string        `json:"bridge_uuid"`
	Account     string        `json:"account"`
	AudioFile   string        `json:"record_filename"`
	HangupCause string        `json:"hangup_cause"`
	DA2Result   string        `json:"da2_result"`
	Details     CDRDetailJSON `json:"details"`
}

type CallerChannel struct {
	Username string   `json:"username"`
	Notify   string   `json:"notify"`
	Current  string   `json:"current"`
	Activity []string `json:"activity"`
}

func (m CTIManager) Run() {
	go m.CleanOrders()
	go m.CloseProject()
	go m.PullCDR()
	go m.PushOrder()
	//go m.PullCallerChannel()
}

func (m CTIManager) CleanOrders() {
	for {
		_ = log.WithTracer(context.Background(), PackageName, "CTIManager CleanOrders", func(ctx context.Context) error {
			db := m.GormDB.WithContext(ctx).
				Model(&model.Order{}).
				Where("status=?", "处理中").
				Where("updated_at<?", time.Now().Add(-60*time.Minute)).
				Update("status", "已完成")
			log.LogAttr(ctx, log.Key("cti.cleanOrders.count").Int64(db.RowsAffected))
			if err := db.Error; err != nil {
				scrm.Logger().WithContext(ctx).Error(err.Error())
			}
			cleanOrdersCounter.Add(ctx, db.RowsAffected)
			return nil
		})
		time.Sleep(time.Minute)
	}
}

func (m CTIManager) CloseProject() {
	for {
		_ = log.WithTracer(context.Background(), PackageName, "CTIManager CloseProject", func(ctx context.Context) error {
			projectSet := map[int]bool{}
			seats, err := GetRedisSeats(ctx)
			if err != nil {
				return err
			}
			for _, seat := range seats {
				if seat.CheckIn {
					log.LogAttr(ctx, log.Key("cti.close.seat.checkedIn.callId").String(seat.CallID))
					log.LogAttr(ctx, log.Key("cti.close.seat.checkedIn.projects").IntSlice(seat.Projects))
					for _, p := range seat.Projects {
						projectSet[p] = true
					}
				}
			}
			projects := make([]int, 0, len(projectSet))
			for p := range projectSet {
				projects = append(projects, p)
			}
			log.LogAttr(ctx, log.Key("cti.close.projects").IntSlice(projects))
			db := scrm.GormDB.WithContext(ctx).
				Model(&model.Project{}).
				Where("running")
			if len(projects) > 0 {
				db = db.Where("id NOT IN (?)", projects)
			}
			db = db.Update("running", false)
			if err := db.Error; err != nil {
				scrm.Logger().WithContext(ctx).Error(err.Error())
				return err
			}
			closeProjectCounter.Add(ctx, db.RowsAffected)
			return nil
		})
		time.Sleep(time.Minute)
	}
}

func (m CTIManager) PullCallerChannel() {
	for {
		err := log.WithTracer(context.Background(), PackageName, "CTIManager PullCallerChannel", func(ctx context.Context) error {
			val, err := m.CTIRDB.BLPop(ctx, 1*time.Minute, m.PullCallerChannelKey).Result()
			if err != nil {
				if err == redis.Nil {
					scrm.Logger().WithContext(ctx).Debug("CTIManager PullCallerChannel timeout")
					return nil
				}
				scrm.Logger().WithContext(ctx).Error("CTIManager PullCallerChannel pull caller channel error ", err.Error())
				return err
			}
			scrm.Logger().WithContext(ctx).Debugf("CTIManager PullCallerChannel data: %s", val[1])
			var caller CallerChannel
			err = json.Unmarshal([]byte(val[1]), &caller)
			if err != nil {
				scrm.Logger().WithContext(ctx).Error("CTIManager PullCallerChannel channel msg json unmarshal error ", err.Error())
				return err
			}
			log.LogAttr(ctx,
				log.Key("cti.pull.caller.username").String(caller.Username),
				log.Key("cti.pull.caller.notify").String(caller.Notify),
				log.Key("cti.pull.caller.current").String(caller.Current),
			)
			if caller.Username == "" || caller.Notify != "hangup" {
				scrm.Logger().WithContext(ctx).Debugf("CTIManager PullCallerChannel wrong username or notify")
				return nil
			}

			var seat model.Seat
			db := m.GormDB.Where("line = ?", caller.Username).Limit(1).Find(&seat)
			if err = db.Error; err != nil {
				scrm.Logger().WithContext(ctx).Error("CTIManager PullCallerChannel get seat error ", err.Error())
				return err
			}
			log.LogAttr(ctx,
				log.Key("cti.pull.seat.rows").Int64(db.RowsAffected),
				log.Key("cti.pull.seat.id").Int(seat.ID),
			)
			if db.RowsAffected > 0 {
				_, err = DefaultSeatHub.seatStateStore.Update(ctx, strconv.Itoa(seat.ID), func(data *SeatWSEventDataStateChanged) *SeatWSEventDataStateChanged {
					if data == nil {
						data = &SeatWSEventDataStateChanged{}
					}
					log.LogAttr(ctx,
						log.Key("cti.pull.seatWsEventDataStateChanged.locked").Bool(data.Locked),
						log.Key("cti.pull.seatWsEventDataStateChanged.callId").String(data.CallID),
					)
					scrm.Logger().WithContext(ctx).Debugf("CTIManager PullCallerChannel seat %d: %#v", seat.ID, *data)
					if !data.Locked && data.CallID == "" {
						return nil
					}
					order := model.Order{}
					db = m.GormDB.
						WithContext(ctx).
						Model(&order).
						Joins("left join scrm_call sc on scrm_order.id = sc.order_id").
						Where("sc.id = ?", data.CallID).
						Find(&order)
					if err = db.Error; err != nil {
						scrm.Logger().WithContext(ctx).Error("CTIManager PullCallerChannel get order error ", err.Error())
					}
					log.LogAttr(ctx, log.Key("cti.pull.order.id").Bool(data.Locked))
					if order.ID > 0 {
						err = m.GormDB.WithContext(ctx).Model(&order).Update("status", OrderStatusFinished).Error
						if err != nil {
							scrm.Logger().WithContext(ctx).Error("CTIManager PullCallerChannel update order status error ", err.Error())
						}
					}

					data.Locked = false
					data.CallID = ""
					return data
				})
				if err != nil {
					scrm.Logger().WithContext(ctx).Error("CTIManager PullCallerChannel store update", err.Error())
					return err
				}
			}
			log.LogAttr(ctx, log.Key("cti.pull.closeProject.count").Int64(db.RowsAffected))
			return nil
		})
		if err != nil {
			time.Sleep(5 * time.Second)
		}
	}
}

func (m CTIManager) PushOrder() {
	for {
		err := log.WithTracer(context.Background(), PackageName, "CTIManager PushOrder", func(ctx context.Context) error {
			//ok, err := m.Check(ctx)
			orders, err := MakeCheckFun(ctx, m)()
			if err != nil {
				return err
			}
			if len(orders) == 0 {
				time.Sleep(1 * time.Second)
				return nil
			}

			var (
				calls    = make([]model.Call, len(orders))
				orderIDs = make([]int, len(orders))
				tasks    = make([]interface{}, len(orders))
			)
			for i, order := range orders {
				callID, err := uuid.NewUUID()
				if err != nil {
					scrm.Logger().WithContext(ctx).Error("CTIManager PushOrder new uuid error ", err.Error())
					return err
				}
				callIDStr := strconv.Itoa(order.ProjectID) + "-" + callID.String()
				msg := PushRequest{
					Params:    PushParams{Number: order.Phone},
					Variables: PushVariables{OriginationUUID: callIDStr},
				}
				b, err := json.Marshal(msg)
				if err != nil {
					scrm.Logger().WithContext(ctx).Error("CTIManager PushOrder marshal push msg error ", err.Error())
					return err
				}

				calls[i] = model.Call{
					ID:      callIDStr,
					OrderID: order.ID,
					Phone:   order.Phone,
				}
				orderIDs[i] = order.ID
				tasks[i] = string(b)
			}
			scrm.Logger().WithContext(ctx).Debugf("orderIDs: %v", orderIDs)
			scrm.Logger().WithContext(ctx).Debugf("tasks: %v", tasks)
			log.LogAttr(ctx, log.Key("cti.push.orderIds").IntSlice(orderIDs))
			err = m.GormDB.Transaction(func(tx *gorm.DB) error {
				{
					db := tx.WithContext(ctx).Create(calls)
					if err := db.Error; err != nil {
						scrm.Logger().WithContext(ctx).Error(err.Error())
						return err
					}
				}
				{
					db := tx.WithContext(ctx).
						Model(&model.Order{}).
						Where("id IN (?)", orderIDs).
						Update("status", OrderStatusProcessing)
					if err := db.Error; err != nil {
						scrm.Logger().WithContext(ctx).Error(err.Error())
						return err
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
			if err := m.CTIRDB.RPush(ctx, m.PushOrderKey, tasks...).Err(); err != nil {
				scrm.Logger().WithContext(ctx).Error(err.Error())
				return err
			}
			pushOrdersCounter.Add(ctx, int64(len(orders)))
			return nil
		})
		if err != nil {
			scrm.Logger().WithContext(context.Background()).Error(err.Error())
			time.Sleep(10 * time.Second)
		}
	}
}

const (
	CTIHangupCauseNormal = "NORMAL_CLEARING" // 正常通话后挂断
)

var CallLabelMap = map[string]string{
	"hold on":        model.LabelNameCallHoldOn,
	"ringback":       model.LabelNameCallNoAnswer,
	"not answer":     model.LabelNameCallNoAnswer,
	"busy now":       model.LabelNameCallDeny,
	"not convenient": model.LabelNameCallDeny,
	"does not exis":  model.LabelNameCallNotExists,
}

func GetCallLabelID(ctx context.Context, cdr CDR) (int, error) {
	var name string
	if len(cdr.Details.Callflow) > 0 && cdr.Details.Callflow[0].Times.AnsweredTime.SqlNullTime().Valid {
		name = "接通"
	} else if n, exists := CallLabelMap[cdr.DA2Result]; exists {
		name = n
	} else {
		name = model.LabelNameCallOther
	}
	var label model.Label
	db := scrm.GormDB.
		WithContext(ctx).
		Where("name=?", name).
		First(&label)
	if err := db.Error; err != nil {
		return 0, err
	}
	return label.ID, nil
}

func (m CTIManager) PullCDR() {
	for {
		err := log.WithTracer(context.Background(), PackageName, "CTIManager PullCDR", func(ctx context.Context) error {
			val, err := m.CTIRDB.BLPop(ctx, 1*time.Minute, m.PullCDRKey).Result()
			if err != nil {
				if err == redis.Nil {
					return nil
				}
				scrm.Logger().WithContext(ctx).Error("CTIManager PullCDR pull cdr msg error ", err.Error())
				return err
			}
			scrm.Logger().WithContext(ctx).Debugf("CTIManager PullCDR data: %s", val[1])
			var cdr CDR
			err = json.Unmarshal([]byte(val[1]), &cdr)
			if err != nil {
				scrm.Logger().WithContext(ctx).Error("CTIManager PullCDR cdr msg json unmarshal error ", err.Error())
				return err
			}

			log.LogAttr(ctx,
				log.Key("cti.pull.cdr.uuid").String(cdr.UUID),
				log.Key("cti.pull.cdr.bridgeUUID").String(cdr.BridgeUUID),
				log.Key("cti.pull.cdr.status").String(cdr.DA2Result),
				log.Key("cti.pull.cdr.answered").Bool(len(cdr.Details.Callflow) > 0 && cdr.Details.Callflow[0].Times.AnsweredTime.SqlNullTime().Valid),
			)

			if cdr.UUID == "" {
				return nil
			}

			ids := make([]string, 1, 2)
			ids[0] = cdr.UUID
			if cdr.BridgeUUID != "" {
				ids = append(ids, cdr.BridgeUUID)
			}

			var call model.Call
			db := m.GormDB.WithContext(ctx).Where("id IN (?)", ids).Limit(1).Find(&call)
			if err = db.Error; err != nil {
				scrm.Logger().WithContext(ctx).Error("CTIManager PullCDR get call error ", err.Error())
				return err
			} else if db.RowsAffected == 0 {
				scrm.Logger().WithContext(ctx).Warnf("CTIManager PullCDR uuid not in database: %v", ids)
				return nil
			}

			log.LogAttr(ctx, log.Key("cti.pull.cdr.callId").String(call.ID))

			var (
				createdTime  sql.NullTime
				answeredTime sql.NullTime
				hangupTime   sql.NullTime
				callFlow     = cdr.Details.Callflow
			)
			if len(callFlow) > 0 {
				createdTime = callFlow[0].Times.CreatedTime.SqlNullTime()
				answeredTime = callFlow[0].Times.AnsweredTime.SqlNullTime()
				hangupTime = callFlow[0].Times.HangupTime.SqlNullTime()
			} else {
				scrm.Logger().WithContext(ctx).Error("no valid data in cdr.detail.callflow")
			}

			if call.ID == cdr.UUID { // stage-1
				ss := strings.Split(cdr.AudioFile, "/")
				if len(ss) > 1 {
					call.AudioFile = ss[len(ss)-2] + "/" + ss[len(ss)-1]
				}
				call.DialUpCustomTime = createdTime
				call.CustomAnswerTime = answeredTime
				call.HangUpTime = hangupTime
				if labelID, err := GetCallLabelID(ctx, cdr); err != nil {
					scrm.Logger().WithContext(ctx).Error(err.Error())
				} else {
					call.CallLabelID = sql.NullInt64{Int64: int64(labelID), Valid: true}
				}
				if _, err := DefaultSeatHub.seatStateStore.UnlockSeat(ctx, "", call.ID); err != nil {
					scrm.Logger().WithContext(ctx).Error(err.Error())
				}

				{
					var order model.Order
					db := scrm.GormDB.Model(&order).
						WithContext(ctx).
						Joins("left join scrm_call sc on scrm_order.id = sc.order_id").
						Where("sc.id = ?", call.ID).
						Find(&order)
					if err := db.Error; err != nil {
						scrm.Logger().WithContext(ctx).Error("get order error: ", err.Error())
					}
					log.LogAttr(ctx, log.Key("cti.pull.order.id").Int(order.ID))
					if order.ID > 0 {
						err := scrm.GormDB.Model(&order).
							WithContext(ctx).
							Update("status", OrderStatusFinished).Error
						if err != nil {
							scrm.Logger().WithContext(ctx).Error("update order status error: ", err.Error())
						}
					}
				}
			} else { // stage-2 call.ID == cdr.BridgeUUID
				//var seat model.Seat
				//db = m.GormDB.WithContext(ctx).Where("line = ?", cdr.Details.Variables.Line).Limit(1).Find(&seat)
				//if err = db.Error; err != nil {
				//	scrm.Logger().WithContext(ctx).Error("CTIManager PullCDR get seat error ", err.Error())
				//	//return err
				//} else if db.RowsAffected != 1 {
				//	scrm.Logger().WithContext(ctx).Errorf("CTIManager PullCDR cannot find seat with line %s, raw input: %s", cdr.Account, val[1])
				//	//return nil
				//} else {
				//	call.SeatID = seat.ID
				//}
				call.DialUpSeatTime = createdTime
				call.SeatAnswerTime = answeredTime
				call.HangUpTime = hangupTime
				call.Line = database.NewNullString(cdr.Details.Variables.Line)
			}
			call.UpdateDuration()
			db = scrm.GormDB.
				WithContext(ctx).
				//Omit("SeatID", "LabelID", "SeatLabelID", "HangupLabelID", "Comment").
				Select(
					"AudioFile",
					"DialUpCustomTime",
					"CustomAnswerTime",
					"HangUpTime",
					"CallLabelID",
					"DialUpSeatTime",
					"SeatAnswerTime",
					"HangUpTime",
					"Line",
					"CustomRingingDuration",
					"SeatRingingDuration",
					"AICallDuration",
					"SeatCallDuration",
					"SwitchingDuration",
					"TotalCallDuration",
				).
				Updates(&call)
			if err := db.Error; err != nil {
				scrm.Logger().WithContext(ctx).Error(err.Error())
				return err
			}
			pullCDRCounter.Add(ctx, 1)
			return nil
		})
		if err != nil {
			time.Sleep(5 * time.Second)
		}
	}
}

type ProjectSeatCount struct {
	ID         int
	BusySeatC  float64
	SpareSeatC float64
	Count      int `gorm:"column:c;"`
}

func GetRedisSeats(ctx context.Context) ([]SeatWSEventDataStateChanged, error) {
	keys, err := scrm.RedisClient.Keys(ctx, "seat:*").Result()
	if err != nil {
		err = fmt.Errorf("CTIManager Check redis get seats error %s", err.Error())
		scrm.Logger().WithContext(ctx).Error(err.Error())
		return nil, err
	}
	log.LogAttr(ctx, log.Key("cti.getRedisSeats.getSeatsResult").StringSlice(keys))
	cmds, err := scrm.RedisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, k := range keys {
			pipe.Get(ctx, k)
		}
		return nil
	})
	if err != nil {
		scrm.Logger().WithContext(ctx).Error(err.Error())
		return nil, err
	}
	log.LogAttr(ctx, log.Key("cti.getRedisSeats.count").Int(len(cmds)))
	seats := make([]SeatWSEventDataStateChanged, 0, len(cmds))
	for _, cmd := range cmds {
		var seat SeatWSEventDataStateChanged
		err = json.Unmarshal([]byte(cmd.(*redis.StringCmd).Val()), &seat)
		if err != nil {
			scrm.Logger().WithContext(ctx).Error("CTIManager Check json unmarshal seats error ", err.Error())
			continue
		}
		seats = append(seats, seat)
	}
	return seats, nil
}

func MakeCheckFun(ctx context.Context, m CTIManager) func() ([]model.Order, error) { //返回参数： 哪些可用，取几个 有啥错误
	return func() ([]model.Order, error) {
		// 1、检查cti推送工单队列数量，若大于阈值，则等待, 小于阈值，则计算下次拉多少个
		ctiQueueLen, err := m.CTIRDB.LLen(ctx, m.PushOrderKey).Result()
		if err != nil {
			scrm.Logger().WithContext(ctx).Error("CTIManager Check get cti queue length error: ", err.Error())
			return nil, err
		}
		log.LogAttr(ctx,
			log.Key("log.cti.check.ctiQueueLen").Int64(ctiQueueLen),
			log.Key("log.cti.check.maxCtiQueueLen").Int64(m.MaxCTIQueueLen),
		)
		scrm.Logger().WithContext(ctx).Debugf("ctiQueueLen %d MaxCTIQueueLen %d", ctiQueueLen, m.MaxCTIQueueLen)
		if ctiQueueLen >= m.MaxCTIQueueLen {
			scrm.Logger().WithContext(ctx).Warn("CTIManager Check cti queue number is greater than max number")
			return nil, nil
		}
		var maxCountToPush = m.MaxCTIQueueLen - ctiQueueLen
		// 2、检查正在进行中的工单，若数量大于机器人最大数量，则等待
		var realLoad int64
		err = m.GormDB.WithContext(ctx).Model(&model.Order{}).Where("status = ?", OrderStatusProcessing).Count(&realLoad).Error
		if err != nil {
			scrm.Logger().WithContext(ctx).Error("CTIManager Check realLoad processing order error")
			return nil, err
		}
		log.LogAttr(ctx,
			log.Key("log.cti.check.realLoad").Int64(realLoad),
			log.Key("log.cti.check.MaxRobotCon").Int64(m.MaxRobotCon),
		)
		scrm.Logger().WithContext(ctx).Debugf("realLoad %d MaxRobotCon %d", realLoad, m.MaxRobotCon)
		if realLoad >= m.MaxRobotCon {
			err := errors.New("CTIManager Check robot is busy")
			scrm.Logger().WithContext(ctx).Warn(err.Error())
			return nil, err
		}
		if maxCountToPush > m.MaxRobotCon-realLoad {
			maxCountToPush = m.MaxRobotCon - realLoad
		}
		// 遍历坐席
		keys, err := m.LocalRDB.Keys(ctx, "seat*").Result()
		if err != nil {
			return nil, fmt.Errorf("CTIManager Check redis get seats error ")
		}
		cmds, err := m.LocalRDB.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			for _, k := range keys {
				pipe.Get(ctx, k)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("CTIManager Check redis get seats error, system error is %s", err.Error())
		}
		projectIDSet := map[int]bool{}
		seats := make([]SeatWSEventDataStateChanged, 0, len(cmds))
		for _, cmd := range cmds {
			var seat SeatWSEventDataStateChanged
			err = json.Unmarshal([]byte(cmd.(*redis.StringCmd).Val()), &seat)
			if err != nil {
				scrm.Logger().WithContext(ctx).Error("CTIManager Check json unmarshal seats error ", err.Error())
				continue
			}
			if !seat.CheckIn {
				continue
			}
			if !seat.Ready && !seat.PreReady {
				continue
			}
			for _, p := range seat.Projects {
				projectIDSet[p] = true
			}
			seats = append(seats, seat)
		}
		if len(projectIDSet) == 0 {
			return nil, nil
		}
		projectIDs := make([]int, 0, len(projectIDSet))
		for p := range projectIDSet {
			projectIDs = append(projectIDs, p)
		}
		{
			b, _ := json.Marshal(seats)
			scrm.Logger().WithContext(ctx).Debugf("projectIDs: %v", projectIDs)
			scrm.Logger().WithContext(ctx).Debugf("seats: %s", b)
			log.LogAttr(ctx, log.Key("log.cti.check.projectIds").IntSlice(projectIDs))
		}
		var projectMap map[int]ProjectSeatCount
		{
			var pscs []ProjectSeatCount
			db := scrm.GormDB.WithContext(ctx).
				Table(model.Project{}.TableName()+" p").
				Select("p.id", "IFNULL(pc.c, 0) c", "p.spare_seat_c", "p.busy_seat_c").
				Joins(
					"LEFT JOIN (?) pc ON pc.project_id=p.id",
					scrm.GormDB.
						Table(model.Order{}.TableName()+" o").
						Select("COUNT(o.project_id) c", "o.project_id").
						Where("o.status=?", OrderStatusProcessing).
						Where("o.project_id IN (?)", projectIDs).
						Group("o.project_id"),
				).
				Where("p.id IN (?)", projectIDs).
				Where("p.running").
				Scan(&pscs)
			if err := db.Error; err != nil {
				scrm.Logger().WithContext(ctx).Error(err.Error())
				return nil, err
			}
			{
				b, _ := json.Marshal(pscs)
				scrm.Logger().WithContext(ctx).Debugf("realProjectLoads: %s", b)
			}
			projectMap = make(map[int]ProjectSeatCount, len(pscs))
			for _, p := range pscs {
				projectMap[p.ID] = p
			}
		}
		maxProjectLoadMap := map[int]float64{}
		for _, seat := range seats {
			if seat.Locked {
				for _, projectID := range seat.Projects {
					maxProjectLoadMap[projectID] += projectMap[projectID].BusySeatC + 1.0
				}
			} else {
				for _, projectID := range seat.Projects {
					maxProjectLoadMap[projectID] += projectMap[projectID].SpareSeatC
				}
			}
		}
		{
			b, _ := json.Marshal(maxProjectLoadMap)
			log.LogAttr(ctx, log.Key("log.cti.check.maxProjectLoadMap").String(string(b)))
			scrm.Logger().WithContext(ctx).Debugf("maxProjectLoadMap: %s", b)
		}
		var validProjectIDs []int
		validProjectLoadMap := map[int]float64{}
		for projectID, maxProjectLoad := range maxProjectLoadMap {
			psc, exists := projectMap[projectID]
			if !exists {
				continue
			}
			if capacitor := maxProjectLoad - float64(psc.Count); capacitor > 0 {
				validProjectIDs = append(validProjectIDs, projectID)
				validProjectLoadMap[projectID] = capacitor
			}
		}
		log.LogAttr(ctx, log.Key("log.cti.check.validProjectIDs").IntSlice(validProjectIDs))
		scrm.Logger().WithContext(ctx).Debugf("validProjectIDs: %v", validProjectIDs)
		if len(validProjectIDs) == 0 {
			return nil, nil
		}
		var orders []model.Order
		{
			db := scrm.GormDB.WithContext(ctx).
				Where("status=?", OrderStatusWaiting).
				Where("project_id IN (?)", validProjectIDs).
				Limit(int(maxCountToPush)).
				Find(&orders)
			if err := db.Error; err != nil {
				scrm.Logger().WithContext(ctx).Error(err.Error())
				return nil, err
			} else if db.RowsAffected == 0 {
				return nil, nil
			}
		}
		log.LogAttr(ctx, log.Key("cti.check.order.length").Int(len(orders)))
		// 把超过项目并发数的order排除掉
		validOrders := make([]model.Order, 0, len(orders))
		for _, order := range orders {
			if validProjectLoadMap[order.ProjectID] > 0 {
				validOrders = append(validOrders, order)
				validProjectLoadMap[order.ProjectID] -= 1
			}
		}
		return validOrders, nil
	}
}
