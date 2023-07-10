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
	"go-admin/common/dto"
	"go-admin/common/log"
	"go-admin/common/util"
)

type UploadTask3Req struct {
	Rows      []Task3FileRow
	ProjectID primitive.ObjectID
}

type UploadTask3Resp struct {
	UploadCount int `json:"uploadCount"`
}

type Task3FileRow struct {
	Name string
	Data []string
}

func (svc *LabelerService) UploadTask3(ctx context.Context, req UploadTask3Req) (UploadTask3Resp, error) {
	var project model.Project3
	if err := svc.CollectionProject3.FindOne(ctx, bson.M{"_id": req.ProjectID}).Decode(&project); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return UploadTask3Resp{}, errors.New("项目不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask3Resp{}, err
	}
	commandLabels := make([]model.Label, len(project.Schema.CommandLabels))
	for i, v := range project.Schema.CommandLabels {
		commandLabels[i] = model.Label{
			Name:    v.Name,
			Value:   "",
			Options: v.Values,
		}
	}
	commandTags := model.Tag{
		Options: project.Schema.CommandTags,
	}
	commandJudgment := make([]model.Judgment, len(project.Schema.CommandJudgment))
	for i, v := range project.Schema.CommandJudgment {
		commandJudgment[i] = model.Judgment{
			Name:  v,
			Value: "未选择",
		}
	}
	outputJudgment := make([]model.Judgment, len(project.Schema.OutputJudgment))
	for i, v := range project.Schema.OutputJudgment {
		outputJudgment[i] = model.Judgment{
			Name:  v,
			Value: "未选择",
		}
	}
	tasks := make([]any, len(req.Rows))
	now := util.Datetime(time.Now())
	for i, row := range req.Rows {
		var outputs []model.Task3OutputItem
		var command model.Task3CommandItem
		data := util.DefaultSlice[string](row.Data)
		for i, v := range data {
			if v == "" {
				break
			}
			if i == 0 {
				command.Content = v
			} else {
				output := model.Task3OutputItem{
					Content: v,
					Result: model.OutputRes{
						Judgment: outputJudgment,
					},
				}
				outputs = append(outputs, output)
			}
		}
		command.Result.Labels = commandLabels
		command.Result.Judgment = commandJudgment
		command.Result.Tags = commandTags

		tasks[i] = model.Task3{
			ID:          primitive.NewObjectID(),
			Name:        row.Name,
			ProjectID:   req.ProjectID,
			Status:      model.TaskStatusAllocate,
			Permissions: model.Permissions{},
			UpdateTime:  now,
			Command:     command,
			Output:      outputs,
		}
	}
	result, err := svc.CollectionTask3.InsertMany(ctx, tasks)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask3Resp{}, err
	}
	return UploadTask3Resp{UploadCount: len(result.InsertedIDs)}, err
}

type SearchTask3Req = SearchTaskReq

type SearchTask3Resp struct {
	ID         primitive.ObjectID      `json:"id"`
	ProjectID  primitive.ObjectID      `json:"projectId"`
	Name       string                  `json:"name"`
	Status     string                  `json:"status"`
	Labeler    string                  `json:"labeler"`
	UpdateTime util.Datetime           `json:"updateTime"`
	Sort       []int                   `json:"sort"`
	Command    model.Task3CommandItem  `json:"command"`
	Output     []model.Task3OutputItem ` json:"output"`
}

func (svc *LabelerService) SearchTask3(ctx context.Context, req SearchTask3Req) ([]SearchTask3Resp, int, error) {
	filter, err := buildFilter(req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	cursor, err := svc.CollectionTask3.Find(ctx, filter, buildOptions(req))
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	var tasks []model.Task3
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	results := svc.tasksToSearchTask3Resp(ctx, tasks)

	count, err := svc.CollectionTask3.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	return results, int(count), nil
}

func (svc *LabelerService) tasksToSearchTask3Resp(ctx context.Context, tasks []model.Task3) []SearchTask3Resp {
	ids := make([]string, 0)
	for _, task := range tasks {
		if task.Permissions.Labeler != nil {
			ids = append(ids, task.Permissions.Labeler.ID)
		}
	}

	res := make([]SearchTask3Resp, len(tasks))

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
		res[i] = SearchTask3Resp{
			ID:         task.ID,
			ProjectID:  task.ProjectID,
			Name:       task.Name,
			Status:     task.Status,
			Labeler:    labeler,
			UpdateTime: task.UpdateTime,
			Command:    task.Command,
			Output:     task.Output,
		}

	}
	return res
}

