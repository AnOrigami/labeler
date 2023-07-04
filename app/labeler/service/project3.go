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

func (svc *LabelerService) CreateProject3(ctx context.Context, req model.Project3) (model.Project3, error) {
	InitObjectID(&req.ID)
	_, err := svc.CollectionProject3.InsertOne(ctx, &req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Project3{}, err
	}
	return req, nil
}

func (svc *LabelerService) UpdateProject3(ctx context.Context, req model.Project3) (model.Project3, error) {
	_, err := svc.CollectionProject3.ReplaceOne(ctx, bson.D{{"_id", req.ID}}, &req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Project3{}, err
	}
	return req, nil
}

type DeleteProject3Req struct {
	ID primitive.ObjectID
}

type DeleteProject3Resp struct {
	DeletedCount int64 `json:"deletedCount"`
}

func (svc *LabelerService) DeleteProject3(ctx context.Context, req DeleteProject3Req) (DeleteProject3Resp, error) {
	result, err := svc.CollectionProject3.DeleteOne(ctx, bson.D{{"_id", req.ID}})
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DeleteProject3Resp{}, err
	}
	return DeleteProject3Resp{DeletedCount: result.DeletedCount}, nil
}

type SearchProject3Req struct {
	FolderID primitive.ObjectID `json:"folderId"`
}

func (svc *LabelerService) SearchProject3(ctx context.Context, req SearchProject3Req) ([]model.Project3, int, error) {
	cursor, err := svc.CollectionProject3.
		Find(
			ctx,
			bson.M{"folderId": req.FolderID},
			options.Find().SetSort(bson.D{{"_id", -1}}),
		)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	var projects []model.Project3
	if err := cursor.All(ctx, &projects); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	return projects, len(projects), nil
}

type Project3CountReq struct {
	ID primitive.ObjectID
}

type Project3CountResp struct {
	Total            int64 `json:"total"`
	UnallocatedLabel int64 `json:"unallocatedLabel"`
	AllocatedLabel   int64 `json:"allocatedLabel"`
	Labeling         int64 `json:"labeling"`
	Submit           int64 `json:"submit"`
}

func (svc *LabelerService) Project3Count(ctx context.Context, req Project3CountReq) (Project3CountResp, error) {
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
	cursor, err := svc.CollectionTask3.Aggregate(ctx, pipe)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Project3CountResp{}, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Project3CountResp{}, err
	}

	var resp Project3CountResp
	for _, result := range results {
		count := int64(result["count"].(int32))
		switch result["_id"] {
		case model.TaskStatusAllocate:
			resp.UnallocatedLabel = count
		case model.TaskStatusLabeling:
			resp.Labeling = count
		case model.TaskStatusSubmit:
			resp.Submit = count
		}
		resp.Total += count
	}
	resp.AllocatedLabel = resp.Total - resp.UnallocatedLabel
	return resp, nil
}
