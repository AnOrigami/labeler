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

type UploadTask5Req struct {
	Tasks5    []model.Task5
	ProjectID primitive.ObjectID
	Name      []string
}

type UploadTask5Resp struct {
	UploadCount int `json:"uploadCount"`
}

func (svc *LabelerService) UploadTask5(ctx context.Context, req UploadTask5Req) (UploadTask5Resp, error) {
	var project5 model.Project5
	if err := svc.CollectionProject5.FindOne(ctx, bson.M{"_id": req.ProjectID}).Decode(&project5); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return UploadTask5Resp{}, errors.New("项目不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask5Resp{}, err
	}
	insertTasks := make([]any, len(req.Tasks5))

	for i, oneTask5 := range req.Tasks5 {
		for j, oneDialog := range oneTask5.Dialog {

			oneTask5.Dialog[j].NewAction = oneDialog.Actions
			oneTask5.Dialog[j].NewOutputs = oneDialog.ModelOutputs
			for k, entity := range oneDialog.Entities {
				numString := strconv.Itoa(entity.Num)
				oneTask5.Dialog[j].Entities[k].ClassType = entity.Class + numString + "[" + entity.Type + "]"
			}
			for i2, action := range oneDialog.Actions {
				if len(action.ActionObject) == 0 {
					oneTask5.Dialog[j].Actions[i2].ActionObject = append(oneTask5.Dialog[j].Actions[i2].ActionObject, model.Object{})
				}
			}
			//使用actions.action_object添加entity
			for _, action := range oneDialog.Actions {
				var insertOneEntity = model.EntityOption{}
				insertOneEntity = model.EntityOption{
					ObjectSummary: action.ActionObject[0].ObjectSummary,
					ClassType:     action.ActionObject[0].ObjectName,
				}
				//判断ObjectSummary是否为空，为空直接不添加
				if insertOneEntity.ObjectSummary != "" {
					oneTask5.Dialog[j].Entities = append(oneTask5.Dialog[j].Entities, insertOneEntity)
				}
			}

			uniqueEntities := make(map[string]model.EntityOption)
			for _, v := range oneTask5.Dialog[j].Entities {
				if existingEntity, ok := uniqueEntities[v.ObjectSummary]; ok {
					// 如果已存在相同ID的记录，则比较B字段的值
					if len(v.ClassType) < len(existingEntity.ClassType) {
						uniqueEntities[v.ObjectSummary] = v
					}
				} else {
					uniqueEntities[v.ObjectSummary] = v
				}
			}
			uniqueEntitiesArray := make([]model.EntityOption, 0, len(uniqueEntities))
			for _, v := range uniqueEntities {
				uniqueEntitiesArray = append(uniqueEntitiesArray, v)
			}
			oneTask5.Dialog[j].Entities = uniqueEntitiesArray
		}

		insertTasks[i] = model.Task5{
			ID:          primitive.NewObjectID(),
			Name:        req.Name[i],
			ProjectID:   req.ProjectID,
			Status:      model.TaskStatusAllocate,
			Permissions: model.Permissions{},
			UpdateTime:  util.Datetime(time.Now()),
			Dialog:      oneTask5.Dialog,
		}
	}
	result, err := svc.CollectionTask5.InsertMany(ctx, insertTasks)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask5Resp{}, err
	}
	return UploadTask5Resp{UploadCount: len(result.InsertedIDs)}, err
}

type SearchTask5Req = SearchTaskReq

type SearchTask5Resp struct {
	ID         primitive.ObjectID  `json:"id"`
	ProjectID  primitive.ObjectID  `json:"projectId"`
	Name       string              `json:"name"`
	Status     string              `json:"status"`
	Labeler    string              `json:"labeler"`
	Checker    string              `json:"checker"`
	UpdateTime util.Datetime       `json:"updateTime"`
	Dialog     []model.ContentText `json:"dialog"`
}

