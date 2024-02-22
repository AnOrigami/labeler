package model

import (
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go-admin/common/util"
)

type Task5 struct {
	ID            primitive.ObjectID `bson:"_id" json:"id"`
	Name          string             `bson:"name" json:"name"`
	FullName      string             `bson:"fullName" json:"fullName"`
	ProjectID     primitive.ObjectID `bson:"projectId" json:"projectId"`
	Status        string             `bson:"status" json:"status"`
	Permissions   Permissions        `bson:"permissions" json:"permissions"`
	UpdateTime    util.Datetime      `bson:"updateTime" json:"updateTime"`
	Dialog        []ContentText      `bson:"dialog" json:"dialog"`
	Remark        string             `bson:"remark" json:"remark"`
	RemarkLen     int                `bson:"remarkLen" json:"remarkLen"`
	RemarkOptions int                `bson:"remarkOptions" json:"remarkOptions"`
	WordCount     int                `bson:"wordCount" json:"wordCount"`       //阅读量
	EditQuantity  int                `bson:"editQuantity" json:"editQuantity"` //修改量
	WorkQuantity  int                `bson:"workQuantity" json:"workQuantity"` //工作量
	Score         Scores             `bson:"score" json:"score"`
	HasScore      bool               `bson:"hasScore" json:"hasScore"`
}
type Scores struct {
	//能准确识别风险并提出转介建议
	IdentifyRisk int `bson:"identifyRisk" json:"identifyRisk"`
	//尊重、好奇地倾听和理解来访者
	UnderstandingVisitor int `bson:"understandingVisitor" json:"understandingVisitor"`
	//向来访者表达适当的共情和关怀
	ExpressingCare int `bson:"respondingVisitor" json:"respondingVisitor"`
	//接纳并恰当回应来访者的负面反馈
	AcceptFeedback int `bson:"respectVisitor" json:"respectVisitor"`
	//在来访者所说的内容中选择并进行了恰当回应或提问以推进咨询进程
	AdvanceProcess int `bson:"acceptFeedback" json:"acceptFeedback"`
	//促进来访者在咨询中更多表达、呈现更多内容
	PromoteProcess int `bson:"advanceProcess" json:"advanceProcess"`
	//启发或协助来访者有了解决方案或思路，或一小步的行动
	InspirationAssistance int `bson:"findSolution" json:"findSolution"`
	//缓解了来访者的负面情绪/提升了其积极情绪
	RelieveEmotions int `bson:"enoughContent" json:"enoughContent"`
	//来访者是否反馈良好
	VisitorFeedback int `bson:"visitorFeedback" json:"visitorFeedback"`
}

type ContentText struct {
	SessionID    string         `bson:"sessionId" json:"session_id"`
	TurnID       int            `bson:"turnId" json:"turn_id"`
	UserContent  string         `bson:"userContent" json:"user_content"`
	BotResponse  string         `bson:"botResponse" json:"bot_response"`
	States       string         `bson:"states" json:"states"`
	Actions      []Action       `bson:"actions" json:"actions"`
	CreatedAt    string         `bson:"createdAt" json:"created_at"`
	ModelOutputs []ModelOutput  `bson:"modelOutputs" json:"model_output"`
	UserID       string         `bson:"userId" json:"user_id"`
	LikeFlag     string         `bson:"likeFlag" json:"like_flag"`
	Feedback     string         `bson:"feedback" json:"feedback"`
	NewAction    []Action       `bson:"newAction" json:"new_action"`
	NewOutputs   []ModelOutput  `bson:"newOutputs" json:"new_outputs"`
	Entities     []EntityOption `bson:"entities" json:"entities"`
	Edit         bool           `bson:"edit" json:"edit"`
	Skip         int            `bson:"skip" json:"skip"`
	Version      int            `bson:"version" json:"version"`
	Priority     int            `bson:"priority" json:"priority"`
}
type EntityOption struct {
	Class         string `bson:"class" json:"class"`
	Num           int    `json:"num" json:"num"`
	Type          string `bson:"type" json:"type"`
	ObjectSummary string `bson:"objectSummary" json:"object_summary"`
	ClassType     string `bson:"classType" json:"classType"`
}

type Action struct {
	ActionName     string   `bson:"actionName" json:"action_name"`
	ActionListNode string   `bson:"actionListNode" json:"actionListNode"`
	ActionObject   []Object `bson:"actionObject" json:"action_object"`
	SolutionMethod string   `bson:"solutionMethod" json:"solution_method"`
}

type Object struct {
	ObjectName    string `bson:"objectName" json:"object_name"`
	ObjectSummary string `bson:"objectSummary" json:"object_summary"`
}

type ModelOutput struct {
	Action  string `bson:"action" json:"action"`
	Content string `bson:"content" json:"content"`
}
