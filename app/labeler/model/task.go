package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-admin/common/util"
)

const (
	TaskStatusAllocate = "未分配"
	TaskStatusLabeling = "待标注"
	TaskStatusSubmit   = "已提交"
	TaskStatusChecking = "待审核"
	TaskStatusPassed   = "已审核"
	TaskStatusFailed   = "审核不通过"
)

type Task struct {
	ID          primitive.ObjectID `bson:"_id" json:"id"`
	ProjectID   primitive.ObjectID `bson:"projectId" json:"projectId"`
	Name        string             `bson:"name" json:"name"`
	Status      string             `bson:"status" json:"status"`
	Document    string             `bson:"document" json:"document"`
	Permissions Permissions        `bson:"permissions" json:"permissions"`
	Contents    []Content          `bson:"contents" json:"contents"`
	Activities  []Activity         `bson:"activities" json:"activities"`
	UpdateTime  util.Datetime      `bson:"updateTime" json:"updateTime"`
	Comments    []Comment          `bson:"comments,omitempty" json:"comments,omitempty"`
}

type Comment struct {
	ID         string        `bson:"id" json:"id"`
	Content    string        `bson:"content" json:"content"`
	CreateTime util.Datetime `bson:"createTime" json:"createTime"`
}

type Content struct {
	ID       string      `bson:"id" json:"id"`
	Raw      Tuple       `bson:"raw" json:"raw"`
	Editable bool        `bson:"editable" json:"editable"`
	Results  []Tuple     `bson:"results" json:"results"`
	Del      bool        `bson:"del" json:"del"`
	Changes  primitive.M `bson:"changes" json:"changes"`
}

type Permissions struct {
	Labeler *Person `bson:"labeler,omitempty" json:"labeler"`
	Checker *Person `bson:"checker,omitempty" json:"checker"`
}

func (p Permissions) IsLabeler(id string) bool {
	return p.Labeler != nil && p.Labeler.ID == id
}

func (p Permissions) IsChecker(id string) bool {
	return p.Checker != nil && p.Checker.ID == id
}

type Person struct {
	ID       string `bson:"id" json:"id"`
	NickName string `json:"nickName"`
}

type Tuple struct {
	Groups  []Group     `bson:"groups" json:"groups"`
	Del     bool        `bson:"del" json:"del"`
	Changes primitive.M `bson:"changes" json:"changes"`
}

type Group struct {
	ID       string      `bson:"id" json:"id"`
	Type     string      `bson:"type" json:"type"`
	Status   string      `bson:"status" json:"status"`
	Entities []Entity    `bson:"entities" json:"entities"`
	Del      bool        `bson:"del" json:"del"`
	Changes  primitive.M `bson:"changes" json:"changes"`
}

type Entity struct {
	ID        string      `bson:"id" json:"id"`
	Sentences []Sentence  `bson:"sentences" json:"sentences"`
	Del       bool        `bson:"del" json:"del"`
	Changes   primitive.M `bson:"changes" json:"changes"`
}

type Sentence struct {
	ID      string      `bson:"id" json:"id"`
	Text    string      `bson:"text" json:"text"`
	Source  string      `bson:"source" json:"source"`
	Span    Span        `bson:"span" json:"span"`
	Del     bool        `bson:"del" json:"del"`
	Changes primitive.M `bson:"changes" json:"changes"`
}

type Span struct {
	Left  int `bson:"left" json:"left"`
	Right int `bson:"right" json:"right"`
}

type Activity struct {
	User      string        `bson:"user" json:"user"`
	Role      string        `bson:"role" json:"role"`
	Action    string        `bson:"action" json:"action"`
	Parameter []interface{} `bson:"parameter" json:"parameter"`
}

//type Changes struct {
//	New  string `bson:"new" json:"new"`
//	Old  string `bson:"old" json:"old"`
//	Type string `bson:"type" json:"type"`
//}
