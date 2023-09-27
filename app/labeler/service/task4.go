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
	"go-admin/common/counter"
	"go-admin/common/dto"
	"go-admin/common/log"
	"go-admin/common/util"
)

type UploadTask4Req struct {
	Rows      []Task4FileRow
	ProjectID primitive.ObjectID
}

type UploadTask4Resp struct {
	UploadCount int `json:"uploadCount"`
}

type Task4FileRow struct {
	Name string
	Data []string
}

func (svc *LabelerService) UploadTask4(ctx context.Context, req UploadTask4Req) (UploadTask4Resp, error) {
	var project model.Project4
	if err := svc.CollectionProject4.FindOne(ctx, bson.M{"_id": req.ProjectID}).Decode(&project); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return UploadTask4Resp{}, errors.New("项目不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask4Resp{}, err
	}

	outputJudgment := make([]model.Judgment, len(project.Schema.OutputJudgment))
	for i, v := range project.Schema.OutputJudgment {
		outputJudgment[i] = model.Judgment{
			Name:  v,
			Value: "未选择",
		}
	}
	scoreGroups := make([]model.ScoreGroup, len(project.Schema.ScoreGroups))
	for i, v := range project.Schema.ScoreGroups {
		scores := make([]model.Score, len(v.Scores))
		for j, k := range v.Scores {
			scores[j].Name = k
		}
		scoreGroups[i] = model.ScoreGroup{
			Name:   v.Name,
			Scores: scores,
			Max:    v.Max,
		}
	}
	tasks := make([]any, len(req.Rows))
	now := util.Datetime(time.Now())
	for i, row := range req.Rows {
		var outputs []model.Task4OutputItem
		var text string
		data := util.DefaultSlice[string](row.Data)
		for i, v := range data {
			if v == "" {
				break
			}
			if i == 0 {
				text = v
			} else {
				output := model.Task4OutputItem{
					Content: v,
					Result: model.Task4OutputRes{
						Judgment:    outputJudgment,
						ScoreGroups: scoreGroups,
					},
				}
				outputs = append(outputs, output)
			}
		}

		tasks[i] = model.Task4{
			ID:          primitive.NewObjectID(),
			Name:        row.Name,
			ProjectID:   req.ProjectID,
			Status:      model.TaskStatusAllocate,
			Permissions: model.Permissions{},
			UpdateTime:  now,
			Text:        text,
			Output:      outputs,
		}
	}
	result, err := svc.CollectionTask4.InsertMany(ctx, tasks)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask4Resp{}, err
	}
	return UploadTask4Resp{UploadCount: len(result.InsertedIDs)}, err
}

type SearchTask4Req = SearchTaskReq

type SearchTask4Resp struct {
	ID         primitive.ObjectID      `json:"id"`
	ProjectID  primitive.ObjectID      `json:"projectId"`
	Name       string                  `json:"name"`
	Status     string                  `json:"status"`
	Labeler    string                  `json:"labeler"`
	UpdateTime util.Datetime           `json:"updateTime"`
	Sort       []int                   `json:"sort"`
	Text       string                  `json:"text"`
	Output     []model.Task4OutputItem ` json:"output"`
}

func (svc *LabelerService) SearchTask4(ctx context.Context, req SearchTask4Req) ([]SearchTask4Resp, int, error) {
	filter, err := buildFilter(req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	cursor, err := svc.CollectionTask4.Find(ctx, filter, buildOptions(req))
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	var tasks []model.Task4
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	results := svc.tasksToSearchTask4Resp(ctx, tasks)

	count, err := svc.CollectionTask4.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	return results, int(count), nil
}

func (svc *LabelerService) tasksToSearchTask4Resp(ctx context.Context, tasks []model.Task4) []SearchTask4Resp {
	ids := make([]string, 0)
	for _, task := range tasks {
		if task.Permissions.Labeler != nil {
			ids = append(ids, task.Permissions.Labeler.ID)
		}
	}

	res := make([]SearchTask4Resp, len(tasks))

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
		res[i] = SearchTask4Resp{
			ID:         task.ID,
			ProjectID:  task.ProjectID,
			Name:       task.Name,
			Status:     task.Status,
			Labeler:    labeler,
			UpdateTime: task.UpdateTime,
			Text:       task.Text,
			Output:     task.Output,
		}

	}
	return res
}