func (svc *LabelerService) SearchTask5(ctx context.Context, req SearchTask5Req) ([]SearchTask5Resp, int, error) {
	filter, err := buildFilter(req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	cursor, err := svc.CollectionTask5.Find(ctx, filter, buildOptions(req))
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	var tasks []model.Task5
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	results := svc.tasksToSearchTask5Resp(ctx, tasks)

	count, err := svc.CollectionTask5.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	return results, int(count), nil
}

func (svc *LabelerService) tasksToSearchTask5Resp(ctx context.Context, tasks []model.Task5) []SearchTask5Resp {
	ids := make([]string, 0)
	for _, task := range tasks {
		if task.Permissions.Labeler != nil {
			ids = append(ids, task.Permissions.Labeler.ID)
		}
	}

	res := make([]SearchTask5Resp, len(tasks))

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
		res[i] = SearchTask5Resp{
			ID:         task.ID,
			ProjectID:  task.ProjectID,
			Name:       task.Name,
			Status:     task.Status,
			Labeler:    labeler,
			UpdateTime: task.UpdateTime,
			Dialog:     task.Dialog,
		}

	}
	return res
}

type Task5BatchAllocLabelerReq struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Number    int64              `json:"number"`
	Persons   []string           `json:"persons"`
}

type Task5BatchAllocLabelerResp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) Task5BatchAllocLabeler(ctx context.Context, req Task5BatchAllocLabelerReq) (Task5BatchAllocLabelerResp, error) {
	filter := bson.M{
		"projectId": req.ProjectID,
		"status":    model.TaskStatusAllocate,
		"permissions.labeler": bson.M{
			"$exists": false,
		},
	}
	count, err := svc.CollectionTask5.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Task5BatchAllocLabelerResp{}, err
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
		result, err := svc.CollectionTask5.Find(ctx, filter, opts)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task5BatchAllocLabelerResp{}, err
		}

		var tasks []model.Task5
		if err = result.All(ctx, &tasks); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task5BatchAllocLabelerResp{}, err
		}

		ft := bson.M{
			"_id": bson.M{
				"$in": util.Map(tasks, func(v model.Task5) primitive.ObjectID { return v.ID }),
			},
		}
		update := bson.M{
			"$set": bson.M{
				"permissions.labeler": model.Person{ID: fmt.Sprint(id)},
				"status":              model.TaskStatusLabeling,
				"updateTime":          util.Datetime(time.Now()),
			},
		}
		if _, err := svc.CollectionTask5.UpdateMany(ctx, ft, update); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task5BatchAllocLabelerResp{}, err
		}
	}

	return Task5BatchAllocLabelerResp{Count: count}, nil
}

type ResetTasks5Req struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Persons   []string           `json:"persons"`
	Statuses  []string           `json:"statuses"`
	ResetType int64              `json:"resetType"`
}

type ResetTasks5Resp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) ResetTasks5(ctx context.Context, req ResetTasks5Req) (ResetTasks5Resp, error) {
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
		result, err := svc.CollectionTask5.UpdateMany(ctx, filter, update)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return ResetTasks5Resp{}, err
		}
		return ResetTasks5Resp{Count: result.ModifiedCount}, nil
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
		result, err := svc.CollectionTask5.UpdateMany(ctx, filter, update)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return ResetTasks5Resp{}, err
		}
		return ResetTasks5Resp{Count: result.ModifiedCount}, nil
	}
}

type UpdateTask5Req struct {
	UserID        string              `json:"-"`
	UserDataScope string              `json:"-"`
	ID            primitive.ObjectID  `json:"id"`
	Dialog        []model.ContentText `json:"dialog"`
}

func (svc *LabelerService) UpdateTask5(ctx context.Context, req UpdateTask5Req) (model.Task5, error) {
	var task model.Task5
	if err := svc.CollectionTask5.FindOne(ctx, bson.M{"_id": req.ID}).Decode(&task); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Task5{}, errors.New("任务不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task5{}, err
	}
	if req.UserDataScope != "1" && req.UserDataScope != "2" && !task.Permissions.IsLabeler(req.UserID) && !task.Permissions.IsChecker(req.UserID) {
		return model.Task5{}, errors.New("权限不足")
	}

	task.Dialog = req.Dialog
	task.UpdateTime = util.Datetime(time.Now())
	update := bson.M{
		"$set": bson.M{
			"dialog":     task.Dialog,
			"updateTime": task.UpdateTime,
		},
	}
	if _, err := svc.CollectionTask5.UpdateByID(ctx, req.ID, update); err != nil {
		log.Logger().WithContext(ctx).Error("update task: ", err.Error())
		return model.Task5{}, err
	}
	return task, nil
}

