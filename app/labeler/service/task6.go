package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go-admin/app/admin/models"
	"go-admin/app/labeler/model"
	"go-admin/common/actions"
	"go-admin/common/dto"
	"go-admin/common/log"
	"go-admin/common/util"
)

type UploadTask6Req struct {
	Tasks6    []model.Task6
	ProjectID primitive.ObjectID
	Name      []string
}

type UploadTask6Resp struct {
	UploadCount int `json:"uploadCount"`
}

func (svc *LabelerService) UploadTask6(ctx context.Context, req UploadTask6Req) (UploadTask6Resp, error) {
	var project6 model.Project6
	if err := svc.CollectionProject6.FindOne(ctx, bson.M{"_id": req.ProjectID}).Decode(&project6); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return UploadTask6Resp{}, errors.New("项目不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask6Resp{}, err
	}
	var folder6 model.Folder
	if err := svc.CollectionFolder6.FindOne(ctx, bson.M{"_id": project6.FolderID}).Decode(&folder6); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return UploadTask6Resp{}, errors.New("文件夹不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask6Resp{}, err
	}
	insertTasks := make([]any, len(req.Tasks6))

	for i, oneTask6 := range req.Tasks6 {
		insertTasks[i] = model.Task6{
			ID:          primitive.NewObjectID(),
			Name:        req.Name[i],
			FullName:    folder6.Name + "/" + project6.Name + "/" + req.Name[i],
			ProjectID:   req.ProjectID,
			Status:      model.TaskStatusAllocate,
			Permissions: model.Permissions{},
			UpdateTime:  util.Datetime(time.Now()),
			Rpg:         oneTask6.Rpg,
		}
	}
	result, err := svc.CollectionTask6.InsertMany(ctx, insertTasks)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask6Resp{}, err
	}
	return UploadTask6Resp{UploadCount: len(result.InsertedIDs)}, err
}

type SearchTask6Req = SearchTaskReq

type SearchTask6Resp struct {
	ID         primitive.ObjectID `json:"id"`
	ProjectID  primitive.ObjectID `json:"projectId"`
	Name       string             `json:"name"`
	Status     string             `json:"status"`
	Labeler    string             `json:"labeler"`
	Checker    string             `json:"checker"`
	UpdateTime util.Datetime      `json:"updateTime"`
	Rpg        bson.M             `json:"rpg"`
}

func (svc *LabelerService) SearchTask6(ctx context.Context, req SearchTask6Req) ([]SearchTask6Resp, int, error) {
	filter, err := buildFilter(req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	cursor, err := svc.CollectionTask6.Find(ctx, filter, buildOptions(req))
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	var tasks []model.Task6
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	results := svc.tasksToSearchTask6Resp(ctx, tasks)

	count, err := svc.CollectionTask6.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	return results, int(count), nil
}

func (svc *LabelerService) tasksToSearchTask6Resp(ctx context.Context, tasks []model.Task6) []SearchTask6Resp {
	ids := make([]string, 0)
	for _, task := range tasks {
		if task.Permissions.Labeler != nil {
			ids = append(ids, task.Permissions.Labeler.ID)
		}
	}

	res := make([]SearchTask6Resp, len(tasks))

	var users []models.SysUser
	if len(ids) > 0 {
		db := svc.GormDB.WithContext(ctx).Find(&users).Select("user_id, nick_name").Where("user_id in ?", ids)
		if err := db.Error; err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
		}
	}
	userMap := make(map[string]string)
	for _, v := range users {
		userMap[strconv.Itoa(v.UserId)] = v.NickName
	}
	for i, task := range tasks {
		var labeler string
		if task.Permissions.Labeler != nil {
			labeler = userMap[task.Permissions.Labeler.ID]
		}
		res[i] = SearchTask6Resp{
			ID:         task.ID,
			ProjectID:  task.ProjectID,
			Name:       task.Name,
			Status:     task.Status,
			Labeler:    labeler,
			UpdateTime: task.UpdateTime,
			Rpg:        task.Rpg,
		}

	}
	return res
}

