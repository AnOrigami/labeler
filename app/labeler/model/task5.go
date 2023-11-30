package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-admin/common/util"
)

type Task5 struct {
	ID          primitive.ObjectID `bson:"_id" json:"id"`
	Name        string             `bson:"name" json:"name"`
	ProjectID   primitive.ObjectID `bson:"projectId" json:"projectId"`
	Status      string             `bson:"status" json:"status"`
	Permissions Permissions        `bson:"permissions" json:"permissions"`
	UpdateTime  util.Datetime      `bson:"updateTime" json:"updateTime"`
	Contents    []ContentText      `json:"contents"`
}

type ContentText struct {
	SessionID    string        `json:"session_id"`
	TurnID       int           `json:"turn_id"`
	UserContent  string        `json:"user_content"`
	BotResponse  string        `json:"bot_response"`
	States       string        `json:"states"`
	Actions      []Action      `json:"actions"`
	CreatedAt    string        `json:"created_at"`
	ModelOutputs []ModelOutput `json:"model_output"`
	UserID       string        `json:"user_id"`
	LikeFlag     string        `json:"like_flag"`
	Feedback     string        `json:"feedback"`
	NewAction    string        `json:"new_action"`
	NewOutputs   []ModelOutput `json:"new_outputs"`
}

type Action struct {
	ActionName   string   `json:"action_name"`
	ActionObject []string `json:"action_object"`
}

type ModelOutput struct {
	Action  string `json:"action"`
	Content string `json:"content"`
}