type Task4BatchAllocLabelerReq struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Number    int64              `json:"number"`
	Persons   []string           `json:"persons"`
}

type Task4BatchAllocLabelerResp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) Task4BatchAllocLabeler(ctx context.Context, req Task4BatchAllocLabelerReq) (Task4BatchAllocLabelerResp, error) {
	filter := bson.M{
		"projectId": req.ProjectID,
		"permissions.labeler": bson.M{
			"$exists": false,
		},
	}
	count, err := svc.CollectionTask4.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Task4BatchAllocLabelerResp{}, err
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
		result, err := svc.CollectionTask4.Find(ctx, filter, opts)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task4BatchAllocLabelerResp{}, err
		}

		var tasks []model.Task4
		if err = result.All(ctx, &tasks); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task4BatchAllocLabelerResp{}, err
		}

		ft := bson.M{
			"_id": bson.M{
				"$in": util.Map(tasks, func(v model.Task4) primitive.ObjectID { return v.ID }),
			},
		}
		update := bson.M{
			"$set": bson.M{
				"permissions.labeler": model.Person{ID: fmt.Sprint(id)},
				"status":              model.TaskStatusLabeling,
				"updateTime":          util.Datetime(time.Now()),
			},
		}
		if _, err := svc.CollectionTask4.UpdateMany(ctx, ft, update); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task4BatchAllocLabelerResp{}, err
		}
	}

	return Task4BatchAllocLabelerResp{Count: count}, nil
}

type ResetTasks4Req struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Persons   []string           `json:"persons"`
	Statuses  []string           `json:"statuses"`
}

