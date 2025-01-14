package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
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

const (
	PermissionTypeLabeler = "标注"
	PermissionTypeChecker = "审核"
)

type UploadTaskResp struct {
	UploadCount int `json:"uploadCount"`
}

func (svc *LabelerService) UploadTask(ctx context.Context, req []model.Task) (UploadTaskResp, error) {
	data := make([]interface{}, len(req))
	for i, task := range req {
		data[i] = task
	}
	result, err := svc.CollectionTask.InsertMany(ctx, data)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTaskResp{}, err
	}
	return UploadTaskResp{UploadCount: len(result.InsertedIDs)}, err
}

func (svc *LabelerService) LabelTask(ctx context.Context, req model.Task, userID int) (model.Task, error) {
	task, err := svc.GetTask(ctx, req.ID)
	if err != nil {
		return model.Task{}, err
	}
	if req.Permissions.Labeler.ID != strconv.Itoa(userID) {
		return model.Task{}, errors.New("无权限修改")
	}
	if !TaskStatusCheck(true, task.Status, req.Status) {
		return model.Task{}, errors.New("当前状态无法修改")
	}

	data := bson.M{
		"$set": bson.M{
			"contents":   req.Contents,
			"status":     req.Status,
			"updateTime": util.Datetime(time.Now()),
		},
	}
	if _, err := svc.CollectionTask.UpdateByID(ctx, req.ID, data); err != nil {
		log.Logger().WithContext(ctx).Error("update task: ", err.Error())
		return model.Task{}, ErrDatabase
	}

	return req, nil
}

type SearchTaskReq struct {
	ProjectID       primitive.ObjectID `json:"projectId"`
	ID              primitive.ObjectID `json:"id"`
	Name            string             `json:"name"`
	Status          []string           `json:"status"`
	Labeler         []string           `json:"labeler"`
	Checker         []string           `json:"checker"`
	UpdateTimeStart string             `json:"updateTimeStart"`
	UpdateTimeEnd   string             `json:"updateTimeEnd"`
	PType           string             `json:"pType"`
	Version         []int              `json:"version"`
	UserID          int
	DataScope       string
	dto.Pagination
}

type SearchTaskResp struct {
	ID         primitive.ObjectID `json:"id"`
	Name       string             `json:"name"`
	Status     string             `json:"status"`
	Labeler    string             `json:"labeler"`
	Checker    string             `json:"checker"`
	UpdateTime util.Datetime      `json:"updateTime"`
}

