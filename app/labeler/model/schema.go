package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Schema struct {
	ID    primitive.ObjectID `bson:"_id" json:"id"`
	Name  string             `bson:"name" json:"name"`
	Rule  Rule               `bson:"rule" json:"rule"`
	Model Model              `bson:"model" json:"model"`
}

type Rule struct {
	Raw    []RuleGroup `bson:"raw" json:"raw"`
	Result []RuleGroup `bson:"result" json:"result"`
}

type RuleGroup struct {
	Type string `bson:"type" json:"type"`
}

type Model struct {
	Source string `bson:"source" json:"source"`
	URL    string `bson:"url" json:"url"`
}
