package model

import "go.mongodb.org/mongo-driver/bson/primitive"

const (
	TaskStatusLabeling = "待标注"
	TaskStatusChecking = "待审核"
	TaskStatusPassed   = "已通过"
	TaskStatusFailed   = "未通过"
)

type Task struct {
	ID          primitive.ObjectID `bson:"_id" json:"id"`
	ProjectID   primitive.ObjectID `bson:"projectId" json:"projectId"`
	Name        string             `bson:"name" json:"name"`
	Status      string             `bson:"status" json:"status"`
	Document    string             `bson:"document" json:"document"`
	Permissions Permissions        `bson:"permissions" json:"permissions"`
	Contents    []Content          `bson:"contents" json:"contents"`
}

type Content struct {
	ID      string  `bson:"id" json:"id"`
	Raw     Tuple   `bson:"raw" json:"raw"`
	Results []Tuple `bson:"results" json:"results"`
}

type Permissions struct {
	Labeler *Person `bson:"labeler,omitempty" json:"labeler"`
	Checker *Person `bson:"checker,omitempty" json:"checker"`
}

type Person struct {
	ID string `bson:"id" json:"id"`
}

type Tuple struct {
	Groups []Group `bson:"groups" json:"groups"`
}

type Group struct {
	ID       string   `bson:"id" json:"id"`
	Type     string   `bson:"type" json:"type"`
	Status   string   `bson:"status" json:"status"`
	Entities []Entity `bson:"entities" json:"entities"`
}

type Entity struct {
	ID        string     `bson:"id" json:"id"`
	Sentences []Sentence `bson:"sentences" json:"sentences"`
}

type Sentence struct {
	Text   string `bson:"text" json:"text"`
	Source string `bson:"source" json:"source"`
	Span   Span   `bson:"span" json:"span"`
}

type Span struct {
	Left  int `bson:"left" json:"left"`
	Right int `bson:"right" json:"right"`
}
