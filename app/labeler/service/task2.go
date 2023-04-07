package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go-admin/app/admin/models"
	"go-admin/app/labeler/model"
	"go-admin/common/dto"
	"go-admin/common/log"
	"go-admin/common/util"
)

type UploadTask2Req struct {
	Rows      [][]string
	ProjectID primitive.ObjectID
}

type UploadTask2Resp struct {
	UploadCount int `json:"uploadCount"`
}

func (svc *LabelerService) UploadTask2(ctx context.Context, req UploadTask2Req) (UploadTask2Resp, error) {
	var project model.Project2
	if err := svc.CollectionProject2.FindOne(ctx, bson.M{"_id": req.ProjectID}).Decode(&project); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return UploadTask2Resp{}, errors.New("项目不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask2Resp{}, err
	}
	if len(project.Schema.ContentTypes) == 0 {
		return UploadTask2Resp{}, errors.New("项目没有配置数据规则")
	}
	labels := make([]model.Task2LabelItem, len(project.Schema.Labels))
	for i, l := range project.Schema.Labels {
		labels[i] = model.Task2LabelItem{
			Name:  l.Name,
			Value: "",
		}
	}
	tasks := make([]any, len(req.Rows))
	now := util.Datetime(time.Now())
	for i, row := range req.Rows {
		row := util.DefaultSlice[string](row)
		contents := make([]model.Task2ContentItem, len(project.Schema.ContentTypes))
		for contentTypeIndex, contentType := range project.Schema.ContentTypes {
			contents[contentTypeIndex] = model.Task2ContentItem{
				Name:  contentType,
				Value: row.At(contentTypeIndex),
			}
		}
		tasks[i] = model.Task2{
			ID:          primitive.NewObjectID(),
			ProjectID:   req.ProjectID,
			Status:      model.TaskStatusAllocate,
			Permissions: model.Permissions{},
			UpdateTime:  now,
			Contents:    contents,
			Labels:      labels,
		}
	}
	result, err := svc.CollectionTask2.InsertMany(ctx, tasks)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask2Resp{}, err
	}
	return UploadTask2Resp{UploadCount: len(result.InsertedIDs)}, err
}

type SearchTask2Req = SearchTaskReq

type SearchTask2Resp struct {
	ID         primitive.ObjectID       `json:"id"`
	ProjectID  primitive.ObjectID       `json:"projectId"`
	Status     string                   `json:"status"`
	Labeler    string                   `json:"labeler"`
	Checker    string                   `json:"checker"`
	UpdateTime util.Datetime            `json:"updateTime"`
	Contents   []model.Task2ContentItem `json:"contents"`
	Labels     []model.Task2LabelItem   `json:"labels"`
}

func (svc *LabelerService) SearchTask2(ctx context.Context, req SearchTask2Req) ([]SearchTask2Resp, int, error) {
	filter, err := buildFilter(req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	cursor, err := svc.CollectionTask2.Find(ctx, filter, buildOptions(req))
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	var tasks []model.Task2
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	results := svc.tasksToSearchTask2Resp(ctx, tasks)

	count, err := svc.CollectionTask2.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	return results, int(count), nil
}

func (svc *LabelerService) tasksToSearchTask2Resp(ctx context.Context, tasks []model.Task2) []SearchTask2Resp {
	ids := make([]string, 0)
	for _, task := range tasks {
		if task.Permissions.Labeler != nil {
			ids = append(ids, task.Permissions.Labeler.ID)
		}
		if task.Permissions.Checker != nil {
			ids = append(ids, task.Permissions.Checker.ID)
		}
	}

	res := make([]SearchTask2Resp, len(tasks))

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
		var labeler, checker string
		if task.Permissions.Labeler != nil {
			labeler = userMap[task.Permissions.Labeler.ID]
		}
		if task.Permissions.Checker != nil {
			checker = userMap[task.Permissions.Checker.ID]
		}
		res[i] = SearchTask2Resp{
			ID:         task.ID,
			ProjectID:  task.ProjectID,
			Status:     task.Status,
			Labeler:    labeler,
			Checker:    checker,
			UpdateTime: task.UpdateTime,
			Contents:   task.Contents,
			Labels:     task.Labels,
		}

	}
	return res
}

type Task2BatchAllocLabelerReq struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Number    int64              `json:"number"`
	Persons   []string           `json:"persons"`
}

