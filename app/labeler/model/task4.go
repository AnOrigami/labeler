package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-admin/common/util"
)

type Task4OutputItem struct {
	Content string         `bson:"content" json:"content"`
	Result  Task4OutputRes `bson:"result" json:"result"`
	Sort    int            `bson:"sort" json:"sort"`
}

type Task4OutputRes struct {
	Scores   []Score    `bson:"scores" json:"scores"`
	Judgment []Judgment `bson:"judgment" json:"judgment"`
	Remarks  string     `bson:"remarks" json:"remarks"`
}

type Score struct {
	Name  string `bson:"name" json:"name"`
	Max   int64  `bson:"max" json:"max"`
	Score int64  `bson:"score" json:"score"`
}

type Task4 struct {
	ID          primitive.ObjectID `bson:"_id" json:"id"`
	Name        string             `bson:"name" json:"name"`
	ProjectID   primitive.ObjectID `bson:"projectId" json:"projectId"`
	Status      string             `bson:"status" json:"status"`
	Permissions Permissions        `bson:"permissions" json:"permissions"`
	UpdateTime  util.Datetime      `bson:"updateTime" json:"updateTime"`
	Text        string             `bson:"text" json:"text"`
	Output      []Task4OutputItem  `bson:"output" json:"output"`
}
