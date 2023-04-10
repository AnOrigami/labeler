package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-admin/common/util"
)

type Task2ContentItem struct {
	Name  string `bson:"name" json:"name"`
	Value string `bson:"value" json:"value"`
}

type Task2LabelItem struct {
	Name  string `bson:"name" json:"name"`
	Value string `bson:"value" json:"value"`
}

type Task2 struct {
	ID          primitive.ObjectID `bson:"_id" json:"id"`
	Name        string             `bson:"name" json:"name"`
	ProjectID   primitive.ObjectID `bson:"projectId" json:"projectId"`
	Status      string             `bson:"status" json:"status"`
	Permissions Permissions        `bson:"permissions" json:"permissions"`
	UpdateTime  util.Datetime      `bson:"updateTime" json:"updateTime"`
	Contents    []Task2ContentItem `bson:"contents" json:"contents"`
	Labels      []Task2LabelItem   `bson:"labels" json:"labels"`
}
