package model

import (
	"go-admin/common/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	ProjectID   primitive.ObjectID `bson:"projectId" json:"projectId"`
	Status      string             `bson:"status" json:"status"`
	Permissions Permissions        `bson:"permissions" json:"permissions"`
	UpdateTime  util.Datetime      `bson:"updateTime" json:"updateTime"`
	Contents    []Task2ContentItem `bson:"contents" json:"contents"`
	Labels      []Task2LabelItem   `bson:"labels" json:"labels"`
}
