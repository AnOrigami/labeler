package service

import (
	"errors"
	ext "go-admin/config"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrNoDoc = errors.New("文档不存在")

type LabelerService struct {
	MongodbClient     *mongo.Client
	MongodbDB         *mongo.Database
	CollectionProject *mongo.Collection
	CollectionFolder  *mongo.Collection
	CollectionSchema  *mongo.Collection
	CollectionTask    *mongo.Collection
}

func NewLabelerService(mongodbClient *mongo.Client) *LabelerService {
	cfg := ext.ExtConfig.Mongodb
	svc := &LabelerService{
		MongodbClient: mongodbClient,
		MongodbDB:     mongodbClient.Database(cfg.LabelerDB),
	}
	svc.CollectionProject = svc.MongodbDB.Collection("project")
	svc.CollectionFolder = svc.MongodbDB.Collection("folder")
	svc.CollectionSchema = svc.MongodbDB.Collection("schema")
	svc.CollectionTask = svc.MongodbDB.Collection("task")
	return svc
}

func InitObjectID(id *primitive.ObjectID) {
	*id = primitive.NewObjectID()
}
