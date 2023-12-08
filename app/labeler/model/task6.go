package model

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-admin/common/util"
)

type Task6 struct {
	ID          primitive.ObjectID `bson:"_id" json:"id"`
	Name        string             `bson:"name" json:"name"`
	ProjectID   primitive.ObjectID `bson:"projectId" json:"projectId"`
	Status      string             `bson:"status" json:"status"`
	Permissions Permissions        `bson:"permissions" json:"permissions"`
	UpdateTime  util.Datetime      `bson:"updateTime" json:"updateTime"`
	Rpg         bson.M             `bson:"rpg" json:"rpg"`
}