type Task2BatchAllocLabelerResp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) Task2BatchAllocLabeler(ctx context.Context, req Task2BatchAllocLabelerReq) (Task2BatchAllocLabelerResp, error) {
	filter := bson.M{
		"projectId": req.ProjectID,
		"permissions.labeler": bson.M{
			"$exists": false,
		},
	}
	count, err := svc.CollectionTask2.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Task2BatchAllocLabelerResp{}, err
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
		result, err := svc.CollectionTask2.Find(ctx, filter, opts)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task2BatchAllocLabelerResp{}, err
		}

		var tasks []model.Task2
		if err = result.All(ctx, &tasks); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task2BatchAllocLabelerResp{}, err
		}

		ft := bson.M{
			"_id": bson.M{
				"$in": util.Map(tasks, func(v model.Task2) primitive.ObjectID { return v.ID }),
			},
		}
		update := bson.M{
			"$set": bson.M{
				"permissions.labeler": model.Person{ID: fmt.Sprint(id)},
				"status":              model.TaskStatusLabeling,
				"updateTime":          util.Datetime(time.Now()),
			},
		}
		if _, err := svc.CollectionTask2.UpdateMany(ctx, ft, update); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task2BatchAllocLabelerResp{}, err
		}
	}

	return Task2BatchAllocLabelerResp{Count: count}, nil
}

type Task2BatchAllocCheckerReq struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Persons   []string           `json:"persons"`
	Number    int64              `json:"number"`
}

type Task2BatchAllocCheckerResp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) Task2BatchAllocChecker(ctx context.Context, req Task2BatchAllocCheckerReq) (Task2BatchAllocCheckerResp, error) {
	if req.Number <= 0 {
		return Task2BatchAllocCheckerResp{}, errors.New("分配任务数量不合法")
	}
	if len(req.Persons) == 0 {
		return Task2BatchAllocCheckerResp{}, errors.New("分配人员数量不能为0")
	}
	filter := bson.M{
		"projectId": req.ProjectID,
		"status":    model.TaskStatusSubmit,
		"permissions.labeler": bson.M{
			"$exists": true,
		},
		"permissions.checker": bson.M{
			"$exists": false,
		},
	}
	maxCount := int(req.Number) / len(req.Persons)
	if maxCount < 1 {
		maxCount = 1
	}
	personMap := make(map[string]int, len(req.Persons))
	for _, id := range req.Persons {
		personMap[id] = 0
	}

	result, err := svc.CollectionTask2.Find(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Task2BatchAllocCheckerResp{}, err
	}
	var tasks []model.Task2
	if err = result.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Task2BatchAllocCheckerResp{}, err
	}
	if len(tasks) == 0 {
		return Task2BatchAllocCheckerResp{}, errors.New("当前无可分配任务")
	}
	var totalCount int64
	for _, task := range tasks {
		var minCount = math.MaxInt
		var minID string
		for i, v := range personMap {
			if i == task.Permissions.Labeler.ID {
				continue
			}
			if v == maxCount {
				continue
			}
			if v < minCount {
				minCount = v
				minID = i
			}
		}
		if minID == "" {
			continue
		}
		personMap[minID]++
		ft := bson.M{
			"_id": task.ID,
		}
		update := bson.M{
			"$set": bson.M{
				"permissions.checker": model.Person{ID: minID},
				"status":              model.TaskStatusChecking,
				"updateTime":          util.Datetime(time.Now()),
			},
		}
		if _, err := svc.CollectionTask2.UpdateOne(ctx, ft, update); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task2BatchAllocCheckerResp{}, err
		}
		totalCount++
		if totalCount == req.Number {
			break
		}
	}
	if totalCount == 0 {
		return Task2BatchAllocCheckerResp{}, errors.New("分配失败：标注员和审核员不能是同一人")
	}
	return Task2BatchAllocCheckerResp{Count: totalCount}, nil
}

type ResetTasks2Req struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Persons   []string           `json:"persons"`
	Statuses  []string           `json:"statuses"`
}

