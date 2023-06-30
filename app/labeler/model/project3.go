package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Project3 struct {
	ID       primitive.ObjectID `bson:"_id" json:"id"`
	Name     string             `bson:"name" json:"name"`
	FolderID primitive.ObjectID `bson:"folderId" json:"folderId"`
	Status   string             `bson:"status" json:"status"`
	Schema   Schema3            `bson:"schema" json:"schema"`
}

type Schema3 struct {
	CommandLabels   []TagSet `bson:"commandLabels" json:"commandLabels"`
	CommandTags     []string `bson:"commandTags" json:"commandTags"`
	CommandJudgment []string `bson:"commandJudgment" json:"commandJudgment"`
	OutputJudgment  []string `bson:"outputJudgment" json:"outputJudgment"`
}

type TagSet struct {
	Name   string   `bson:"name" json:"name"`
	Values []string `bson:"values" json:"values"`
}