type BatchSetTask5StatusReq struct {
	UserID        string               `json:"-"`
	UserDataScope string               `json:"-"`
	IDs           []primitive.ObjectID `json:"ids"`
	Status        string               `json:"status"`
	WorkType      int64                `json:"workType"`
}

type BatchSetTask5StatusResp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) BatchSetTask5Status(ctx context.Context, req BatchSetTask5StatusReq) (BatchSetTask5StatusResp, error) {
	if len(req.IDs) == 0 {
		return BatchSetTask5StatusResp{}, errors.New("什么也没有发生")
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

	result, err := svc.CollectionTask5.UpdateMany(ctx, filter, update)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return BatchSetTask5StatusResp{}, err
	}
	if int(result.ModifiedCount) < len(req.IDs) {
		if req.Status == model.TaskStatusSubmit {
			return BatchSetTask5StatusResp{}, errors.New("提交失败：任务已被分配审核")
		}
		return BatchSetTask5StatusResp{}, errors.New("部分任务状态没有修改")
	}
	return BatchSetTask5StatusResp{Count: result.ModifiedCount}, err
}

type SearchMyTask5Req struct {
	ID       primitive.ObjectID `json:"id"`
	UserID   string
	Status   []string `json:"status"`
	TaskType string   `json:"taskType"`
	dto.Pagination
}

func (svc *LabelerService) SearchMyTask5(ctx context.Context, req SearchMyTask5Req) ([]SearchTask5Resp, int, error) {
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
	cursor, err := svc.CollectionTask5.Find(ctx, filter, buildOptions5(req))
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	var tasks []model.Task5
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	results := svc.tasksToSearchTask5Resp(ctx, tasks)

	count, err := svc.CollectionTask5.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	return results, int(count), nil
}

func buildOptions5(req SearchMyTask5Req) *options.FindOptions {
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

func (svc *LabelerService) DeleteTask5(ctx context.Context, id primitive.ObjectID) error {
	if _, err := svc.CollectionTask5.DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		log.Logger().WithContext(ctx).Error("delete task: ", err.Error())
		return err
	}
	return nil
}

type DownloadTask5Req struct {
	ProjectID       primitive.ObjectID `json:"projectId"`
	Status          []string           `json:"status"`
	UpdateTimeStart string             `json:"updateTimeStart"`
	UpdateTimeEnd   string             `json:"updateTimeEnd"`
}

type DownloadTask5Resp struct {
	File     *string `json:"file"`
	FileName string  `json:"filename"`
}

func (svc *LabelerService) DownloadTask5(ctx context.Context, req DownloadTask5Req) (DownloadTask5Resp, error) {
	filter := bson.M{
		"projectId": req.ProjectID,
		"status": bson.M{
			"$in": req.Status,
		},
	}

	if len(req.UpdateTimeStart) > 0 && len(req.UpdateTimeEnd) > 0 {
		startTime, err := time.Parse(util.TimeLayoutDatetime, req.UpdateTimeStart)
		if err != nil {
			return DownloadTask5Resp{}, ErrTimeParse
		}
		endTime, err := time.Parse(util.TimeLayoutDatetime, req.UpdateTimeEnd)
		if err != nil {
			return DownloadTask5Resp{}, ErrTimeParse
		}
		filter["updateTime"] = bson.M{
			"$gte": startTime,
			"$lte": endTime,
		}
	}

	cursor, err := svc.CollectionTask5.Find(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask5Resp{}, err
	}

	var tasks []*model.Task5
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask5Resp{}, err
	}

	nameStr := time.Now().Format("2006-01-02 15-04-05") + "下载文件.zip"
	buf := new(bytes.Buffer)

	zipWriter := zip.NewWriter(buf)

	for _, task := range tasks {
		data, err := json.Marshal(task)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTask5Resp{}, err
		}
		taskName := strings.Split(task.Name, ".")
		w1, err := zipWriter.Create(taskName[0] + ".json")
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTask5Resp{}, err
		}
		_, err = w1.Write(data)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTask5Resp{}, err
		}
	}
	if err = zipWriter.Close(); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask5Resp{}, err
	}

	res := base64.StdEncoding.EncodeToString(buf.Bytes())
	return DownloadTask5Resp{File: &res, FileName: nameStr}, nil
}

