package service

import (
	"context"
	"go-admin/app/labeler/model"
	"go-admin/common/log"
	"go.mongodb.org/mongo-driver/bson"
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
		log.Logger().WithContext(ctx).Error("update folder: ", err.Error())
		return model.Task{}, ErrDatabase
	}

	return req, nil
}
