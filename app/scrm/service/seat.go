package service

import (
	"context"
	"errors"
	"fmt"
	"go-admin/app/scrm"
	"go-admin/app/scrm/model"
	"go-admin/common/actions"
	"go-admin/common/database"
	"go-admin/common/gormscope"
	"gorm.io/gorm"
	"strconv"
	"time"
)

const (
	SeatWSEventCheckInChanged   = "checkinChanged"
	SeatWSEventReadinessChanged = "readinessChanged"
	SeatWSEventLockedChanged    = "lockedChanged"
	SeatWSEventStateChanged     = "stateChanged"
	SeatWSEventError            = "error"
	PongWait                    = 10 * time.Second
	pingPeriod                  = (PongWait * 9) / 10
	WriteWait                   = 3 * time.Second
)

type CreateSeatReq struct {
	Nickname  string `json:"nickname"`
	UserID    int    `json:"userId"`
	Line      string `json:"line"`
	LineGroup string `json:"lineGroup"`
}

func CreateSeat(ctx context.Context, req CreateSeatReq) error {
	if req.UserID == 0 {
		return errors.New("userid error")
	}
	s := model.Seat{
		ID:        req.UserID,
		Nickname:  req.Nickname,
		UserID:    req.UserID,
		DeptID:    0,
		Line:      database.NewNullString(req.Line),
		LineGroup: database.NewNullString(req.LineGroup),
	}
	db := scrm.GormDB.WithContext(ctx).Create(&s)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("坐席创建失败: ", err.Error())
		return err
	}
	return nil
}

