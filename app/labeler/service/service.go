package service

import (
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"gorm.io/gorm"

	ext "go-admin/config"
)

var (
	ErrDatabase  = errors.New("服务器出问题了")
	ErrNoDoc     = errors.New("文档不存在")
	ErrTimeParse = errors.New("时间格式异常")
)

type LabelerService struct {
	MongodbClient      *mongo.Client
	MongodbDB          *mongo.Database
	CollectionProject  *mongo.Collection
	CollectionFolder   *mongo.Collection
	CollectionSchema   *mongo.Collection
	CollectionTask     *mongo.Collection
	CollectionProject2 *mongo.Collection
	CollectionTask2    *mongo.Collection
	CollectionFolder2  *mongo.Collection
	CollectionTask3    *mongo.Collection
	CollectionProject3 *mongo.Collection
	CollectionFolder3  *mongo.Collection
	GormDB             *gorm.DB
}

func NewLabelerService(mongodbClient *mongo.Client, gormDB *gorm.DB) *LabelerService {
	cfg := ext.ExtConfig.Mongodb
	svc := &LabelerService{
		MongodbClient: mongodbClient,
		MongodbDB:     mongodbClient.Database(cfg.LabelerDB),
		GormDB:        gormDB,
	}
	svc.CollectionProject = svc.MongodbDB.Collection("project")
	svc.CollectionFolder = svc.MongodbDB.Collection("folder")
	svc.CollectionSchema = svc.MongodbDB.Collection("schema")
	svc.CollectionTask = svc.MongodbDB.Collection("task")
	svc.CollectionProject2 = svc.MongodbDB.Collection("project2")
	svc.CollectionTask2 = svc.MongodbDB.Collection("task2")
	svc.CollectionFolder2 = svc.MongodbDB.Collection("folder2")
	svc.CollectionTask3 = svc.MongodbDB.Collection("task3")
	svc.CollectionProject3 = svc.MongodbDB.Collection("project3")
	svc.CollectionFolder3 = svc.MongodbDB.Collection("folder3")
	return svc
}

func InitObjectID(id *primitive.ObjectID) {
	*id = primitive.NewObjectID()
}
