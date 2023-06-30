package service

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