func (svc *LabelerService) SearchTask(ctx context.Context, req SearchTaskReq) ([]SearchTaskResp, int, error) {
	filter, err := buildFilter(req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	cursor, err := svc.CollectionTask.Find(ctx, filter, buildOptions(req))
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	var tasks []model.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	results := svc.tasksToSearchTaskResp(ctx, tasks)

	count, err := svc.CollectionTask.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	return results, int(count), nil
}

func buildOptions(req SearchTaskReq) *options.FindOptions {
	if req.PageIndex < 1 {
		req.PageIndex = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 10
	}
	opts := options.Find().
		SetLimit(int64(req.PageSize)).
		SetSkip(int64((req.PageIndex - 1) * req.PageSize)).
		SetSort(bson.D{{"_id", 1}})
	return opts
}

func buildFilter(req SearchTaskReq) (bson.M, error) {
	filter := bson.M{}
	if len(req.Version) > 0 {
		filter["version"] = bson.M{
			"$in": req.Version,
		}
	}
	if !req.ID.IsZero() {
		filter["_id"] = req.ID
	}
	if !req.ProjectID.IsZero() {
		filter["projectId"] = req.ProjectID
	}
	if len(req.Status) > 0 {
		filter["status"] = bson.M{
			"$in": req.Status,
		}
	}
	if len(req.Labeler) > 0 {
		filter["permissions.labeler.id"] = bson.M{
			"$in": req.Labeler,
		}
	}
	if len(req.Checker) > 0 {
		filter["permissions.checker.id"] = bson.M{
			"$in": req.Checker,
		}
	}
	if len(req.Name) > 0 {
		filter["name"] = bson.M{
			"$regex": req.Name,
		}
	}
	if len(req.UpdateTimeStart) > 0 {
		t, err := time.Parse(util.TimeLayoutDatetime, req.UpdateTimeStart)
		if err != nil {
			return nil, ErrTimeParse
		}
		filter["updateTime"] = bson.M{
			"$gte": t,
		}
	}
	if len(req.UpdateTimeEnd) > 0 {
		t, err := time.Parse(util.TimeLayoutDatetime, req.UpdateTimeEnd)
		if err != nil {
			return nil, ErrTimeParse
		}
		value, ok := filter["updateTime"]
		if ok {
			value.(bson.M)["$lte"] = t
			filter["updateTime"] = value
		}

		//filter["updateTime"] = bson.M{
		//	"$lte": t,
		//}
	}
	switch req.PType {
	case PermissionTypeLabeler:
		filter["permissions.labeler.id"] = fmt.Sprint(req.UserID)
	case PermissionTypeChecker:
		filter["permissions.checker.id"] = fmt.Sprint(req.UserID)
	default:
		if req.DataScope == "5" {
			filter["$or"] = bson.A{
				bson.M{
					"permissions.labeler.id": fmt.Sprint(req.UserID),
				},
				bson.M{
					"permissions.checker.id": fmt.Sprint(req.UserID)},
			}
		}
	}
	return filter, nil
}

func (svc *LabelerService) tasksToSearchTaskResp(ctx context.Context, tasks []model.Task) []SearchTaskResp {
	ids := make([]string, 0)
	for _, task := range tasks {
		if task.Permissions.Labeler != nil {
			ids = append(ids, task.Permissions.Labeler.ID)
		}
		if task.Permissions.Checker != nil {
			ids = append(ids, task.Permissions.Checker.ID)
		}
	}

	res := make([]SearchTaskResp, len(tasks))

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
		res[i] = SearchTaskResp{
			ID:         task.ID,
			Name:       task.Name,
			Status:     task.Status,
			Labeler:    labeler,
			Checker:    checker,
			UpdateTime: task.UpdateTime,
		}

	}
	return res
}

func (svc *LabelerService) GetTask(ctx context.Context, id primitive.ObjectID) (model.Task, error) {
	var task model.Task
	var users []models.SysUser
	ids := make([]string, 0)
	if err := svc.CollectionTask.FindOne(ctx, bson.D{{"_id", id}}).Decode(&task); err != nil {
		if err == mongo.ErrNoDocuments {
			return model.Task{}, ErrNoDoc
		}
		log.Logger().WithContext(ctx).Error("get task: ", err.Error())
		return model.Task{}, err
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
	return task, nil
}

type AllocateTasksReq struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Number    int64              `json:"number"`
	Persons   []string           `json:"persons"`
}

func (svc *LabelerService) AllocateTasks(ctx context.Context, req AllocateTasksReq) error {
	filter := bson.M{
		"projectId": req.ProjectID,
		"permissions.labeler": bson.M{
			"$exists": false,
		},
	}
	count, err := svc.CollectionTask.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return err
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
		result, err := svc.CollectionTask.Find(ctx, filter, opts)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return err
		}

		var tasks []model.Task
		if err = result.All(ctx, &tasks); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return err
		}

		idArray := []primitive.ObjectID{}
		for _, t := range tasks {
			idArray = append(idArray, t.ID)
		}

		ft := bson.M{
			"_id": bson.M{
				"$in": idArray,
			},
		}
		update := bson.M{
			"$set": bson.M{
				"permissions.labeler": model.Person{ID: fmt.Sprint(id)},
				"status":              model.TaskStatusLabeling,
				"updateTime":          util.Datetime(time.Now()),
			},
		}
		if _, err := svc.CollectionTask.UpdateMany(ctx, ft, update); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return err
		}
	}

	return nil
}

