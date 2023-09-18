package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Project4 struct {
	ID       primitive.ObjectID `bson:"_id" json:"id"`
	Name     string             `bson:"name" json:"name"`
	FolderID primitive.ObjectID `bson:"folderId" json:"folderId"`
	Status   string             `bson:"status" json:"status"`
	Schema   Schema4            `bson:"schema" json:"schema"`
}

type Schema4 struct {
	OutputJudgment []string   `bson:"outputJudgment" json:"outputJudgment"`
	Scores         []ScoreSet `bson:"scoreSet" json:"scoreSet"`
}

type ScoreSet struct {
	Name string `bson:"name" json:"name"`
	Max  int64  `bson:"max" json:"max"`
}
