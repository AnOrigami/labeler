package service

import (
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