type ResetTasksReq struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Persons   []string           `json:"persons"`
	Statuses  []string           `json:"statuses"`
}

func (svc *LabelerService) ResetTasks(ctx context.Context, req ResetTasksReq) error {
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
	if _, err := svc.CollectionTask.UpdateMany(ctx, filter, update); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return err
	}
	return nil
}

func (svc *LabelerService) CheckTask(ctx context.Context, req model.Task, userID int) (model.Task, error) {
	task, err := svc.GetTask(ctx, req.ID)
	if err != nil {
		return model.Task{}, err
	}
	if task.Permissions.Checker == nil || task.Permissions.Checker.ID != strconv.Itoa(userID) {
		return model.Task{}, errors.New("当前用户无权限审核")
	}
	if !TaskStatusCheck(false, task.Status, req.Status) {
		return model.Task{}, errors.New(fmt.Sprintf("当前任务状态为:%s,无法修改为:%s", task.Status, req.Status))
	}

	data := bson.M{
		"$set": bson.M{
			"contents":   req.Contents,
			"status":     req.Status,
			"updateTime": util.Datetime(time.Now()),
		},
	}
	if _, err := svc.CollectionTask.UpdateByID(ctx, req.ID, data); err != nil {
		log.Logger().WithContext(ctx).Error("update task: ", err.Error())
		return model.Task{}, ErrDatabase
	}

	return req, nil
}

type CommentTaskReq struct {
	ID      primitive.ObjectID `json:"id"`
	Content string             `json:"content"`

	UserID string
}

func (svc *LabelerService) CommentTask(ctx context.Context, req CommentTaskReq) error {
	task, err := svc.GetTask(ctx, req.ID)
	if err != nil {
		return err
	}
	if task.Permissions.Checker == nil || task.Permissions.Checker.ID != req.UserID {
		return errors.New("当前用户无权限备注")
	}
	comment := model.Comment{
		ID:         req.UserID,
		Content:    req.Content,
		CreateTime: util.Datetime(time.Now()),
	}
	data := bson.M{
		"$set": bson.M{
			"updateTime": util.Datetime(time.Now()),
		},
		"$push": bson.M{
			"comments": comment,
		},
	}
	if _, err := svc.CollectionTask.UpdateByID(ctx, req.ID, data); err != nil {
		log.Logger().WithContext(ctx).Error("update task: ", err.Error())
		return ErrDatabase
	}
	return nil
}

type ModelParseReq struct {
	Raw      model.Tuple `json:"raw"`
	ModelURL string      `json:"modelURL"`
}

