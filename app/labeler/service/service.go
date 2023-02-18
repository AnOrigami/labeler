package service

import (
	"context"
	"go-admin/app/labeler/model"
	ext "go-admin/config"
	"go.mongodb.org/mongo-driver/mongo"
)

type LabelerService struct {
	MongodbClient *mongo.Client
	MongodbDB     *mongo.Database
}

func NewLabelerService(mongodbClient *mongo.Client) *LabelerService {
	cfg := ext.ExtConfig.Mongodb
	return &LabelerService{
		MongodbClient: mongodbClient,
		MongodbDB:     mongodbClient.Database(cfg.LabelerDB),
	}
}

func (svc *LabelerService) CreateProject(ctx context.Context, req model.Project) {
	result, err := svc.MongodbDB.Collection("project").InsertOne(ctx, req)
	_, _ = result, err
}