type Task6BatchAllocLabelerReq struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Number    int64              `json:"number"`
	Persons   []string           `json:"persons"`
}

type Task6BatchAllocLabelerResp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) Task6BatchAllocLabeler(ctx context.Context, req Task6BatchAllocLabelerReq) (Task6BatchAllocLabelerResp, error) {
	filter := bson.M{
		"projectId": req.ProjectID,
		"status":    model.TaskStatusAllocate,
		"permissions.labeler": bson.M{
			"$exists": false,
		},
	}
	count, err := svc.CollectionTask6.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Task6BatchAllocLabelerResp{}, err
	}

	if count > req.Number {
		count = req.Number
	}
	maxCount := count / int64(len(req.Persons))
	if maxCount < 1 {
		maxCount = 1
	}
	for _, id := range req.Persons {
		opts := options.Find().SetProjection(bson.D{{"_id", 1}}).SetLimit(maxCount)
		result, err := svc.CollectionTask6.Find(ctx, filter, opts)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task6BatchAllocLabelerResp{}, err
		}

		var tasks []model.Task6
		if err = result.All(ctx, &tasks); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task6BatchAllocLabelerResp{}, err
		}

		ft := bson.M{
			"_id": bson.M{
				"$in": util.Map(tasks, func(v model.Task6) primitive.ObjectID { return v.ID }),
			},
		}
		update := bson.M{
			"$set": bson.M{
				"permissions.labeler": model.Person{ID: fmt.Sprint(id)},
				"status":              model.TaskStatusLabeling,
				"updateTime":          util.Datetime(time.Now()),
			},
		}
		if _, err := svc.CollectionTask6.UpdateMany(ctx, ft, update); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task6BatchAllocLabelerResp{}, err
		}
	}

	return Task6BatchAllocLabelerResp{Count: count}, nil
}

type ResetTasks6Req struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Persons   []string           `json:"persons"`
	Statuses  []string           `json:"statuses"`
	ResetType int64              `json:"resetType"`
}

type ResetTasks6Resp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) ResetTasks6(ctx context.Context, req ResetTasks6Req) (ResetTasks6Resp, error) {
	filter := bson.M{}
	filter["projectId"] = req.ProjectID
	filter["status"] = bson.M{
		"$in": req.Statuses,
	}

	if req.ResetType == 0 {
		filter["permissions.labeler.id"] = bson.M{
			"$in": req.Persons,
		}
		update := bson.M{
			"$set": bson.M{
				"permissions": model.Permissions{},
				"status":      model.TaskStatusAllocate,
				"updateTime":  util.Datetime(time.Now()),
			},
		}
		result, err := svc.CollectionTask6.UpdateMany(ctx, filter, update)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return ResetTasks6Resp{}, err
		}
		return ResetTasks6Resp{Count: result.ModifiedCount}, nil
	} else {
		filter["permissions.checker.id"] = bson.M{
			"$in": req.Persons,
		}
		update := bson.M{
			"$set": bson.M{
				"permissions.checker": model.Person{},
				"status":              model.TaskStatusSubmit,
				"updateTime":          util.Datetime(time.Now()),
			},
		}
		result, err := svc.CollectionTask6.UpdateMany(ctx, filter, update)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return ResetTasks6Resp{}, err
		}
		return ResetTasks6Resp{Count: result.ModifiedCount}, nil
	}
}

type UpdateTask6Req struct {
	UserID        string             `json:"-"`
	UserDataScope string             `json:"-"`
	ID            primitive.ObjectID `json:"id"`
	Rpg           bson.M             `json:"rpg"`
	Version       int                `json:"version"`
}