type ResetTasks4Resp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) ResetTasks4(ctx context.Context, req ResetTasks4Req) (ResetTasks4Resp, error) {
	filter := bson.M{}
	if !req.ProjectID.IsZero() {
		filter["projectId"] = req.ProjectID
	}
	if len(req.Persons) > 0 {
		filter["permissions.labeler.id"] = bson.M{
			"$in": req.Persons,
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
	result, err := svc.CollectionTask4.UpdateMany(ctx, filter, update)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return ResetTasks4Resp{}, err
	}
	return ResetTasks4Resp{Count: result.ModifiedCount}, nil
}

type UpdateTask4Req struct {
	UserID        string                  `json:"-"`
	UserDataScope string                  `json:"-"`
	ID            primitive.ObjectID      `json:"id"`
	Output        []model.Task4OutputItem `json:"output"`
}

func (svc *LabelerService) UpdateTask4(ctx context.Context, req UpdateTask4Req) (model.Task4, error) {
	var task model.Task4
	if err := svc.CollectionTask4.FindOne(ctx, bson.M{"_id": req.ID}).Decode(&task); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Task4{}, errors.New("任务不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task4{}, err
	}
	if req.UserDataScope != "1" && req.UserDataScope != "2" && !task.Permissions.IsLabeler(req.UserID) {
		return model.Task4{}, errors.New("权限不足")
	}
	if err := svc.CheckTask4(ctx, task, req); err != nil {
		return model.Task4{}, err
	}
	task.Output = req.Output
	task.UpdateTime = util.Datetime(time.Now())
	update := bson.M{
		"$set": bson.M{
			"output":     task.Output,
			"updateTime": task.UpdateTime,
		},
	}
	if _, err := svc.CollectionTask4.UpdateByID(ctx, req.ID, update); err != nil {
		log.Logger().WithContext(ctx).Error("update task: ", err.Error())
		return model.Task4{}, err
	}
	return task, nil
}

func (svc *LabelerService) CheckTask4(ctx context.Context, task model.Task4, req UpdateTask4Req) error {
	var project model.Project4
	if err := svc.CollectionProject4.FindOne(ctx, bson.M{"_id": task.ProjectID}).Decode(&project); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("项目不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return err
	}
	for i, v := range req.Output {
		errStr := fmt.Sprintf("输出%v排序未完成", i+1)
		if v.Sort < 0 || v.Sort > 5 {
			return errors.New(errStr)
		}
		errStr = fmt.Sprintf("输出%v未完成标注", i+1)
		for _, j := range v.Result.Judgment {
			if j.Value == "未选择" {
				return errors.New(errStr)
			}
		}
	}
	return nil
}

type BatchSetTask4StatusReq struct {
	UserID        string               `json:"-"`
	UserDataScope string               `json:"-"`
	IDs           []primitive.ObjectID `json:"ids"`
	Status        string               `json:"status"`
}

type BatchSetTask4StatusResp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) BatchSetTask4Status(ctx context.Context, req BatchSetTask4StatusReq) (BatchSetTask4StatusResp, error) {
	if len(req.IDs) == 0 {
		return BatchSetTask4StatusResp{}, errors.New("什么也没有发生")
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
	normalStatusMap := map[string][]string{
		model.TaskStatusFailed:   {model.TaskStatusChecking, model.TaskStatusPassed, model.TaskStatusFailed},
		model.TaskStatusPassed:   {model.TaskStatusChecking, model.TaskStatusPassed, model.TaskStatusFailed},
		model.TaskStatusChecking: {model.TaskStatusSubmit, model.TaskStatusFailed},
		model.TaskStatusSubmit:   {model.TaskStatusLabeling, model.TaskStatusSubmit},
	}
	specialStatusMap := map[string][]string{
		model.TaskStatusFailed:   {model.TaskStatusChecking, model.TaskStatusPassed, model.TaskStatusSubmit},
		model.TaskStatusPassed:   {model.TaskStatusChecking, model.TaskStatusPassed, model.TaskStatusSubmit},
		model.TaskStatusChecking: {model.TaskStatusFailed},
		model.TaskStatusSubmit:   {model.TaskStatusLabeling, model.TaskStatusAllocate},
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
	if req.UserDataScope != "1" && req.UserDataScope != "2" {
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

	result, err := svc.CollectionTask4.UpdateMany(ctx, filter, update)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return BatchSetTask4StatusResp{}, err
	}
	if int(result.ModifiedCount) < len(req.IDs) {
		if req.Status == model.TaskStatusSubmit {
			return BatchSetTask4StatusResp{}, errors.New("提交失败：任务已被分配审核")
		}
		return BatchSetTask4StatusResp{}, errors.New("部分任务状态没有修改")
	}
	return BatchSetTask4StatusResp{Count: result.ModifiedCount}, err
}

type SearchMyTask4Req struct {
	ID       primitive.ObjectID `json:"id"`
	UserID   string
	Status   []string `json:"status"`
	TaskType string   `json:"taskType"`
	dto.Pagination
}

func (svc *LabelerService) SearchMyTask4(ctx context.Context, req SearchMyTask4Req) ([]SearchTask4Resp, int, error) {
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
	cursor, err := svc.CollectionTask4.Find(ctx, filter, buildOptions4(req))
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	var tasks []model.Task4
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	results := svc.tasksToSearchTask4Resp(ctx, tasks)

	count, err := svc.CollectionTask4.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	return results, int(count), nil
}

func buildOptions4(req SearchMyTask4Req) *options.FindOptions {
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

func (svc *LabelerService) DeleteTask4(ctx context.Context, id primitive.ObjectID) error {
	if _, err := svc.CollectionTask4.DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		log.Logger().WithContext(ctx).Error("delete task: ", err.Error())
		return err
	}
	return nil
}

type DownloadTask4Req struct {
	ProjectID       primitive.ObjectID `json:"projectId"`
	Status          []string           `json:"status"`
	UpdateTimeStart string             `json:"updateTimeStart"`
	UpdateTimeEnd   string             `json:"updateTimeEnd"`
}

type DownloadTask4Resp struct {
	File     *string `json:"file"`
	FileName string  `json:"filename"`
}

func (svc *LabelerService) DownloadTask4(ctx context.Context, req DownloadTask4Req) (DownloadTask4Resp, error) {
	filter := bson.M{
		"projectId": req.ProjectID,
		"status": bson.M{
			"$in": req.Status,
		},
	}

	if len(req.UpdateTimeStart) > 0 && len(req.UpdateTimeEnd) > 0 {
		startTime, err := time.Parse(util.TimeLayoutDatetime, req.UpdateTimeStart)
		if err != nil {
			return DownloadTask4Resp{}, ErrTimeParse
		}
		endTime, err := time.Parse(util.TimeLayoutDatetime, req.UpdateTimeEnd)
		if err != nil {
			return DownloadTask4Resp{}, ErrTimeParse
		}
		filter["updateTime"] = bson.M{
			"$gte": startTime,
			"$lte": endTime,
		}
	}

	cursor, err := svc.CollectionTask4.Find(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask4Resp{}, err
	}

	var tasks []*model.Task4
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask4Resp{}, err
	}

	nameStr := time.Now().Format("2006-01-02 15-04-05") + "下载文件.zip"
	buf := new(bytes.Buffer)

	zipWriter := zip.NewWriter(buf)

	for _, task := range tasks {
		data, err := json.Marshal(task)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTask4Resp{}, err
		}
		taskName := strings.Split(task.Name, ".")
		w1, err := zipWriter.Create(taskName[0] + ".json")
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTask4Resp{}, err
		}
		_, err = w1.Write(data)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTask4Resp{}, err
		}
	}
	if err = zipWriter.Close(); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask4Resp{}, err
	}

	res := base64.StdEncoding.EncodeToString(buf.Bytes())
	return DownloadTask4Resp{File: &res, FileName: nameStr}, nil
}

