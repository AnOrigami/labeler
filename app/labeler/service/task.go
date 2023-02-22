package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-admin/app/labeler/model"
	"go-admin/common/dto"
	"go-admin/common/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"time"
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

func (svc *LabelerService) UpdateTask(ctx context.Context, req model.Task) (model.Task, error) {
	data := bson.M{"$set": bson.M{"contents": req.Contents}}
	if _, err := svc.CollectionTask.UpdateByID(ctx, req.ID, data); err != nil {
		log.Logger().WithContext(ctx).Error("update task: ", err.Error())
		return model.Task{}, ErrDatabase
	}

	return req, nil
}

type SearchTaskReq struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Status    string             `json:"status"`
	PType     string             `json:"pType"`

	UserID    int
	DataScope string
	dto.Pagination
}

func (svc *LabelerService) SearchTask(ctx context.Context, req SearchTaskReq) ([]model.Task, int, error) {
	filter := bson.M{}
	if !req.ProjectID.IsZero() {
		filter["projectId"] = req.ProjectID
	}
	if len(req.Status) > 0 {
		filter["status"] = req.Status
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

	opts := options.Find().
		SetSort(bson.D{{"_id", 1}}).
		SetLimit(int64(req.PageSize)).
		SetSkip(int64((req.PageIndex - 1) * req.PageSize))
	cursor, err := svc.CollectionTask.Find(ctx, filter, opts)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	var tasks []model.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	count, err := svc.CollectionTask.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	return tasks, int(count), nil
}

func (svc *LabelerService) GetTask(ctx context.Context, id primitive.ObjectID) (model.Task, error) {
	var task model.Task
	if err := svc.CollectionTask.FindOne(ctx, bson.D{{"_id", id}}).Decode(&task); err != nil {
		if err == mongo.ErrNoDocuments {
			return model.Task{}, ErrNoDoc
		}
		log.Logger().WithContext(ctx).Error("get task: ", err.Error())
		return model.Task{}, err
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

	for i, id := range req.Persons {
		opts := options.Find().SetProjection(bson.D{{"_id", 1}})
		if i < len(req.Persons)-1 {
			opts.SetLimit(count / int64(len(req.Persons)))
		} else {
			opts.SetLimit(count/int64(len(req.Persons)) + count%int64(len(req.Persons)))
		}

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

		var idArray bson.A
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
			},
		}
		r, err := svc.CollectionTask.UpdateMany(ctx, ft, update)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return err
		}
		fmt.Printf("id:%s, update:%v", id, r)
	}

	return nil
}

type ModelParseReq struct {
	Raw model.Tuple `json:"raw"`
}

func (svc *LabelerService) ModelParse(ctx context.Context, raw ModelParseReq) ([]model.Tuple, error) {
	buf, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, svc.ModelServerURL, bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")

	client := &http.Client{
		Timeout: 5 * time.Minute,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var respBody struct {
		Data struct {
			Results []model.Tuple `json:"results"`
		} `json:"data"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, err
	}

	return respBody.Data.Results, nil
}