func (svc *LabelerService) UpdateTask6(ctx context.Context, req UpdateTask6Req) (model.Task6, error) {
	var task model.Task6
	if err := svc.CollectionTask6.FindOne(ctx, bson.M{"_id": req.ID}).Decode(&task); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Task6{}, errors.New("任务不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task6{}, err
	}
	if req.UserDataScope != "1" && req.UserDataScope != "2" && !task.Permissions.IsLabeler(req.UserID) && !task.Permissions.IsChecker(req.UserID) {
		return model.Task6{}, errors.New("权限不足")
	}

	task.Rpg = req.Rpg
	task.UpdateTime = util.Datetime(time.Now())
	update := bson.M{
		"$set": bson.M{
			"rpg":        task.Rpg,
			"updateTime": task.UpdateTime,
			"version":    req.Version,
		},
	}
	fiter := bson.M{
		"_id":     req.ID,
		"version": req.Version - 1,
	}
	updateResult, err := svc.CollectionTask6.UpdateOne(ctx, fiter, update)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task6{}, err
	}

	if updateResult.MatchedCount == 0 {
		log.Logger().WithContext(ctx).Error("查询的文档不存在或版本过旧,请刷新重试", err.Error())
		return model.Task6{}, errors.New("查询的文档不存在或版本过旧,请刷新重试")
	} else {
		return model.Task6{}, nil
	}
}

type BatchSetTask6StatusReq struct {
	UserID        string               `json:"-"`
	UserDataScope string               `json:"-"`
	IDs           []primitive.ObjectID `json:"ids"`
	Status        string               `json:"status"`
	WorkType      int64                `json:"workType"`
}

type BatchSetTask6StatusResp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) BatchSetTask6Status(ctx context.Context, req BatchSetTask6StatusReq) (BatchSetTask6StatusResp, error) {
	if len(req.IDs) == 0 {
		return BatchSetTask6StatusResp{}, errors.New("什么也没有发生")
	}
	filter := bson.M{
		"_id": bson.M{
			"$in": req.IDs,
		},
	}
	if req.WorkType != 0 {
		filter["$or"] = bson.A{
			bson.M{"permissions.labeler.id": req.UserID},
			bson.M{"permissions.checker.id": req.UserID},
		}
	}
	normalStatusMap := map[string][]string{
		//model.TaskStatusFailed:   {model.TaskStatusChecking, model.TaskStatusPassed, model.TaskStatusFailed},
		//model.TaskStatusPassed:   {model.TaskStatusChecking, model.TaskStatusPassed, model.TaskStatusFailed},
		//model.TaskStatusChecking: {model.TaskStatusSubmit, model.TaskStatusFailed},
		model.TaskStatusSubmit: {model.TaskStatusLabeling, model.TaskStatusSubmit /*, model.TaskStatusFailed*/},
	}
	specialStatusMap := map[string][]string{
		//model.TaskStatusFailed:   {model.TaskStatusChecking, model.TaskStatusPassed, model.TaskStatusSubmit},
		//model.TaskStatusPassed:   {model.TaskStatusChecking, model.TaskStatusPassed, model.TaskStatusSubmit},
		//model.TaskStatusChecking: {model.TaskStatusFailed},
		model.TaskStatusSubmit: {model.TaskStatusLabeling, model.TaskStatusAllocate, model.TaskStatusSubmit /*, model.TaskStatusFailed*/},
	}

	//任务状态为{未分配}，管理员点击进入之后为标注页面，点击提交之后任务状态变更为已提交
	//
	//任务状态为{待标注}，管理员点击进入之后为标注页面，点击提交之后任务状态变更为已提交
	//
	//任务状态为{审核不通过}，管理员点击进入之后为标注页面，点击提交之后任务状态变更为待审核
	//
	//任务状态为{已提交}，管理员点击进入之后为审核页面，点击审核通过之后任务状态变更为已审核，点击审核不通过之后任务状态变更为审核不通过
	//
	//任务状态为{待审核}，管理员点击进入之后为审核页面，点击审核通过之后任务状态变更为已审核，点击审核不通过之后任务状态变更为审核不通过
	//
	//任务状态为{已审核}，管理员点击进入之后为审核页面，点击审核通过之后任务状态变更为已审核，点击审核不通过之后任务状态变更为审核不通过
	update := bson.M{
		"$set": bson.M{
			"status":     req.Status,
			"updateTime": util.Datetime(time.Now()),
		},
	}
	if req.WorkType != 0 {
		if validSourceStatus := normalStatusMap[req.Status]; validSourceStatus != nil {
			filter["status"] = bson.M{
				"$in": validSourceStatus,
			}
		}
	} else {
		if validSourceStatus := specialStatusMap[req.Status]; validSourceStatus != nil {
			filter["status"] = bson.M{
				"$in": validSourceStatus,
			}
		}
	}

	result, err := svc.CollectionTask6.UpdateMany(ctx, filter, update)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return BatchSetTask6StatusResp{}, err
	}
	if int(result.ModifiedCount) < len(req.IDs) {
		if req.Status == model.TaskStatusSubmit {
			return BatchSetTask6StatusResp{}, errors.New("提交失败：任务已被分配审核")
		}
		return BatchSetTask6StatusResp{}, errors.New("部分任务状态没有修改")
	}
	return BatchSetTask6StatusResp{Count: result.ModifiedCount}, err
}