func (svc *LabelerService) ModelParse(ctx context.Context, req ModelParseReq) ([]model.Tuple, error) {
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, req.ModelURL, bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Content-Type", "application/json;charset=utf-8")

	client := &http.Client{
		Timeout: 5 * time.Minute,
	}
	resp, err := client.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var respBody struct {
		Code int    `json:"code"`
		Msg  string `json:"error_msg"`
		Data struct {
			Results []model.Tuple `json:"results"`
		} `json:"data"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, err
	}
	if respBody.Msg != "" {
		return nil, fmt.Errorf("解析出错: %s", respBody.Msg)
	}
	if len(respBody.Data.Results) == 0 {
		return nil, errors.New("解析失败")
	}

	return respBody.Data.Results, nil
}

var Labeler = map[string]map[string]bool{
	model.TaskStatusLabeling: {model.TaskStatusLabeling: true, model.TaskStatusSubmit: true},
	model.TaskStatusSubmit:   {model.TaskStatusSubmit: true},
	model.TaskStatusFailed:   {model.TaskStatusFailed: true, model.TaskStatusChecking: true},
}

var Checker = map[string]map[string]bool{
	model.TaskStatusChecking: {model.TaskStatusChecking: true, model.TaskStatusPassed: true, model.TaskStatusFailed: true},
	model.TaskStatusPassed:   {model.TaskStatusChecking: true, model.TaskStatusPassed: true, model.TaskStatusFailed: true},
	model.TaskStatusFailed:   {model.TaskStatusChecking: true, model.TaskStatusPassed: true, model.TaskStatusFailed: true},
}

func TaskStatusCheck(isLabeler bool, src string, dst string) (result bool) {
	if isLabeler {
		if value, ok := Labeler[src]; ok {
			if value[dst] {
				result = true
			}
		}
		return
	}
	if value, ok := Checker[src]; ok {
		if value[dst] {
			result = true
		}
	}
	return
}

type AllocateCheckTasksReq struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Persons   []string           `json:"persons"`
	Number    int64              `json:"number"`
}

func (svc *LabelerService) AllocateCheckTasks(ctx context.Context, req AllocateCheckTasksReq) error {
	if req.Number <= 0 {
		return errors.New("分配任务数量不合法")
	}
	if len(req.Persons) == 0 {
		return errors.New("分配人员数量不能为0")
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

	result, err := svc.CollectionTask.Find(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return err
	}
	var tasks []model.Task
	if err = result.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return err
	}
	if len(tasks) == 0 {
		return errors.New("当前无可分配任务")
	}
	nowTime := util.Datetime(time.Now())
	var totalCount int
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
				"updateTime":          nowTime,
			},
		}
		if _, err := svc.CollectionTask.UpdateOne(ctx, ft, update); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return err
		}
		totalCount++
		if totalCount == int(req.Number) {
			break
		}
	}
	if totalCount == 0 {
		return errors.New("分配失败：标注员和审核员不能是同一人")
	}
	return nil
}

type SearchMyTaskReq struct {
	ID       primitive.ObjectID `json:"id"`
	UserID   string
	TaskType string   `json:"taskType"`
	Status   []string `json:"status"`
}

func (svc *LabelerService) SearchMyTask(ctx context.Context, req SearchMyTaskReq) ([]SearchTaskResp, int, error) {
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

	cursor, err := svc.CollectionTask.Find(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	var tasks []model.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	results := svc.tasksToSearchTaskResp(ctx, tasks)

	count, err := svc.CollectionTask.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	return results, int(count), nil
}

type DownloadTaskReq struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Status    []string           `json:"status"`
}

type DownloadTaskResp struct {
	File     *string `json:"file"`
	FileName string  `json:"filename"`
}

func (svc *LabelerService) DownloadTask(ctx context.Context, req DownloadTaskReq) (DownloadTaskResp, error) {
	filter := bson.M{
		"projectId": req.ProjectID,
		"status": bson.M{
			"$in": req.Status,
		},
	}
	cursor, err := svc.CollectionTask.Find(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTaskResp{}, err
	}

	var tasks []model.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTaskResp{}, err
	}

	nameStr := time.Now().Format("2006-01-02 15-04-05") + "下载文件.zip"
	buf := new(bytes.Buffer)

	zipWriter := zip.NewWriter(buf)

	for _, task := range tasks {
		data, err := json.Marshal(task)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTaskResp{}, err
		}
		taskName := strings.Split(task.Name, ".")
		w1, err := zipWriter.Create(taskName[0] + ".json")
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTaskResp{}, err
		}
		_, err = w1.Write(data)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTaskResp{}, err
		}
	}
	if err = zipWriter.Close(); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTaskResp{}, err
	}

	res := base64.StdEncoding.EncodeToString(buf.Bytes())
	return DownloadTaskResp{File: &res, FileName: nameStr}, nil
}

func (svc *LabelerService) DeleteTask(ctx context.Context, id primitive.ObjectID) error {
	if _, err := svc.CollectionTask.DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		log.Logger().WithContext(ctx).Error("delete task: ", err.Error())
		return err
	}
	return nil
}