type Task3BatchAllocLabelerReq struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Number    int64              `json:"number"`
	Persons   []string           `json:"persons"`
}

type Task3BatchAllocLabelerResp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) Task3BatchAllocLabeler(ctx context.Context, req Task3BatchAllocLabelerReq) (Task3BatchAllocLabelerResp, error) {
	filter := bson.M{
		"projectId": req.ProjectID,
		"permissions.labeler": bson.M{
			"$exists": false,
		},
	}
	count, err := svc.CollectionTask3.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Task3BatchAllocLabelerResp{}, err
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
		result, err := svc.CollectionTask3.Find(ctx, filter, opts)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task3BatchAllocLabelerResp{}, err
		}

		var tasks []model.Task3
		if err = result.All(ctx, &tasks); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task3BatchAllocLabelerResp{}, err
		}

		ft := bson.M{
			"_id": bson.M{
				"$in": util.Map(tasks, func(v model.Task3) primitive.ObjectID { return v.ID }),
			},
		}
		update := bson.M{
			"$set": bson.M{
				"permissions.labeler": model.Person{ID: fmt.Sprint(id)},
				"status":              model.TaskStatusLabeling,
				"updateTime":          util.Datetime(time.Now()),
			},
		}
		if _, err := svc.CollectionTask3.UpdateMany(ctx, ft, update); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return Task3BatchAllocLabelerResp{}, err
		}
	}

	return Task3BatchAllocLabelerResp{Count: count}, nil
}

type ResetTasks3Req struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Persons   []string           `json:"persons"`
	Statuses  []string           `json:"statuses"`
}

type ResetTasks3Resp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) ResetTasks3(ctx context.Context, req ResetTasks3Req) (ResetTasks3Resp, error) {
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
	result, err := svc.CollectionTask3.UpdateMany(ctx, filter, update)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return ResetTasks3Resp{}, err
	}
	return ResetTasks3Resp{Count: result.ModifiedCount}, nil
}

type UpdateTask3Req struct {
	UserID        string                  `json:"-"`
	UserDataScope string                  `json:"-"`
	ID            primitive.ObjectID      `json:"id"`
	Command       model.Task3CommandItem  `json:"command"`
	Output        []model.Task3OutputItem `json:"output"`
}

func (svc *LabelerService) UpdateTask3(ctx context.Context, req UpdateTask3Req) (model.Task3, error) {
	var task model.Task3
	if err := svc.CollectionTask3.FindOne(ctx, bson.M{"_id": req.ID}).Decode(&task); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Task3{}, errors.New("任务不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task3{}, err
	}
	if req.UserDataScope != "1" && req.UserDataScope != "2" && !task.Permissions.IsLabeler(req.UserID) {
		return model.Task3{}, errors.New("权限不足")
	}
	if err := svc.CheckTask3(ctx, task, req); err != nil {
		return model.Task3{}, err
	}
	task.Command = req.Command
	task.Output = req.Output
	task.UpdateTime = util.Datetime(time.Now())
	update := bson.M{
		"$set": bson.M{
			"command":    task.Command,
			"output":     task.Output,
			"updateTime": task.UpdateTime,
		},
	}
	if _, err := svc.CollectionTask3.UpdateByID(ctx, req.ID, update); err != nil {
		log.Logger().WithContext(ctx).Error("update task: ", err.Error())
		return model.Task3{}, err
	}
	return task, nil
}

func (svc *LabelerService) CheckTask3(ctx context.Context, task model.Task3, req UpdateTask3Req) error {
	var project model.Project3
	if err := svc.CollectionProject3.FindOne(ctx, bson.M{"_id": task.ProjectID}).Decode(&project); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("项目不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return err
	}
	for _, v := range req.Command.Result.Labels {
		if v.Value == "" {
			return errors.New("指令未完成标注")
		}
	}
	for _, v := range req.Command.Result.Judgment {
		if v.Value == "未选择" {
			return errors.New("指令未完成标注")
		}
	}
	for i, v := range req.Output {
		errStr := fmt.Sprintf("输出%v未完成标注", i+1)
		if v.Sort < 1 || v.Sort > len(req.Output) {
			return errors.New(errStr)
		}
		if v.Skip {
			continue
		}
		if v.Result.Score < 1 || v.Result.Score > 7 {
			return errors.New(errStr)
		}
		for _, j := range v.Result.Judgment {
			if j.Value == "未选择" {
				return errors.New(errStr)
			}
		}
	}
	return nil
}

