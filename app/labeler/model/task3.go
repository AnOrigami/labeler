package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-admin/common/util"
)

type Task3CommandItem struct {
	Content string     `bson:"content" json:"content"`
	Result  CommandRes `bson:"result" json:"result"`
}

type CommandRes struct {
	Labels   []Label    `bson:"labels" json:"labels"`
	Tags     Tag        `bson:"tags" json:"tags"`
	Judgment []Judgment `bson:"judgment" json:"judgment"`
	Remarks  string     `bson:"remarks" json:"remarks"`
}

type Label struct {
	Name    string   `bson:"name" json:"name"`
	Value   string   `bson:"value" json:"value"`
	Options []string `bson:"options" json:"options"`
}

type Tag struct {
	Values  []string `bson:"values" json:"values"`
	Options []string `bson:"options" json:"options"`
}

type Task3OutputItem struct {
	Content string    `bson:"content" json:"content"`
	Result  OutputRes `bson:"result" json:"result"`
	Skip    bool      `bson:"skip" json:"skip"`
}

type OutputRes struct {
	Score    int64      `bson:"score" json:"score"`
	Judgment []Judgment `bson:"judgment" json:"judgment"`
	Remarks  string     `bson:"remarks" json:"remarks"`
}

type Judgment struct {
	Name  string `bson:"name" json:"name"`
	Value string `bson:"value" json:"value"`
}
type Task3 struct {
	ID          primitive.ObjectID `bson:"_id" json:"id"`
	Name        string             `bson:"name" json:"name"`
	ProjectID   primitive.ObjectID `bson:"projectId" json:"projectId"`
	Status      string             `bson:"status" json:"status"`
	Permissions Permissions        `bson:"permissions" json:"permissions"`
	Sort        []int              `bson:"sort" json:"sort"`
	UpdateTime  util.Datetime      `bson:"updateTime" json:"updateTime"`
	Command     Task3CommandItem   `bson:"command" json:"command"`
	Output      []Task3OutputItem  `bson:"output" json:"output"`
}
