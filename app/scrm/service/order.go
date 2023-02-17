package service

import (
	"context"
	"errors"
	"github.com/xuri/excelize/v2"
	"go-admin/app/scrm"
	"go-admin/app/scrm/model"
	"go-admin/common/actions"
	"go-admin/common/gormscope"
	common "go-admin/common/models"
	"go-admin/common/util"
	"gorm.io/gorm"
	"mime/multipart"
	"regexp"
	"time"
)

const (
	OrderStatusWaiting    = "等待中"
	OrderStatusProcessing = "处理中"
	OrderStatusFinished   = "已完成"
)

type UploadOrderGroupReq struct {
	ProjectID int
	DeptID    int
	Orders    []*model.Order
	File      multipart.File
	Filename  string

	common.ControlBy
}

type UploadOrderGroupResp struct{}

func UploadOrderGroupAndCreateOrders(ctx context.Context, req UploadOrderGroupReq) error {
	orders, err := ReadExcel(ctx, req.File)
	if err != nil {
		scrm.Logger().WithContext(ctx).Error(err.Error())
		return err
	}
	orderGroup := model.OrderGroup{
		Filename:  req.Filename,
		ProjectID: req.ProjectID,
		DeptID:    req.DeptID,
		Count:     len(orders),
		ControlBy: common.ControlBy{CreateBy: req.CreateBy},
	}
	return scrm.GormDB.Transaction(func(tx *gorm.DB) error {
		db := tx.WithContext(ctx).Create(&orderGroup)
		if err := db.Error; err != nil {
			scrm.Logger().WithContext(ctx).Error("create order group error", err.Error())
			return err
		}
		for _, v := range orders {
			v.CreateBy = req.CreateBy
			v.ProjectID = req.ProjectID
			v.DeptID = req.DeptID
			v.OrderGroupID = orderGroup.ID
			v.Status = OrderStatusWaiting
		}
		db = tx.WithContext(ctx).Create(&orders)
		if err := db.Error; err != nil {
			scrm.Logger().WithContext(ctx).Error("create order error", err.Error())
			return err
		}
		return nil
	})
}

type SearchOrderGroupReq struct {
	Start string `json:"start"`
	End   string `json:"end"`

	Pagination
}

type OrderGroupResponseItem struct {
	ID        int       `json:"id"`
	Filename  string    `json:"filename"`
	Dept      string    `json:"dept"`
	Project   string    `json:"project"`
	CreatedAt time.Time `json:"createdAt"`
}

func SearchOrderGroup(ctx context.Context, req SearchOrderGroupReq) ([]OrderGroupResponseItem, int64, error) {
	var (
		count   int64
		results []OrderGroupResponseItem
	)
	db := scrm.GormDB.WithContext(ctx).
		Table(model.OrderGroup{}.TableName()+" og").
		Select("og.id, og.filename, og.created_at, d.dept_name dept, p.name project").
		Joins("left join scrm_project p on p.id=og.project_id").
		Joins("left join sys_dept d on d.dept_id=og.dept_id").
		Scopes(
			actions.DeptPermission(ctx, "og"),
			gormscope.Paginate(&req.Pagination),
			gormscope.CreateDateRange(req.Start, req.End, "og"),
		).
		Order("id asc").
		Scan(&results)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("search order group error", err.Error())
		return nil, 0, err
	}
	db = db.Limit(-1).Offset(-1).Count(&count)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("count order group error", err.Error())
		return nil, 0, err
	}
	return results, count, nil
}

type GetOrderListReq struct {
	ProjectID int `json:"projectId"`

	Pagination
}

type OrderResponseItem struct {
	ID      int      `json:"id"`
	Code    string   `json:"code"`
	Sex     string   `json:"sex"`
	Phone   string   `json:"phone"`
	Status  string   `json:"status"`
	Calls   []string `json:"calls"`
	Project string   `json:"project"`
}

func GetOrderList(ctx context.Context, req GetOrderListReq) ([]OrderResponseItem, int64, error) {
	var (
		count int64
		items []OrderResponseItem
	)
	var orders []model.Order
	db := scrm.GormDB.WithContext(ctx).
		Model(&model.Order{}).
		Scopes(
			actions.DeptPermission(ctx, model.Order{}.TableName()),
			gormscope.Paginate(&req.Pagination),
		).
		Preload("Calls", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, order_id")
		}).Preload("Project").
		Order("id asc")
	if req.ProjectID > 0 {
		db = db.Where("project_id = ?", req.ProjectID)
	}
	db = db.Find(&orders)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("search order error", err.Error())
		return nil, 0, err
	}
	db = db.Limit(-1).Offset(-1).Count(&count)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("count order error", err.Error())
		return nil, 0, err
	}
	for _, order := range orders {
		var project string
		if order.Project != nil {
			project = order.Project.Name
		}
		items = append(items, OrderResponseItem{
			ID:      order.ID,
			Code:    order.Code,
			Sex:     order.Sex,
			Phone:   util.HidePhone(order.Phone),
			Status:  order.Status,
			Project: project,
			Calls:   util.Convert(order.Calls, func(s model.Call) string { return s.ID }),
		})
	}
	return items, count, nil
}

type GetOrderDetailReq struct {
	ID     int `form:"id"`
	DeptID int
}

