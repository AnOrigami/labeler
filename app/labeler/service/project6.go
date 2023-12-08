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

func (svc *LabelerService) CreateProject6(ctx context.Context, req model.Project6) (model.Project6, error) {
	InitObjectID(&req.ID)
	_, err := svc.CollectionProject6.InsertOne(ctx, &req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Project6{}, err
	}
	return req, nil
}

func (svc *LabelerService) UpdateProject6(ctx context.Context, req model.Project6) (model.Project6, error) {
	_, err := svc.CollectionProject6.ReplaceOne(ctx, bson.D{{"_id", req.ID}}, &req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Project6{}, err
	}
	return req, nil
}

type DeleteProject6Req struct {
	ID primitive.ObjectID
}

type DeleteProject6Resp struct {
	DeletedCount int64 `json:"deletedCount"`
}

func (svc *LabelerService) DeleteProject6(ctx context.Context, req DeleteProject6Req) (DeleteProject6Resp, error) {
	result, err := svc.CollectionProject6.DeleteOne(ctx, bson.D{{"_id", req.ID}})
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DeleteProject6Resp{}, err
	}
	return DeleteProject6Resp{DeletedCount: result.DeletedCount}, nil
}

type SearchProject6Req struct {
	FolderID primitive.ObjectID `json:"folderId"`
}

func (svc *LabelerService) SearchProject6(ctx context.Context, req SearchProject6Req) ([]model.Project6, int, error) {
	cursor, err := svc.CollectionProject6.
		Find(
			ctx,
			bson.M{"folderId": req.FolderID},
			options.Find().SetSort(bson.D{{"_id", -1}}),
		)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	var projects []model.Project6
	if err := cursor.All(ctx, &projects); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	return projects, len(projects), nil
}

type Project6CountReq struct {
	ID primitive.ObjectID
}

type Project6CountResp struct {
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

func (svc *LabelerService) Project6Count(ctx context.Context, req Project6CountReq) (Project6CountResp, error) {
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
	cursor, err := svc.CollectionTask6.Aggregate(ctx, pipe)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Project6CountResp{}, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Project6CountResp{}, err
	}

	var resp Project6CountResp
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
	allocatedLabelFilter := bson.M{
		"projectId": req.ID,
		"status": bson.M{
			"$in": []string{"已提交", "待审核", "已审核", "审核不通过"},
		},
		"permissions.labeler": bson.M{
			"$exists": true,
		},
	}
	allocatedCheckFilter := bson.M{
		"projectId": req.ID,
		"status": bson.M{
			"$in": []string{"已审核", "审核不通过"},
		},
		"permissions.checker": bson.M{
			"$exists": true,
		},
	}
	count, err := svc.CollectionTask6.CountDocuments(ctx, allocatedLabelFilter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Project6CountResp{}, err
	}
	resp.AllocatedLabel = resp.Labeling + count

	count, err = svc.CollectionTask6.CountDocuments(ctx, allocatedCheckFilter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return Project6CountResp{}, err
	}
	resp.AllocatedCheck = resp.Checking + count

	resp.UnallocatedCheck = resp.Submit
	return resp, nil
}