type BatchSetTask3StatusReq struct {
	UserID        string               `json:"-"`
	UserDataScope string               `json:"-"`
	IDs           []primitive.ObjectID `json:"ids"`
	Status        string               `json:"status"`
}

type BatchSetTask3StatusResp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) BatchSetTask3Status(ctx context.Context, req BatchSetTask3StatusReq) (BatchSetTask3StatusResp, error) {
	if len(req.IDs) == 0 {
		return BatchSetTask3StatusResp{}, errors.New("什么也没有发生")
	}
	filter := bson.M{
		"_id": bson.M{
			"$in": req.IDs,
		},
	}
	if req.UserDataScope != "1" && req.UserDataScope != "2" {
		filter["permissions.labeler.id"] = req.UserID
	}
	update := bson.M{
		"$set": bson.M{
			"status":     req.Status,
			"updateTime": util.Datetime(time.Now()),
		},
	}
	result, err := svc.CollectionTask3.UpdateMany(ctx, filter, update)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return BatchSetTask3StatusResp{}, err
	}
	if result.ModifiedCount == 0 {
		return BatchSetTask3StatusResp{}, errors.New("权限不足")
	}
	return BatchSetTask3StatusResp{Count: result.ModifiedCount}, err
}

type SearchMyTask3Req struct {
	ID     primitive.ObjectID `json:"id"`
	UserID string
	Status []string `json:"status"`
	dto.Pagination
}

func (svc *LabelerService) SearchMyTask3(ctx context.Context, req SearchMyTask3Req) ([]SearchTask3Resp, int, error) {
	filter := bson.M{
		"projectId":              req.ID,
		"permissions.labeler.id": req.UserID,
	}
	if len(req.Status) != 0 {
		filter["status"] = bson.M{
			"$in": req.Status,
		}
	}

	cursor, err := svc.CollectionTask3.Find(ctx, filter, buildOptions3(req))
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	var tasks []model.Task3
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	results := svc.tasksToSearchTask3Resp(ctx, tasks)

	count, err := svc.CollectionTask3.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	return results, int(count), nil
}

func buildOptions3(req SearchMyTask3Req) *options.FindOptions {
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

func (svc *LabelerService) DeleteTask3(ctx context.Context, id primitive.ObjectID) error {
	if _, err := svc.CollectionTask3.DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		log.Logger().WithContext(ctx).Error("delete task: ", err.Error())
		return err
	}
	return nil
}

type DownloadTask3Req struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Status    []string           `json:"status"`
}

type DownloadTask3Resp struct {
	File     *string `json:"file"`
	FileName string  `json:"filename"`
}

func (svc *LabelerService) DownloadTask3(ctx context.Context, req DownloadTask3Req) (DownloadTask3Resp, error) {
	filter := bson.M{
		"projectId": req.ProjectID,
		"status": bson.M{
			"$in": req.Status,
		},
	}
	cursor, err := svc.CollectionTask3.Find(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask3Resp{}, err
	}

	var tasks []*model.Task3
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask3Resp{}, err
	}

	nameStr := time.Now().Format("2006-01-02 15-04-05") + "下载文件.zip"
	buf := new(bytes.Buffer)

	zipWriter := zip.NewWriter(buf)

	for _, task := range tasks {
		data, err := json.Marshal(task)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTask3Resp{}, err
		}
		taskName := strings.Split(task.Name, ".")
		w1, err := zipWriter.Create(taskName[0] + ".json")
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTask3Resp{}, err
		}
		_, err = w1.Write(data)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTask3Resp{}, err
		}
	}
	if err = zipWriter.Close(); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask3Resp{}, err
	}

	res := base64.StdEncoding.EncodeToString(buf.Bytes())
	return DownloadTask3Resp{File: &res, FileName: nameStr}, nil
}

func (svc *LabelerService) GetTask3(ctx context.Context, id primitive.ObjectID) (model.Task3, error) {
	var task model.Task3
	var users []models.SysUser
	ids := make([]string, 0)
	if err := svc.CollectionTask3.FindOne(ctx, bson.D{{"_id", id}}).Decode(&task); err != nil {
		if err == mongo.ErrNoDocuments {
			return model.Task3{}, ErrNoDoc
		}
		log.Logger().WithContext(ctx).Error("get task: ", err.Error())
		return model.Task3{}, err
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
