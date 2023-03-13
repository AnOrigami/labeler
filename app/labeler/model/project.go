package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Project struct {
	ID       primitive.ObjectID `bson:"_id" json:"id"`
	Name     string             `bson:"name" json:"name"`
	FolderID primitive.ObjectID `bson:"folderId" json:"folderId"`
	Status   string             `bson:"status" json:"status"`
	Schema   Schema             `bson:"schema" json:"schema"`
}
