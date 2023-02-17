package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/minio/minio-go/v7"
	"go-admin/app/scrm"
	"go-admin/app/scrm/model"
	"go-admin/common/actions"
	"go-admin/common/database"
	"go-admin/common/gormscope"
	"go-admin/common/log"
	"go-admin/common/util"
	"go-admin/config"
	"gorm.io/gorm"
	"io"
	"strconv"
	"time"
)

const (
	ExportTaskStatusWaiting = "导出中"
	ExportTaskStatusDone    = "已完成"
)

type ExportTask interface {
	Do(ctx context.Context, task model.ExportTask) (string, error)
	GetTaskType() string
}

type ExportTaskService struct {
	typesMap      map[string]ExportTask
	durationWait  time.Duration
	durationShare time.Duration
}

func MakeExportTaskService(durationWait, durationShare time.Duration, tasks ...ExportTask) ExportTaskService {
	typesMap := make(map[string]ExportTask)
	for _, v := range tasks {
		typesMap[v.GetTaskType()] = v
	}
	return ExportTaskService{
		durationWait:  durationWait,
		durationShare: durationShare,
		typesMap:      typesMap,
	}
}

func (s ExportTaskService) Run() {
	for {
		_ = log.WithTracer(context.Background(), PackageName, "Async Export Service", func(ctx context.Context) error {
			var task model.ExportTask
			err := scrm.GormDB.WithContext(ctx).
				Where("status = ?", ExportTaskStatusWaiting).
				First(&task).
				Error
			if err != nil {
				if err != gorm.ErrRecordNotFound {
					scrm.Logger().Error("query task error", err.Error())
				}
				time.Sleep(s.durationWait)
				return err
			}
			if typeExporter, ok := s.typesMap[task.Type]; ok {
				fileName, err := typeExporter.Do(ctx, task)
				if err != nil {
					scrm.Logger().WithContext(ctx).Error("task execute error ", err.Error())
					return err
				}
				task.Status = ExportTaskStatusDone
				task.FileName = fileName
				if err = scrm.GormDB.WithContext(ctx).
					Where("id = ?", task.ID).
					Updates(&task).Error; err != nil {
					scrm.Logger().Error("query task error: ", err.Error())
					return err
				}
			}
			return nil
		})
		time.Sleep(s.durationShare)
	}
}

type SearchExportTaskReq struct {
	Status string `json:"status"`

	Pagination
}

type SearchExportTaskRespItem struct {
	CreatedAt string `json:"createdAt"`
	Type      string `json:"type"`
	Args      string `json:"args"`
	Status    string `json:"status"`
	ID        int    `json:"id"`
}

func SearchExportTask(ctx context.Context, req SearchExportTaskReq) ([]SearchExportTaskRespItem, int64, error) {
	var (
		count int64
		tasks = make([]model.ExportTask, 0, req.PageSize)
	)
	db := scrm.GormDB.WithContext(ctx).
		Scopes(
			actions.DeptPermission(ctx, model.ExportTask{}.TableName()),
			gormscope.Paginate(&req.Pagination),
		)
	if req.Status != "" {
		db = db.Where("status = ?", req.Status)
	}
	db = db.Order("created_at desc").Find(&tasks)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("search export task database error ", db.Error.Error())
		return nil, 0, err
	}
	db = db.Limit(-1).Offset(-1).Count(&count)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("search export find total count database error")
		return nil, 0, err
	}
	items := make([]SearchExportTaskRespItem, len(tasks))
	for idx, task := range tasks {
		items[idx] = SearchExportTaskRespItem{
			CreatedAt: task.CreatedAt.Format(util.TimeLayoutDatetimeN),
			Type:      task.Type,
			Args:      task.Args,
			Status:    task.Status,
			ID:        task.ID,
		}
	}
	return items, count, nil
}

type ExportTaskFileReq struct {
	ID int `form:"id"`
}

type ExportTaskFileResp struct {
	File     *string `json:"file"`
	FileName string  `json:"fileName"`
}

