package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Schema struct {
	ID         primitive.ObjectID `bson:"_id" json:"id"`
	Name       string             `bson:"name" json:"name"`
	RuleGroups []RuleGroup        `bson:"ruleGroups" json:"ruleGroups"`
	Model      Model              `bson:"model" json:"model"`
}

type RuleGroup struct {
	Type string `bson:"type" json:"type"`
}

type Model struct {
	Source string `bson:"source" json:"source"`
	URL    string `bson:"url" json:"url"`
}