type GetTask4Req struct {
	ID              primitive.ObjectID `json:"id"`
	Name            string             `json:"name"`
	Labeler         string             `json:"labeler"`
	Checker         string             `json:"checker"`
	UpdateTimeStart string             `json:"updateTimeStart"`
	UpdateTimeEnd   string             `json:"updateTimeEnd"`
	Status          []string           `json:"status"`
	WorkType        int64              `json:"workType"`
}

type GetTask4Resp struct {
	Last primitive.ObjectID `json:"last"`
	Next primitive.ObjectID `json:"next"`
	model.Task4
}

func (svc *LabelerService) GetTask4(ctx context.Context, req GetTask4Req, p *actions.DataPermission) (GetTask4Resp, error) {
	var task model.Task4
	var users []models.SysUser
	filter := bson.M{}
	ids := make([]string, 0)
	userID := strconv.Itoa(p.UserId)
	if err := svc.CollectionTask4.FindOne(ctx, bson.D{{"_id", req.ID}}).Decode(&task); err != nil {
		if err == mongo.ErrNoDocuments {
			return GetTask4Resp{}, ErrNoDoc
		}
		log.Logger().WithContext(ctx).Error("get task: ", err.Error())
		return GetTask4Resp{}, err
	}
	if req.WorkType == 1 {
		if !task.Permissions.IsLabeler(userID) {
			return GetTask4Resp{}, errors.New("任务已被撤回/删除，请刷新任务列表重新进入")
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
			return GetTask4Resp{}, errors.New("任务已被撤回/删除，请刷新任务列表重新进入")
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
		filter = buildTask4DetailFilter(req)
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

	var nextTask model.Task4
	var lastTask model.Task4
	filter["_id"] = bson.M{"$gt": task.ID}
	if err := svc.CollectionTask4.FindOne(ctx, filter).Decode(&nextTask); err != nil {
		if err != mongo.ErrNoDocuments {
			log.Logger().WithContext(ctx).Error("get task: ", err.Error())
			return GetTask4Resp{}, err
		}
	}

	filter["_id"] = bson.M{"$lt": task.ID}
	option := options.FindOne().SetSort(bson.M{"_id": -1})
	if err := svc.CollectionTask4.FindOne(ctx, filter, option).Decode(&lastTask); err != nil {
		if err != mongo.ErrNoDocuments {
			log.Logger().WithContext(ctx).Error("get task: ", err.Error())
			return GetTask4Resp{}, err
		}
	}
	res := GetTask4Resp{
		Task4: task,
	}
	res.Last = lastTask.ID
	res.Next = nextTask.ID
	return res, nil
}

func buildTask4DetailFilter(req GetTask4Req) bson.M {
	filter := bson.M{}
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

type Task4BatchAllocCheckerReq struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Persons   []string           `json:"persons"`
	Number    int64              `json:"number"`
}

type Task4BatchAllocCheckerResp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) Task4BatchAllocChecker(ctx context.Context, req Task4BatchAllocCheckerReq) (Task4BatchAllocCheckerResp, error) {
	if req.Number <= 0 {
		return Task4BatchAllocCheckerResp{}, errors.New("分配任务数量不合法")
	}
	if len(req.Persons) == 0 {
		return Task4BatchAllocCheckerResp{}, errors.New("分配人员数量不能为0")
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

	result, err := svc.CollectionTask4.Find(ctx, filter, options.Find().SetProjection(bson.M{
		"_id":                 1,
		"permissions.labeler": 1,
	}))
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Task4BatchAllocCheckerResp{}, err
	}
	var tasks []model.Task4
	if err = result.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Task4BatchAllocCheckerResp{}, err
	}
	if len(tasks) == 0 {
		return Task4BatchAllocCheckerResp{}, errors.New("当前无可分配任务")
	}

	checkerTaskCounter := make(counter.Counter[string], len(req.Persons))
	for _, person := range req.Persons {
		checkerTaskCounter[person] = 0
	}
	for _, task := range tasks {
		checkerTaskCounter.IncIfExists(task.Permissions.Labeler.ID, 1)
	}
	var totalCount int64
	personTasks := make(map[string][]primitive.ObjectID, len(req.Persons))
OuterLoop:
	for len(checkerTaskCounter) > 0 {
		labeler, _ := checkerTaskCounter.PopMax()
		for i := len(tasks) - 1; i >= 0; i-- {
			if totalCount == req.Number {
				break OuterLoop
			}
			if len(personTasks[labeler]) == maxCount {
				break
			}
			if labeler == tasks[i].Permissions.Labeler.ID {
				continue
			}
			personTasks[labeler] = append(personTasks[labeler], tasks[i].ID)
			totalCount++
			//减少标注数量
			checkerTaskCounter.IncIfExists(tasks[i].Permissions.Labeler.ID, -1)
			// 从 tasks 切片中删除任务
			tasks[i] = tasks[len(tasks)-1]
			tasks = tasks[:len(tasks)-1]
		}
	}
	//保存此次分配分任务
	for labeler, checkTasks := range personTasks {
		if _, err := svc.CollectionTask4.UpdateMany(
			ctx,
			bson.M{
				"_id": bson.M{"$in": checkTasks},
			},
			bson.M{
				"$set": bson.M{
					"permissions.checker": model.Person{ID: labeler},
					"status":              model.TaskStatusChecking,
					"updateTime":          util.Datetime(time.Now()),
				},
			},
		); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task4BatchAllocCheckerResp{}, err
		}
	}

	if totalCount == 0 {
		return Task4BatchAllocCheckerResp{}, errors.New("分配失败：标注员和审核员不能是同一人")
	}
	return Task4BatchAllocCheckerResp{Count: totalCount}, nil
}