type GetTask5Req struct {
	ID              primitive.ObjectID `json:"id"`
	Name            string             `json:"name"`
	Labeler         string             `json:"labeler"`
	Checker         string             `json:"checker"`
	UpdateTimeStart string             `json:"updateTimeStart"`
	UpdateTimeEnd   string             `json:"updateTimeEnd"`
	Status          []string           `json:"status"`
	WorkType        int64              `json:"workType"`
}

type GetTask5Resp struct {
	Last primitive.ObjectID `json:"last"`
	Next primitive.ObjectID `json:"next"`
	model.Task5
}

func (svc *LabelerService) GetTask5(ctx context.Context, req GetTask5Req, p *actions.DataPermission) (GetTask5Resp, error) {
	var task model.Task5
	var users []models.SysUser
	filter := bson.M{}
	ids := make([]string, 0)
	userID := strconv.Itoa(p.UserId)
	if err := svc.CollectionTask5.FindOne(ctx, bson.D{{"_id", req.ID}}).Decode(&task); err != nil {
		if err == mongo.ErrNoDocuments {
			return GetTask5Resp{}, ErrNoDoc
		}
		log.Logger().WithContext(ctx).Error("get task: ", err.Error())
		return GetTask5Resp{}, err
	}
	if req.WorkType == 1 {
		if !task.Permissions.IsLabeler(userID) {
			return GetTask5Resp{}, errors.New("任务已被撤回/删除，请刷新任务列表重新进入")
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
			return GetTask5Resp{}, errors.New("任务已被撤回/删除，请刷新任务列表重新进入")
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
		filter = buildTask5DetailFilter(req)
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

	var nextTask model.Task5
	var lastTask model.Task5
	filter["_id"] = bson.M{"$gt": task.ID}
	if err := svc.CollectionTask5.FindOne(ctx, filter).Decode(&nextTask); err != nil {
		if err != mongo.ErrNoDocuments {
			log.Logger().WithContext(ctx).Error("get task: ", err.Error())
			return GetTask5Resp{}, err
		}
	}

	filter["_id"] = bson.M{"$lt": task.ID}
	option := options.FindOne().SetSort(bson.M{"_id": -1})
	if err := svc.CollectionTask5.FindOne(ctx, filter, option).Decode(&lastTask); err != nil {
		if err != mongo.ErrNoDocuments {
			log.Logger().WithContext(ctx).Error("get task: ", err.Error())
			return GetTask5Resp{}, err
		}
	}
	res := GetTask5Resp{
		Task5: task,
	}
	res.Last = lastTask.ID
	res.Next = nextTask.ID
	return res, nil
}

func buildTask5DetailFilter(req GetTask5Req) bson.M {
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

type SearchMyTask5CountReq struct {
	ID       primitive.ObjectID `json:"id"`
	UserID   string
	TaskType string `json:"taskType"`
}

type SearchMyTask5CountRes struct {
	Labeling int64 `json:"labeling"`
	Submit   int64 `json:"submit"`
	Checking int64 `json:"checking"`
	Passed   int64 `json:"passed"`
	Failed   int64 `json:"failed"`
}

func (svc *LabelerService) SearchMyTask5Count(ctx context.Context, req SearchMyTask5CountReq) (SearchMyTask5CountRes, error) {
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
	cursor, err := svc.CollectionTask5.Aggregate(ctx, pipe)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return SearchMyTask5CountRes{}, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return SearchMyTask5CountRes{}, err
	}

	var resp SearchMyTask5CountRes
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

type Node struct {
	Label    string `json:"label"`
	Value    string `json:"value"`
	Children []Node `json:"children"`
}

var ActionTags = []Node{
	{
		Label: "提问",
		Value: "1",
		Children: []Node{
			{
				Label: "短焦问句",
				Value: "10",
				Children: []Node{
					{
						Label:    "提问对来访者来说满分是什么样的",
						Value:    "22",
						Children: nil,
					},
					{
						Label:    "提问如果ta做出改变的话，来访者会有什么不同",
						Value:    "23",
						Children: nil,
					},
					{
						Label:    "以`结果问句`的方式进行提问",
						Value:    "24",
						Children: nil,
					},
					{
						Label:    "以`量尺问句`的方式进行提问",
						Value:    "25",
						Children: nil,
					},
					{
						Label:    "以`例外问句`的方式进行提问",
						Value:    "26",
						Children: nil,
					},
					{
						Label:    "以`奇迹问句`的方式进行提问",
						Value:    "27",
						Children: nil,
					},
					{
						Label:    "以`应对问句`的方式进行提问",
						Value:    "28",
						Children: nil,
					},
					{
						Label:    "以`关系问句`的方式进行提问",
						Value:    "29",
						Children: nil,
					},
				},
			},
			{
				Label: "通用问句",
				Value: "11",
				Children: []Node{
					{
						Label:    "提问来访者想先谈论哪个话题",
						Value:    "30",
						Children: nil,
					},
					{
						Label:    "提问描述具体事例",
						Value:    "31",
						Children: nil,
					},
					{
						Label:    "提问情绪的原因",
						Value:    "32",
						Children: nil,
					},
					{
						Label:    "提问来访者当下的情绪/感受",
						Value:    "33",
						Children: nil,
					},
					{
						Label:    "提问来访者对某事例的感受",
						Value:    "34",
						Children: nil,
					},
					{
						Label:    "提问来访者对某事例将采取的做法",
						Value:    "35",
						Children: nil,
					},
					{
						Label:    "提问来访者对某事例的想法",
						Value:    "36",
						Children: nil,
					},
					{
						Label:    "提问当下的事例对来访者的影响",
						Value:    "37",
						Children: nil,
					},
					{
						Label:    "提问某想法对来访者的影响",
						Value:    "38",
						Children: nil,
					},
					{
						Label:    "提问某行为对来访者的影响",
						Value:    "39",
						Children: nil,
					},
					{
						Label:    "提问身边资源有哪些",
						Value:    "40",
						Children: nil,
					},
					{
						Label:    "提问自身优势有哪些",
						Value:    "41",
						Children: nil,
					},
					{
						Label:    "提问来访者的期待",
						Value:    "42",
						Children: nil,
					},
					{
						Label:    "提问那我们可以一起探索一下。可以和我说说你最近的生活吗？",
						Value:    "43",
						Children: nil,
					},
					{
						Label:    "提问来访者需要什么帮助",
						Value:    "44",
						Children: nil,
					},
					{
						Label:    "追问模糊信息",
						Value:    "45",
						Children: nil,
					},
					{
						Label:    "追问还有吗？",
						Value:    "46",
						Children: nil,
					},
				},
			},
		},
	},
	{
		Label: "回应",
		Value: "2",
		Children: []Node{
			{
				Label:    "总结或重复",
				Value:    "12",
				Children: nil,
			},
			{
				Label:    "反馈",
				Value:    "13",
				Children: nil,
			},
			{
				Label:    "一般化",
				Value:    "14",
				Children: nil,
			},
			{
				Label:    "赞美",
				Value:    "15",
				Children: nil,
			},
			{
				Label:    "鼓励",
				Value:    "16",
				Children: nil,
			},
			{
				Label:    "安全岛技术",
				Value:    "17",
				Children: nil,
			},
		},
	},
	{
		Label:    "解释、分析",
		Value:    "3",
		Children: nil,
	},
	{
		Label: "提供思路、心理作业",
		Value: "4",
		Children: []Node{
			{
				Label: "思考类",
				Value: "18",
				Children: []Node{
					{
						Label:    "生命意义、人生价值",
						Value:    "47",
						Children: nil,
					},
					{
						Label:    "现实类的问题",
						Value:    "48",
						Children: nil,
					},
					{
						Label:    "过往经历",
						Value:    "49",
						Children: nil,
					},
				},
			},
			{
				Label: "书写类",
				Value: "19",
				Children: []Node{
					{
						Label:    "对未来",
						Value:    "50",
						Children: nil,
					},
					{
						Label:    "当下发生",
						Value:    "51",
						Children: nil,
					},
					{
						Label:    "过往经历",
						Value:    "52",
						Children: nil,
					},
				},
			},
			{
				Label: "行为类",
				Value: "20",
				Children: []Node{
					{
						Label:    "运动",
						Value:    "53",
						Children: nil,
					},
					{
						Label:    "音乐疗法",
						Value:    "54",
						Children: nil,
					},
					{
						Label: "绘画类",
						Value: "55",
						Children: []Node{
							{
								Label:    "画画",
								Value:    "62",
								Children: nil,
							},
							{
								Label:    "家庭关系图",
								Value:    "63",
								Children: nil,
							},
							{
								Label:    "其他",
								Value:    "64",
								Children: nil,
							},
						},
					},
					{
						Label: "情绪宣泄",
						Value: "56",
						Children: []Node{
							{
								Label:    "激烈运动",
								Value:    "65",
								Children: nil,
							},
							{
								Label:    "呐喊",
								Value:    "66",
								Children: nil,
							},
							{
								Label:    "蹦迪",
								Value:    "67",
								Children: nil,
							},
							{
								Label:    "极限运动",
								Value:    "68",
								Children: nil,
							},
							{
								Label:    "唱歌",
								Value:    "69",
								Children: nil,
							},
							{
								Label:    "撕纸",
								Value:    "70",
								Children: nil,
							},
							{
								Label:    "其他",
								Value:    "71",
								Children: nil,
							},
						},
					},
					{
						Label: "身心疗愈",
						Value: "57",
						Children: []Node{
							{
								Label:    "正念",
								Value:    "72",
								Children: nil,
							},
							{
								Label:    "冥想",
								Value:    "73",
								Children: nil,
							},
							{
								Label:    "催眠",
								Value:    "74",
								Children: nil,
							},
							{
								Label:    "呼吸",
								Value:    "75",
								Children: nil,
							},
							{
								Label:    "肌肉放松",
								Value:    "76",
								Children: nil,
							},
						},
					},
					{
						Label: "艺术疗愈",
						Value: "58",
						Children: []Node{
							{
								Label:    "阅读",
								Value:    "77",
								Children: nil,
							},
							{
								Label:    "书法",
								Value:    "78",
								Children: nil,
							},
							{
								Label:    "茶道",
								Value:    "79",
								Children: nil,
							},
							{
								Label:    "花道",
								Value:    "80",
								Children: nil,
							},
							{
								Label:    "香道",
								Value:    "81",
								Children: nil,
							},
							{
								Label:    "陶艺",
								Value:    "82",
								Children: nil,
							},
						},
					},
					{
						Label:    "自我暗示",
						Value:    "59",
						Children: nil,
					},
					{
						Label: "寻求他人帮助",
						Value: "60",
						Children: []Node{
							{
								Label:    "亲密关系支持",
								Value:    "83",
								Children: nil,
							},
							{
								Label:    "兴趣爱好小组",
								Value:    "84",
								Children: nil,
							},
							{
								Label:    "专业性支持",
								Value:    "85",
								Children: nil,
							},
							{
								Label:    "其他社会性支持",
								Value:    "86",
								Children: nil,
							},
						},
					},
					{
						Label:    "模拟练习",
						Value:    "61",
						Children: nil,
					},
				},
			},
			{
				Label:    "其他",
				Value:    "21",
				Children: nil,
			},
		},
	},
	{
		Label:    "心理科普",
		Value:    "5",
		Children: nil,
	},
	{
		Label:    "引导回主题",
		Value:    "6",
		Children: nil,
	},
	{
		Label:    "确认主题",
		Value:    "7",
		Children: nil,
	},
	{
		Label:    "提示如果存在安全风险，请联系线下相关机构或组织。保护自身及他人人身财产安全",
		Value:    "8",
		Children: nil,
	},
	{
		Label:    "其他",
		Value:    "9",
		Children: nil,
	},
}