type SearchMyTask6Req struct {
	ID       primitive.ObjectID `json:"id"`
	UserID   string
	Status   []string `json:"status"`
	TaskType string   `json:"taskType"`
	dto.Pagination
}

func (svc *LabelerService) SearchMyTask6(ctx context.Context, req SearchMyTask6Req) ([]SearchTask6Resp, int, error) {
	filter := bson.M{
		"projectId": req.ID,
	}
	if len(req.Status) != 0 {
		filter["status"] = bson.M{
			"$in": req.Status,
		}
	}
	if req.TaskType == "标注" {
		filter["permissions.labeler.id"] = req.UserID
	} else if req.TaskType == "审核" {
		filter["permissions.checker.id"] = req.UserID
	} else {
		filter["$or"] = []bson.M{
			{"permissions.labeler.id": req.UserID},
			{"permissions.checker.id": req.UserID},
		}
	}
	cursor, err := svc.CollectionTask6.Find(ctx, filter, buildOptions6(req))
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	var tasks []model.Task6
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	results := svc.tasksToSearchTask6Resp(ctx, tasks)

	count, err := svc.CollectionTask6.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	return results, int(count), nil
}

func buildOptions6(req SearchMyTask6Req) *options.FindOptions {
	if req.PageIndex < 1 {
		req.PageIndex = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 10
	}
	opts := options.Find().
		SetLimit(int64(req.PageSize)).
		SetSkip(int64((req.PageIndex - 1) * req.PageSize))
	return opts
}

func (svc *LabelerService) DeleteTask6(ctx context.Context, id primitive.ObjectID) error {
	if _, err := svc.CollectionTask6.DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		log.Logger().WithContext(ctx).Error("delete task: ", err.Error())
		return err
	}
	return nil
}

type DownloadTask6Req struct {
	ProjectID       primitive.ObjectID `json:"projectId"`
	Status          []string           `json:"status"`
	UpdateTimeStart string             `json:"updateTimeStart"`
	UpdateTimeEnd   string             `json:"updateTimeEnd"`
}

type DownloadTask6Resp struct {
	File     *string `json:"file"`
	FileName string  `json:"filename"`
}

func (svc *LabelerService) DownloadTask6(ctx context.Context, req DownloadTask6Req) (DownloadTask6Resp, error) {
	filter := bson.M{
		"projectId": req.ProjectID,
		"status": bson.M{
			"$in": req.Status,
		},
	}

	if len(req.UpdateTimeStart) > 0 && len(req.UpdateTimeEnd) > 0 {
		startTime, err := time.Parse(util.TimeLayoutDatetime, req.UpdateTimeStart)
		if err != nil {
			return DownloadTask6Resp{}, ErrTimeParse
		}
		endTime, err := time.Parse(util.TimeLayoutDatetime, req.UpdateTimeEnd)
		if err != nil {
			return DownloadTask6Resp{}, ErrTimeParse
		}
		filter["updateTime"] = bson.M{
			"$gte": startTime,
			"$lte": endTime,
		}
	}

	cursor, err := svc.CollectionTask6.Find(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask6Resp{}, err
	}

	var tasks []*model.Task6
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask6Resp{}, err
	}

	nameStr := time.Now().Format("2006-01-02 15-04-05") + "下载文件.zip"
	buf := new(bytes.Buffer)

	zipWriter := zip.NewWriter(buf)

	for _, task := range tasks {
		data, err := json.Marshal(task)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTask6Resp{}, err
		}
		taskName := strings.Split(task.Name, ".")
		w1, err := zipWriter.Create(taskName[0] + ".json")
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTask6Resp{}, err
		}
		_, err = w1.Write(data)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTask6Resp{}, err
		}
	}
	if err = zipWriter.Close(); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask6Resp{}, err
	}

	res := base64.StdEncoding.EncodeToString(buf.Bytes())
	return DownloadTask6Resp{File: &res, FileName: nameStr}, nil
}