type Seat struct {
	UserID    int64  `json:"userId"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	IsSeat    bool   `json:"isSeat"`
	Line      string `json:"line"`
	LineGroup string `json:"lineGroup"`
}

type GetSeatListReq struct {
	UserName string `json:"userName"`
	Nickname string `json:"nickname"`
	Pagination
}

func GetSeatList(ctx context.Context, p *actions.DataPermission, req GetSeatListReq) ([]Seat, int64, error) {
	var (
		u     []Seat
		count int64
	)
	db := scrm.GormDB.WithContext(ctx).
		Table("sys_user u").
		Joins("left join scrm_seat s on s.user_id=u.user_id").
		Select("u.user_id, u.username, ifnull(s.nickname,'') nickname, (s.user_id is not null) is_seat,s.line,s.line_group").
		Joins("inner join sys_dept dept on dept.dept_id=u.dept_id").
		Scopes(
			gormscope.Paginate(&req.Pagination),
			GetSeatListScope(req),
		).
		Where("dept.dept_path like ?", fmt.Sprintf("%%/%d/%%", p.DeptId)).
		Where("s.deleted_at is null").
		Where("u.deleted_at is null").
		Scan(&u)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("坐席列表查找失败: ", err.Error())
		return nil, 0, err
	}
	db = db.Limit(-1).Offset(-1).Count(&count)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("count call error", err.Error())
		return nil, 0, err
	}
	return u, count, nil
}

func GetSeatListScope(req GetSeatListReq) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if len(req.UserName) > 0 {
			db = db.Where("u.username = ?", req.UserName)
		}
		if len(req.Nickname) > 0 {
			db = db.Where("s.nickname = ?", req.Nickname)
		}
		return db
	}
}

type UpdateSeatReq struct {
	ID        int    `json:"id"`
	Nickname  string `json:"nickname"`
	Line      string `json:"line"`
	LineGroup string `json:"lineGroup"`
}

func UpdateSeat(ctx context.Context, req UpdateSeatReq) error {
	s := model.Seat{
		ID:        req.ID,
		Nickname:  req.Nickname,
		Line:      database.NewNullString(req.Line),
		LineGroup: database.NewNullString(req.LineGroup),
	}
	db := scrm.GormDB.
		WithContext(ctx).
		Select("Nickname", "Line", "LineGroup").
		Updates(&s)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("坐席更新失败: ", err.Error())
		return err
	}
	return nil
}

func DelSeat(ctx context.Context, id int) error {
	db := scrm.GormDB.WithContext(ctx).Delete(&model.Seat{}, id)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("删除坐席失败: ", err.Error())
		return err
	}
	return nil
}

type ErrorData struct {
	Error string `json:"error"`
}

type MessageOut[T any] struct {
	Event string `json:"event"`
	Data  T      `json:"data"`
}

func NewMessage[T any](event string, data T) MessageOut[T] {
	return MessageOut[T]{
		Event: event,
		Data:  data,
	}
}

func NewErrorMessage(err string) MessageOut[ErrorData] {
	return NewMessage(SeatWSEventError, ErrorData{Error: err})
}

func SliceEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	if (a == nil) != (b == nil) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func RedisSeatKey(id string) string {
	res := fmt.Sprintf("seat:" + id)
	return res
}

type LockSeatReq struct {
	ProjectID int    `json:"projectId"`
	SeatID    int    `json:"seatId"`
	CallID    string `json:"callId"`
}

type LockSeatRes struct {
	Success bool `json:"success"`
}

func RedisCallSeatKey(callID string) string {
	return fmt.Sprintf("call:%s:seat", callID)
}

func LockSeat(ctx context.Context, req LockSeatReq) (LockSeatRes, error) {
	res := LockSeatRes{Success: false}
	if err := scrm.RedisClient.Set(ctx, RedisCallSeatKey(req.CallID), strconv.Itoa(req.SeatID), 24*time.Hour).Err(); err != nil {
		scrm.Logger().WithContext(ctx).Error(err.Error())
		return LockSeatRes{}, err
	}
	data, err := DefaultSeatHub.seatStateStore.Update(ctx, strconv.Itoa(req.SeatID), func(data *SeatWSEventDataStateChanged) *SeatWSEventDataStateChanged {
		if !data.Ready || !data.CheckIn || data.Locked {
			return nil
		}
		data.Locked = true
		data.Ready = false
		data.PreReady = false
		data.CallID = req.CallID
		res.Success = true
		return data
	})
	if err != nil {
		scrm.Logger().WithContext(ctx).Error("store update: ", err.Error())
		return LockSeatRes{}, err
	}
	if res.Success {
		massage := NewMessage(SeatWSEventLockedChanged, data)
		DefaultSeatHub.SendMessage(ctx, strconv.Itoa(req.SeatID), "", massage)
		{
			db := scrm.GormDB.WithContext(ctx).
				Model(&model.Call{}).
				Where("id=?", req.CallID).
				Update("seat_id", req.SeatID)
			if err := db.Error; err != nil {
				scrm.Logger().WithContext(ctx).Error(err.Error())
				// no return
			}
		}
	}
	return res, nil
}

type SeatState struct {
	ID             int    `json:"id"`
	Ready          bool   `json:"ready"`
	Locked         bool   `json:"locked"`
	Line           string `json:"line"`
	LineGroup      string `json:"lineGroup"`
	LockCount      int64  `json:"lockCount"`
	CallDuration   int64  `json:"callDuration"`
	ReadyTimestamp int64  `json:"readyTimestamp"`
}

func GetSeatListOfProject(ctx context.Context, projectID int) ([]SeatState, error) {
	var p model.Project
	var list []SeatState
	db := scrm.GormDB.WithContext(ctx).
		Preload("Seats").
		Find(&p, projectID)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("db ERROR: ", err.Error())
		return nil, err
	}
	for _, seat := range p.Seats {
		data, err := DefaultSeatHub.seatStateStore.Get(ctx, strconv.Itoa(seat.ID))
		if err != nil {
			scrm.Logger().WithContext(ctx).Error("Get UserState Error: ", err.Error())
			return nil, err
		}
		if data.CheckIn {
			for _, pID := range data.Projects {
				cache := SeatStatSvc.Get(seat.ID)
				if projectID == pID {
					list = append(list, SeatState{
						ID:             seat.ID,
						Ready:          data.Ready,
						Locked:         data.Locked,
						Line:           seat.Line.String,
						LineGroup:      seat.LineGroup.String,
						LockCount:      cache.TotalCallAmount,
						CallDuration:   cache.TotalCallDuration,
						ReadyTimestamp: data.ReadyTimestamp,
					})
				}
			}
		}
	}
	return list, nil
}

type UnlockSeatRes struct {
	Success bool `json:"success"`
}

func UnlockSeat(ctx context.Context, req LockSeatReq) (UnlockSeatRes, error) {
	ok, err := DefaultSeatHub.seatStateStore.UnlockSeat(ctx, strconv.Itoa(req.SeatID), req.CallID)
	if err != nil {
		return UnlockSeatRes{}, err
	}
	return UnlockSeatRes{Success: ok}, nil
}

type SearchProjectsOfSeatRespItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type SearchProjectsOfSeatReq struct {
	SeatID int
}

func SearchProjectsOfSeat(ctx context.Context, req SearchProjectsOfSeatReq) ([]SearchProjectsOfSeatRespItem, error) {
	var projects []*model.Project
	db := scrm.GormDB.WithContext(ctx).
		Select("id, name").
		Joins("left join scrm_project_seat on scrm_project.id=scrm_project_seat.project_id").
		Scopes(
			actions.DeptPermission(ctx, model.Project{}.TableName()),
		).
		Where("scrm_project_seat.seat_id = ?", req.SeatID).
		Order("id asc").
		Find(&projects)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("search project error", err.Error())
		return nil, err
	}
	items := make([]SearchProjectsOfSeatRespItem, len(projects))
	for i, project := range projects {
		items[i] = SearchProjectsOfSeatRespItem{
			ID:   project.ID,
			Name: project.Name,
		}
	}
	return items, nil
}

type SetSeatPreReadyReq struct {
	SeatID int
}

func SetSeatPreReady(ctx context.Context, req SetSeatPreReadyReq) error {
	if req.SeatID <= 0 {
		return errors.New("参数异常")
	}
	data, err := DefaultSeatHub.seatStateStore.Update(ctx, strconv.Itoa(req.SeatID), func(data *SeatWSEventDataStateChanged) *SeatWSEventDataStateChanged {
		if data == nil {
			data = &SeatWSEventDataStateChanged{}
		}
		data.PreReady = true
		return data
	})
	if err != nil {
		scrm.Logger().WithContext(ctx).Error("store update: ", err.Error())
		return err
	}
	massage := NewMessage(SeatWSEventStateChanged, data)
	DefaultSeatHub.SendMessage(ctx, strconv.Itoa(req.SeatID), "", massage)
	return nil
}