type ResetTasks2Resp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) ResetTasks2(ctx context.Context, req ResetTasks2Req) (ResetTasks2Resp, error) {
	filter := bson.M{}
	if !req.ProjectID.IsZero() {
		filter["projectId"] = req.ProjectID
	}
	if len(req.Persons) > 0 {
		filter["$or"] = bson.A{
			bson.M{
				"permissions.labeler.id": bson.M{
					"$in": req.Persons,
				},
			},
			bson.M{
				"permissions.checker.id": bson.M{
					"$in": req.Persons,
				},
			},
		}
	}
	if len(req.Statuses) > 0 {
		filter["status"] = bson.M{
			"$in": req.Statuses,
		}
	}
	update := bson.M{
		"$set": bson.M{
			"permissions": model.Permissions{},
			"status":      model.TaskStatusAllocate,
			"updateTime":  util.Datetime(time.Now()),
		},
	}
	result, err := svc.CollectionTask2.UpdateMany(ctx, filter, update)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return ResetTasks2Resp{}, err
	}
	return ResetTasks2Resp{Count: result.ModifiedCount}, nil
}

type UpdateTask2Req struct {
	UserID        string                   `json:"-"`
	UserDataScope string                   `json:"-"`
	ID            primitive.ObjectID       `json:"id"`
	Contents      []model.Task2ContentItem `json:"contents"`
	Labels        []model.Task2LabelItem   `json:"labels"`
}

func (svc *LabelerService) UpdateTask2(ctx context.Context, req UpdateTask2Req) (model.Task2, error) {
	var task model.Task2
	if err := svc.CollectionTask2.FindOne(ctx, bson.M{"_id": req.ID}).Decode(&task); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Task2{}, errors.New("任务不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task2{}, err
	}
	if req.UserDataScope != "1" && req.UserDataScope != "2" && !task.Permissions.IsLabeler(req.UserID) && !task.Permissions.IsChecker(req.UserID) {
		return model.Task2{}, errors.New("权限不足")
	}
	task.Contents = req.Contents
	task.Labels = req.Labels
	task.UpdateTime = util.Datetime(time.Now())
	update := bson.M{
		"$set": bson.M{
			"contents":   task.Contents,
			"labels":     task.Labels,
			"updateTime": task.UpdateTime,
		},
	}
	if _, err := svc.CollectionTask2.UpdateByID(ctx, req.ID, update); err != nil {
		log.Logger().WithContext(ctx).Error("update task: ", err.Error())
		return model.Task2{}, err
	}
	return task, nil
}

type BatchSetTask2StatusReq struct {
	UserID        string               `json:"-"`
	UserDataScope string               `json:"-"`
	IDs           []primitive.ObjectID `json:"ids"`
	Status        string               `json:"status"`
}

type BatchSetTask2StatusResp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) BatchSetTask2Status(ctx context.Context, req BatchSetTask2StatusReq) (BatchSetTask2StatusResp, error) {
	if len(req.IDs) == 0 {
		return BatchSetTask2StatusResp{}, errors.New("什么也没有发生")
	}
	filter := bson.M{
		"_id": bson.M{
			"$in": req.IDs,
		},
	}
	if req.UserDataScope != "1" && req.UserDataScope != "2" {
		filter["$or"] = bson.A{
			bson.M{"permissions.labeler.id": req.UserID},
			bson.M{"permissions.checker.id": req.UserID},
		}
	}
	update := bson.M{
		"$set": bson.M{
			"status":     req.Status,
			"updateTime": util.Datetime(time.Now()),
		},
	}
	result, err := svc.CollectionTask2.UpdateMany(ctx, filter, update)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return BatchSetTask2StatusResp{}, err
	}
	if result.ModifiedCount == 0 {
		return BatchSetTask2StatusResp{}, errors.New("权限不足")
	}
	return BatchSetTask2StatusResp{Count: result.ModifiedCount}, err
}

type SearchMyTask2Req struct {
	ID       primitive.ObjectID `json:"id"`
	UserID   string
	TaskType string   `json:"taskType"`
	Status   []string `json:"status"`
	dto.Pagination
}

func (svc *LabelerService) SearchMyTask2(ctx context.Context, req SearchMyTask2Req) ([]SearchTask2Resp, int, error) {
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

	cursor, err := svc.CollectionTask2.Find(ctx, filter, buildOptions2(req))
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	var tasks []model.Task2
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	results := svc.tasksToSearchTask2Resp(ctx, tasks)

	count, err := svc.CollectionTask2.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	return results, int(count), nil
}

func buildOptions2(req SearchMyTask2Req) *options.FindOptions {
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

func (svc *LabelerService) DeleteTask2(ctx context.Context, id primitive.ObjectID) error {
	if _, err := svc.CollectionTask2.DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		log.Logger().WithContext(ctx).Error("delete task: ", err.Error())
		return err
	}
	return nil
}