type OrderDetail struct {
	ID      int        `json:"id"`
	Code    string     `json:"code"`
	Phone   string     `json:"phone"`
	Status  string     `json:"status"`
	Project string     `json:"project"`
	Calls   []CallItem `json:"calls"`
}

type CallItem struct {
	ID     string `json:"id"`
	Detail string `json:"detail"`
}

func GetOrderDetail(ctx context.Context, req GetOrderDetailReq) (OrderDetail, error) {
	var order model.Order
	err := scrm.GormDB.WithContext(ctx).
		Scopes(
			actions.DeptPermission(ctx, order.TableName()),
		).
		Preload("Calls").Preload("Project").
		First(&order, req.ID).Error
	if err != nil {
		scrm.Logger().WithContext(ctx).Error("get order error", err.Error())
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errors.New("查看对象不存在或无权查看")
		}
		return OrderDetail{}, err
	}
	var project string
	if order.Project != nil {
		project = order.Project.Name
	}
	return OrderDetail{
		ID:      order.ID,
		Code:    order.Code,
		Phone:   util.HidePhone(order.Phone),
		Status:  order.Status,
		Project: project,
		Calls: util.Convert(
			order.Calls,
			func(s model.Call) CallItem {
				return CallItem{ID: s.ID, Detail: s.Detail}
			}),
	}, nil
}

type DeleteOrderReq struct {
	ID int `form:"id"`
}

type DeleteOrderResp struct{}

func DeleteOrder(ctx context.Context, req DeleteOrderReq) error {
	db := scrm.GormDB.WithContext(ctx).
		Scopes(
			actions.DeptPermission(ctx, model.Order{}.TableName()),
		).Delete(&model.Order{}, req.ID)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error(ctx, "Error found in delete project", err.Error())
		return err
	}
	if db.RowsAffected == 0 {
		return errors.New("无权删除该数据")
	}
	return nil
}

func ReadExcel(ctx context.Context, file multipart.File) ([]*model.Order, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		scrm.Logger().WithContext(ctx).Error(err.Error())
		return nil, err
	}
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("表格式异常")
	}
	sheet := sheets[0]
	rows, err := f.GetRows(sheet)
	if err != nil {
		scrm.Logger().WithContext(context.Background()).Error(err.Error())
		return nil, err
	}
	orders := make([]*model.Order, 0)
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 3 {
			scrm.Logger().WithContext(context.Background()).Error("表单列数少于3列")
			return nil, errors.New("当前系统支持的格式是：第一行作为表头，第1列是编号，第2列是电话号码，第3列是性别")
		}
		//if !IsCode(row[0]) {
		//	scrm.Logger().Error("编号格式错误")
		//	errStr := fmt.Sprintf("您好，上传的编号格式有误，错误在第%d行第1列，请检查后上传", i+1)
		//	return nil, errors.New(errStr)
		//}
		//if !IsMobile(row[1]) {
		//	scrm.Logger().WithContext(context.Background()).Error("手机号码格式错误")
		//	errStr := fmt.Sprintf("您好，上传的电话号码格式有误，错误在第%d行第2列，请检查后上传", i+1)
		//	return nil, errors.New(errStr)
		//}
		//if row[2] != "男" && row[2] != "女" {
		//	scrm.Logger().Error("性别格式错误")
		//	errStr := fmt.Sprintf("您好，上传的性别格式有误，错误在第%d行第3列，请检查后上传", i+1)
		//	return nil, errors.New(errStr)
		//}
		orders = append(orders, &model.Order{
			Code:  row[0],
			Phone: row[1],
			Sex:   row[2],
		})
	}
	return orders, nil
}

func IsMobile(phone string) bool {
	regRuler := "^1[3456789]{1}\\d{9}$"
	reg := regexp.MustCompile(regRuler)
	return reg.MatchString(phone)
}

//func IsCode(code string) bool {
//	flag := 0
//	for _, v := range []rune(code) {
//		if !unicode.IsNumber(v) {
//			flag = 1
//			break
//		}
//	}
//	return flag == 0
//}

func ExportOrders(ctx context.Context, req GetOrderListReq) (ExportResp, error) {
	var orders []*model.Order
	db := scrm.GormDB.WithContext(ctx).
		Model(&model.Order{}).
		Scopes(
			actions.DeptPermission(ctx, model.Order{}.TableName()),
		).
		//Preload("Calls", func(db *gorm.DB) *gorm.DB {
		//	return db.Select("id, order_id")
		//}).
		Preload("Project")
	if req.ProjectID > 0 {
		db = db.Where("project_id = ?", req.ProjectID)
	}
	db = db.Find(&orders)
	if err := db.Error; err != nil {
		scrm.Logger().WithContext(ctx).Error("search order error", err.Error())
		return ExportResp{}, err
	}
	columns := []string{"序号", "工单编号", "电话号码", "性别", "状态"}
	data, filename, err := util.CreateExcelFile(
		orderToSlice(orders),
		columns,
		"工单",
	)
	if err != nil {
		return ExportResp{}, err
	}
	return ExportResp{data, filename}, nil
}

func orderToSlice(orders []*model.Order) [][]interface{} {
	var res [][]interface{}
	for index, order := range orders {
		s := []interface{}{
			index + 1,
			order.ID,
			util.HidePhone(order.Phone),
			order.Sex,
			order.Status,
		}
		res = append(res, s)
	}
	return res
}
