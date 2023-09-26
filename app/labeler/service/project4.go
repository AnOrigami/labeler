package service

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go-admin/app/labeler/model"
	"go-admin/common/log"
)

func (svc *LabelerService) CreateProject4(ctx context.Context, req model.Project4) (model.Project4, error) {
	InitObjectID(&req.ID)
	for _, v := range req.Schema.ScoreGroups {
		if v.Max > 5 {
			return model.Project4{}, errors.New("分数最大值为5")
		}
	}
	_, err := svc.CollectionProject4.InsertOne(ctx, &req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Project4{}, err
	}
	return req, nil
}

func (svc *LabelerService) UpdateProject4(ctx context.Context, req model.Project4) (model.Project4, error) {
	_, err := svc.CollectionProject4.ReplaceOne(ctx, bson.D{{"_id", req.ID}}, &req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Project4{}, err
	}
	return req, nil
}

type DeleteProject4Req struct {
	ID primitive.ObjectID
}

type DeleteProject4Resp struct {
	DeletedCount int64 `json:"deletedCount"`
}

func (svc *LabelerService) DeleteProject4(ctx context.Context, req DeleteProject4Req) (DeleteProject4Resp, error) {
	result, err := svc.CollectionProject4.DeleteOne(ctx, bson.D{{"_id", req.ID}})
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DeleteProject4Resp{}, err
	}
	return DeleteProject4Resp{DeletedCount: result.DeletedCount}, nil
}

type SearchProject4Req struct {
	FolderID primitive.ObjectID `json:"folderId"`
}

func (svc *LabelerService) SearchProject4(ctx context.Context, req SearchProject4Req) ([]model.Project4, int, error) {
	cursor, err := svc.CollectionProject4.
		Find(
			ctx,
			bson.M{"folderId": req.FolderID},
			options.Find().SetSort(bson.D{{"_id", -1}}),
		)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	var projects []model.Project4
	if err := cursor.All(ctx, &projects); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	return projects, len(projects), nil
}

type Project4CountReq struct {
	ID primitive.ObjectID
}

type Project4CountResp struct {
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

func (svc *LabelerService) Project4Count(ctx context.Context, req Project4CountReq) (Project4CountResp, error) {
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
	cursor, err := svc.CollectionTask4.Aggregate(ctx, pipe)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Project4CountResp{}, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Project4CountResp{}, err
	}

	var resp Project4CountResp
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
		resp.Total += count
	}
	resp.AllocatedCheck = resp.Checking + resp.Passed + resp.Failed
	resp.UnallocatedCheck = resp.Submit
	resp.AllocatedLabel = resp.Total - resp.UnallocatedLabel
	return resp, nil
}
