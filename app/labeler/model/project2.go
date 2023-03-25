package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Schema2Labels struct {
	Name   string   `bson:"name" json:"name"`
	Values []string `bson:"values" json:"values"`
}

type Schema2 struct {
	//ID           primitive.ObjectID `bson:"_id" json:"id"`
	ContentTypes []string        `bson:"contentTypes" json:"contentTypes"`
	Labels       []Schema2Labels `bson:"labels" json:"labels"`
}

type Project2 struct {
	ID       primitive.ObjectID `bson:"_id" json:"id"`
	Name     string             `bson:"name" json:"name"`
	FolderID primitive.ObjectID `bson:"folderId" json:"folderId"`
	Status   string             `bson:"status" json:"status"`
	Schema   Schema2            `bson:"schema" json:"schema"`
}