type GetTask6Req struct {
	ID              primitive.ObjectID `json:"id"`
	Name            string             `json:"name"`
	Labeler         string             `json:"labeler"`
	Checker         string             `json:"checker"`
	UpdateTimeStart string             `json:"updateTimeStart"`
	UpdateTimeEnd   string             `json:"updateTimeEnd"`
	Status          []string           `json:"status"`
	WorkType        int64              `json:"workType"`
}

type GetTask6Resp struct {
	Last primitive.ObjectID `json:"last"`
	Next primitive.ObjectID `json:"next"`
	model.Task6
}

func (svc *LabelerService) GetTask6(ctx context.Context, req GetTask6Req, p *actions.DataPermission) (GetTask6Resp, error) {
	var task model.Task6
	var users []models.SysUser
	filter := bson.M{}
	ids := make([]string, 0)
	userID := strconv.Itoa(p.UserId)
	if err := svc.CollectionTask6.FindOne(ctx, bson.D{{"_id", req.ID}}).Decode(&task); err != nil {
		if err == mongo.ErrNoDocuments {
			return GetTask6Resp{}, ErrNoDoc
		}
		log.Logger().WithContext(ctx).Error("get task: ", err.Error())
		return GetTask6Resp{}, err
	}
	if req.WorkType == 1 {
		if !task.Permissions.IsLabeler(userID) {
			return GetTask6Resp{}, errors.New("任务已被撤回/删除，请刷新任务列表重新进入")
		}
		filter = bson.M{
			"projectId":              task.ProjectID,
			"permissions.labeler.id": userID,
			"status": bson.M{
				"$in": req.Status,
			},
		}
	}
	if req.WorkType == 2 {
		if !task.Permissions.IsChecker(userID) {
			return GetTask6Resp{}, errors.New("任务已被撤回/删除，请刷新任务列表重新进入")
		}
		filter = bson.M{
			"projectId":              task.ProjectID,
			"permissions.checker.id": userID,
			"status": bson.M{
				"$in": req.Status,
			},
		}
	}
	if req.WorkType == 0 {
		filter = buildTask6DetailFilter(req)
		filter["projectId"] = task.ProjectID
	}
	if task.Permissions.Labeler != nil {
		ids = append(ids, task.Permissions.Labeler.ID)
	}
	if task.Permissions.Checker != nil {
		ids = append(ids, task.Permissions.Checker.ID)
	}
	if len(ids) > 0 {
		db := svc.GormDB.WithContext(ctx).Find(&users).Select("user_id, nick_name").Where("user_id in ?", ids)
		if err := db.Error; err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
		}
	}
	userMap := make(map[string]string)
	for _, v := range users {
		userMap[strconv.Itoa(v.UserId)] = v.NickName
	}
	if task.Permissions.Labeler != nil {
		task.Permissions.Labeler.NickName = userMap[task.Permissions.Labeler.ID]
	}
	if task.Permissions.Checker != nil {
		task.Permissions.Checker.NickName = userMap[task.Permissions.Checker.ID]
	}

	var nextTask model.Task6
	var lastTask model.Task6
	filter["_id"] = bson.M{"$gt": task.ID}
	if err := svc.CollectionTask6.FindOne(ctx, filter).Decode(&nextTask); err != nil {
		if err != mongo.ErrNoDocuments {
			log.Logger().WithContext(ctx).Error("get task: ", err.Error())
			return GetTask6Resp{}, err
		}
	}

	filter["_id"] = bson.M{"$lt": task.ID}
	option := options.FindOne().SetSort(bson.M{"_id": -1})
	if err := svc.CollectionTask6.FindOne(ctx, filter, option).Decode(&lastTask); err != nil {
		if err != mongo.ErrNoDocuments {
			log.Logger().WithContext(ctx).Error("get task: ", err.Error())
			return GetTask6Resp{}, err
		}
	}
	var project6 model.Project6
	if err := svc.CollectionProject6.FindOne(ctx, bson.M{"_id": task.ProjectID}).Decode(&project6); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return GetTask6Resp{}, errors.New("项目不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return GetTask6Resp{}, err
	}
	var folder6 model.Folder
	if err := svc.CollectionFolder6.FindOne(ctx, bson.M{"_id": project6.FolderID}).Decode(&folder6); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return GetTask6Resp{}, errors.New("文件夹不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return GetTask6Resp{}, err
	}
	task.FullName = folder6.Name + "/" + project6.Name + "/" + task.Name
	res := GetTask6Resp{
		Task6: task,
	}
	res.Last = lastTask.ID
	res.Next = nextTask.ID
	return res, nil
}

