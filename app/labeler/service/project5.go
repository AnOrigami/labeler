package service

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go-admin/app/labeler/model"
	"go-admin/common/log"
)

func (svc *LabelerService) CreateProject5(ctx context.Context, req model.Project5) (model.Project5, error) {
	InitObjectID(&req.ID)
	_, err := svc.CollectionProject5.InsertOne(ctx, &req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Project5{}, err
	}
	return req, nil
}

func (svc *LabelerService) UpdateProject5(ctx context.Context, req model.Project5) (model.Project5, error) {
	_, err := svc.CollectionProject5.ReplaceOne(ctx, bson.D{{"_id", req.ID}}, &req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Project5{}, err
	}
	return req, nil
}

type DeleteProject5Req struct {
	ID primitive.ObjectID
}

type DeleteProject5Resp struct {
	DeletedCount int64 `json:"deletedCount"`
}

func (svc *LabelerService) DeleteProject5(ctx context.Context, req DeleteProject5Req) (DeleteProject5Resp, error) {
	result, err := svc.CollectionProject5.DeleteOne(ctx, bson.D{{"_id", req.ID}})
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DeleteProject5Resp{}, err
	}
	return DeleteProject5Resp{DeletedCount: result.DeletedCount}, nil
}

type SearchProject5Req struct {
	FolderID primitive.ObjectID `json:"folderId"`
}

func (svc *LabelerService) SearchProject5(ctx context.Context, req SearchProject5Req) ([]model.Project5, int, error) {
	cursor, err := svc.CollectionProject5.
		Find(
			ctx,
			bson.M{"folderId": req.FolderID},
			options.Find().SetSort(bson.D{{"_id", -1}}),
		)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	var projects []model.Project5
	if err := cursor.All(ctx, &projects); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	return projects, len(projects), nil
}

type Project5CountReq struct {
	ID primitive.ObjectID
}

type Project5CountResp struct {
	Total            int64 `json:"total"`
	UnallocatedLabel int64 `json:"unallocatedLabel"`
	AllocatedLabel   int64 `json:"allocatedLabel"`
	Labeling         int64 `json:"labeling"`
	Submit           int64 `json:"submit"`
	UnallocatedCheck int64 `json:"unallocatedCheck"`
	AllocatedCheck   int64 `json:"allocatedCheck"`
	Checking         int64 `json:"checking"`
	Passed           int64 `json:"passed"`
	Failed           int64 `json:"failed"`
}

func (svc *LabelerService) Project5Count(ctx context.Context, req Project5CountReq) (Project5CountResp, error) {
	pipe := mongo.Pipeline{
		bson.D{
			{
				"$match",
				bson.D{{"projectId", req.ID}},
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
	cursor, err := svc.CollectionLabeledTask5.Aggregate(ctx, pipe)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Project5CountResp{}, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Project5CountResp{}, err
	}

	var resp Project5CountResp
	for _, result := range results {
		count := int64(result["count"].(int32))
		switch result["_id"] {
		case model.TaskStatusAllocate:
			resp.UnallocatedLabel = count
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
	totalFilter := bson.M{
		"projectId": req.ID,
		"status":    "未分配",
	}
	count, err := svc.CollectionTask5.CountDocuments(ctx, totalFilter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Project5CountResp{}, err
	}
	resp.Total = count

	allocatedLabelFilter := bson.M{
		"projectId": req.ID,
		"status":    "未分配",
	}

	cursor, err = svc.CollectionTask5.Find(ctx, allocatedLabelFilter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Project5CountResp{}, err
	}
	var tasks []*model.Task5
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Project5CountResp{}, err
	}

	var allocatedLabel, unAllocatedLabel int
	for _, task := range tasks {
		if task.Dialog[0].Priority == 0 {
			unAllocatedLabel++
		} else {
			allocatedLabel += task.Dialog[0].Priority
		}
	}
	resp.AllocatedLabel = int64(allocatedLabel)
	resp.UnallocatedCheck = int64(unAllocatedLabel)

	return resp, nil
}