func ExportTaskFile(ctx context.Context, req ExportTaskFileReq) (ExportTaskFileResp, error) {
	var task model.ExportTask
	err := scrm.GormDB.WithContext(ctx).
		Where("id = ?", req.ID).
		First(&task).Error
	if err != nil {
		scrm.Logger().WithContext(ctx).Error("find in sql error")
		return ExportTaskFileResp{}, err
	}
	if task.Status != ExportTaskStatusDone {
		return ExportTaskFileResp{}, errors.New("正在后台导出中,请稍候")
	}
	obj, err := scrm.MinIOClient.GetObject(ctx, config.ExtConfig.MinIO.ExportFileBucket, task.FileName, minio.GetObjectOptions{})
	if err != nil {
		scrm.Logger().WithContext(ctx).Error("minio get file: ", err.Error())
		return ExportTaskFileResp{}, err
	}
	defer obj.Close()
	fileContent, err := io.ReadAll(obj)
	if err != nil {
		scrm.Logger().WithContext(ctx).Error("read file: ", err.Error())
		return ExportTaskFileResp{}, err
	}
	result := base64.StdEncoding.EncodeToString(fileContent)
	return ExportTaskFileResp{
		&result,
		task.FileName,
	}, nil
}

// ExportCallTask 导出call表
type ExportCallTask struct {
	BatchSize int
}

func (t ExportCallTask) Do(ctx context.Context, task model.ExportTask) (string, error) {
	var (
		m   model.Call
		ns  []NameOfSeat
		req SearchCallHistoryReq
	)
	calls := make([]*model.Call, 0)
	if err := json.Unmarshal([]byte(task.Args), &req); err != nil {
		scrm.Logger().WithContext(ctx).Error("json unmarshal error")
		return "", err
	}
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
			actions.DeptPermissionFromDeptId(task.DeptID, "Order"),
			gormscope.CreateDateTimeRange(req.Start, req.End, m.TableName()),
			SearchCallScope(req),
		)
	if req.ProjectID > 0 {
		db = db.Where("Order.project_id = ?", req.ProjectID)
	}
	next := database.BatchQueryExcludeNull[model.Call](db, t.BatchSize, []database.OrderColumn{
		{ColumnName: "scrm_call.created_at", Asc: true},
		{ColumnName: "scrm_call.id", Asc: true},
	}, func(table model.Call) []any {
		return []any{table.CreatedAt, table.ID}
	})
	for {
		batchCall, err := next()
		if err != nil {
			scrm.Logger().WithContext(ctx).Error("database ", err.Error())
			return "", err
		}
		if len(batchCall) == 0 {
			break
		}
		batchCallPointers := database.Apply(batchCall, func(i model.Call) *model.Call {
			return &i
		})
		calls = append(calls, batchCallPointers...)
	}
	seatIdCollect := util.MakeCollectTint()
	for _, v := range calls {
		if v.SeatID > 0 {
			seatIdCollect.Add(v.SeatID)
		}
	}
	seatIDs := seatIdCollect.Export()
	seatMap := make(map[int]NameOfSeat, len(calls))
	if len(seatIDs) > 0 {
		db := scrm.GormDB.WithContext(ctx).
			Table("scrm_seat").
			Joins("left join sys_user on sys_user.user_id = scrm_seat.user_id").
			Select("scrm_seat.id,scrm_seat.nickname,sys_user.username").
			Where("scrm_seat.id IN ?", seatIDs).
			Scan(&ns)
		if db.Error != nil {
			scrm.Logger().WithContext(ctx).Error("get username and nickname err", db.Error)
		}
	}
	for _, v := range ns {
		seatMap[v.SeatID] = v
	}
	columns := []string{"序号", "通话编号", "工单编号", "电话号码", "项目编号", "创建时间",
		"通话记录", "模型标签", "通话标签", "坐席标签", "挂断标签", "备注", "呼出给客户时刻", "客户接起时刻",
		"转换时刻", "呼出给坐席时刻", "坐席接起时刻", "挂断时刻", "客户响铃时长", "坐席响铃时长",
		"机器人通话时长", "坐席通话时长", "客户等待转接时长", "综合通话时长", "坐席用户名", "坐席名", "线路"}
	excelBuf, err := util.MakeExcelFromData(
		callToSlice(calls, seatMap),
		columns,
	).WriteToBuffer()
	if err != nil {
		scrm.Logger().WithContext(ctx).Error("convert excelize.File to buffer: ", err.Error())
		return "", err
	}
	filename := strconv.Itoa(task.ID) + "-" + util.GetExcelFileName("通话记录")
	{
		_, err := scrm.MinIOClient.PutObject(ctx, config.ExtConfig.MinIO.ExportFileBucket, filename, excelBuf, -1, minio.PutObjectOptions{})
		if err != nil {
			scrm.Logger().WithContext(ctx).Error("minio save file: ", err.Error())
			return "", err
		}
	}
	return filename, nil
}

func (t ExportCallTask) GetTaskType() string {
	return "通话记录"
}