func buildTask6DetailFilter(req GetTask6Req) bson.M {
	filter := bson.M{}

	if len(req.Status) > 0 {
		filter["status"] = bson.M{
			"$in": req.Status,
		}
	}

	if len(req.Labeler) > 0 {
		filter["permissions.labeler.id"] = req.Labeler
	}
	if len(req.Checker) > 0 {
		filter["permissions.checker.id"] = req.Checker
	}
	if len(req.Name) > 0 {
		filter["name"] = bson.M{
			"$regex": req.Name,
		}
	}
	if len(req.UpdateTimeStart) > 0 {
		t, err := time.Parse(util.TimeLayoutDatetime, req.UpdateTimeStart)
		if err != nil {
			return nil
		}
		filter["updateTime"] = bson.M{
			"$gte": t,
		}
	}
	if len(req.UpdateTimeEnd) > 0 {
		t, err := time.Parse(util.TimeLayoutDatetime, req.UpdateTimeEnd)
		if err != nil {
			return nil
		}
		value, ok := filter["updateTime"]
		if ok {
			value.(bson.M)["$lte"] = t
			filter["updateTime"] = value
		}
	}
	return filter
}

type SearchMyTask6CountReq struct {
	ID       primitive.ObjectID `json:"id"`
	UserID   string
	TaskType string `json:"taskType"`
}

type SearchMyTask6CountRes struct {
	Labeling int64 `json:"labeling"`
	Submit   int64 `json:"submit"`
	Checking int64 `json:"checking"`
	Passed   int64 `json:"passed"`
	Failed   int64 `json:"failed"`
}

func (svc *LabelerService) SearchMyTask6Count(ctx context.Context, req SearchMyTask6CountReq) (SearchMyTask6CountRes, error) {
	filter := bson.M{
		"projectId": req.ID,
	}
	if req.TaskType == "标注" {
		filter["permissions.labeler.id"] = req.UserID
	} else {
		filter["permissions.checker.id"] = req.UserID
	}
	pipe := mongo.Pipeline{
		bson.D{
			{
				"$match",
				filter,
			},
		},
		bson.D{
			{
				"$group",
				bson.D{
					{"_id", "$status"},
					{"count", bson.D{{"$sum", 1}}},
				},
			},
		},
	}
	cursor, err := svc.CollectionTask6.Aggregate(ctx, pipe)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return SearchMyTask6CountRes{}, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return SearchMyTask6CountRes{}, err
	}

	var resp SearchMyTask6CountRes
	for _, result := range results {
		count := int64(result["count"].(int32))
		switch result["_id"] {
		case model.TaskStatusLabeling:
			resp.Labeling = count
		case model.TaskStatusSubmit:
			resp.Submit = count
		case model.TaskStatusChecking:
			resp.Checking = count
		case model.TaskStatusPassed:
			resp.Passed = count
		case model.TaskStatusFailed:
			resp.Failed = count
		}
	}
	return resp, nil
}
