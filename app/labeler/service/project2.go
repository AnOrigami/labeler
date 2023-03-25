package service

import (
	"context"
	"go-admin/app/labeler/model"
	"go-admin/common/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (svc *LabelerService) CreateProject2(ctx context.Context, req model.Project2) (model.Project2, error) {
	InitObjectID(&req.ID)
	_, err := svc.CollectionProject2.InsertOne(ctx, &req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Project2{}, err
	}
	return req, nil
}

func (svc *LabelerService) UpdateProject2(ctx context.Context, req model.Project2) (model.Project2, error) {
	_, err := svc.CollectionProject2.ReplaceOne(ctx, bson.D{{"_id", req.ID}}, &req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Project2{}, err
	}
	return req, nil
}

type DeleteProject2Req struct {
	ID primitive.ObjectID
}

type DeleteProject2Resp struct {
	DeletedCount int64 `json:"deletedCount"`
}

func (svc *LabelerService) DeleteProject2(ctx context.Context, req DeleteProject2Req) (DeleteProject2Resp, error) {
	result, err := svc.CollectionProject2.DeleteOne(ctx, bson.D{{"_id", req.ID}})
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DeleteProject2Resp{}, err
	}
	return DeleteProject2Resp{DeletedCount: result.DeletedCount}, nil
}

type SearchProject2Req struct {
	FolderID primitive.ObjectID `json:"folderId"`
}

func (svc *LabelerService) SearchProject2(ctx context.Context, req SearchProject2Req) ([]model.Project2, int, error) {
	cursor, err := svc.CollectionProject2.
		Find(
			ctx,
			bson.M{"folderId": req.FolderID},
			options.Find().SetSort(bson.D{{"_id", -1}}),
		)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	var projects []model.Project2
	if err := cursor.All(ctx, &projects); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	return projects, len(projects), nil
}

type Project2CountReq struct {
	ID primitive.ObjectID
}

type Project2CountResp struct {
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

func (svc *LabelerService) Project2Count(ctx context.Context, req Project2CountReq) (Project2CountResp, error) {
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
	cursor, err := svc.CollectionTask2.Aggregate(ctx, pipe)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Project2CountResp{}, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Project2CountResp{}, err
	}

	var resp Project2CountResp
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
