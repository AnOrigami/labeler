package service

import (
	"context"
	"go-admin/app/labeler/model"
	"go-admin/common/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func (svc *LabelerService) GetSchema(ctx context.Context) ([]model.Schema, error) {
	cursor, err := svc.CollectionSchema.Find(ctx, bson.D{})
	if err != nil {
		log.Logger().WithContext(ctx).Error("get schema: ", err.Error())
		return nil, ErrDatabase
	}

	var results []model.Schema
	if err = cursor.All(ctx, &results); err != nil {
		log.Logger().WithContext(ctx).Error("get schema: ", err.Error())
		return nil, ErrDatabase
	}

	return results, nil
}

func (svc *LabelerService) CreateSchema(ctx context.Context, req model.Schema) (model.Schema, error) {
	InitObjectID(&req.ID)
	if _, err := svc.CollectionSchema.InsertOne(ctx, req); err != nil {
		log.Logger().WithContext(ctx).Error("create schema: ", err.Error())
		return model.Schema{}, ErrDatabase
	}

	return req, nil
}

func (svc *LabelerService) UpdateSchema(ctx context.Context, req model.Schema) (model.Schema, error) {
	if _, err := svc.CollectionSchema.ReplaceOne(ctx, bson.D{{"_id", req.ID}}, &req); err != nil {
		log.Logger().WithContext(ctx).Error("update schema: ", err.Error())
		return model.Schema{}, ErrDatabase
	}

	return req, nil
}

func (svc *LabelerService) DeleteSchema(ctx context.Context, id primitive.ObjectID) error {
	if _, err := svc.CollectionSchema.DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		log.Logger().WithContext(ctx).Error("delete schema: ", err.Error())
		return ErrDatabase
	}

	log.Logger().WithContext(ctx).Warnf("delete schema:%s", id.Hex())

	return nil
}
