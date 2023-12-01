package service

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"go-admin/app/labeler/model"
	"go-admin/common/log"
	"go-admin/common/util"
)

type UploadTask5Req struct {
	Task5     []model.Task5
	ProjectID primitive.ObjectID
	Name      []string
}

type UploadTask5Resp struct {
	UploadCount int `json:"uploadCount"`
}

type Node struct {
	Value    string
	Children []Node
}

var ActionTags = Node{
	Value: "动作",
	Children: []Node{
		{
			Value: "提问",
			Children: []Node{
				{
					Value: "短焦问句",
					Children: []Node{
						{
							Value:    "以`结果问句1`的方式进行提问",
							Children: nil,
						},
						{
							Value:    "以`结果问句2`的方式进行提问",
							Children: nil,
						},
						{
							Value:    "以`结果问句3`的方式进行提问",
							Children: nil,
						},
						{
							Value:    "以`量尺问句1`的方式进行提问",
							Children: nil,
						},
						{
							Value:    "以`量尺问句2`的方式进行提问",
							Children: nil,
						},
						{
							Value:    "以`量尺问句3`的方式进行提问",
							Children: nil,
						},
						{
							Value:    "提问对来访者来说满分是什么样的",
							Children: nil,
						},
						{
							Value:    "以`例外问句1`的方式进行提问",
							Children: nil,
						},
						{
							Value:    "以`例外问句2`的方式进行提问",
							Children: nil,
						},
						{
							Value:    "以`奇迹问句`的方式进行提问",
							Children: nil,
						},
						{
							Value:    "以`应对问句`的方式进行提问",
							Children: nil,
						},
						{
							Value:    "以`关系问句`的方式进行提问",
							Children: nil,
						},
						{
							Value:    "提问如果ta做出改变的话，来访者会有什么不同",
							Children: nil,
						},
					},
				},
				{
					Value: "通用问句",
					Children: []Node{
						{
							Value:    "提问来访者想先谈论哪个话题",
							Children: nil,
						},
						{
							Value:    "提问描述具体事例",
							Children: nil,
						},
						{
							Value:    "提问情绪的原因",
							Children: nil,
						},
						{
							Value:    "提问来访者当下的情绪/感受",
							Children: nil,
						},
						{
							Value:    "提问来访者对某事例的感受",
							Children: nil,
						},
						{
							Value:    "提问来访者对某事例将采取的做法",
							Children: nil,
						},
						{
							Value:    "提问来访者对某事例的想法",
							Children: nil,
						},
						{
							Value:    "提问当下的事例对来访者的影响",
							Children: nil,
						},
						{
							Value:    "提问某想法对来访者的影响",
							Children: nil,
						},
						{
							Value:    "提问某行为对来访者的影响",
							Children: nil,
						},
						{
							Value:    "提问身边资源有哪些",
							Children: nil,
						},
						{
							Value:    "提问自身优势有哪些",
							Children: nil,
						},
						{
							Value:    "提问来访者的期待",
							Children: nil,
						},
						{
							Value:    "提问那我们可以一起探索一下。可以和我说说你最近的生活吗？",
							Children: nil,
						},
						{
							Value:    "提问来访者需要什么帮助",
							Children: nil,
						},
						{
							Value:    "追问模糊信息",
							Children: nil,
						},
						{
							Value:    "追问还有吗？",
							Children: nil,
						},
					},
				},
			},
		},
		{
			Value: "回应",
			Children: []Node{
				{
					Value:    "总结或重复",
					Children: nil,
				},
				{
					Value:    "反馈",
					Children: nil,
				},
				{
					Value:    "一般化",
					Children: nil,
				},
				{
					Value:    "赞美",
					Children: nil,
				},
				{
					Value:    "鼓励",
					Children: nil,
				},
				{
					Value:    "安全岛技术",
					Children: nil,
				},
			},
		},
		{
			Value:    "解释、分析",
			Children: nil,
		},
		{
			Value:    "提供思路、心理作业",
			Children: nil,
		},
		{
			Value:    "心理科普",
			Children: nil,
		},
		{
			Value:    "引导回主题",
			Children: nil,
		},
		{
			Value:    "确认主题",
			Children: nil,
		},
		{
			Value:    "提示如果存在安全风险，请联系线下相关机构或组织。保护自身及他人人身财产安全",
			Children: nil,
		},
		{
			Value:    "其他",
			Children: nil,
		},
	},
}

func (svc *LabelerService) UploadTask5(ctx context.Context, req UploadTask5Req) (UploadTask5Resp, error) {
	var task model.Task5
	if err := svc.CollectionProject4.FindOne(ctx, bson.M{"_id": req.ProjectID}).Decode(&task); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return UploadTask5Resp{}, errors.New("项目不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask5Resp{}, err
	}
	tasks := make([]any, len(req.Task5))
	for i, row := range req.Task5 {
		for j, row2 := range row.Dialog {
			row.Dialog[j].NewAction = row2.Actions
			row.Dialog[j].NewOutputs = row2.ModelOutputs

		}
		tasks[i] = model.Task5{
			ID:          primitive.NewObjectID(),
			Name:        req.Name[i],
			ProjectID:   req.ProjectID,
			Status:      model.TaskStatusAllocate,
			Permissions: model.Permissions{},
			UpdateTime:  util.Datetime(time.Now()),
			Dialog:      row.Dialog,
		}
	}
	result, err := svc.CollectionTask5.InsertMany(ctx, tasks)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask5Resp{}, err
	}
	return UploadTask5Resp{UploadCount: len(result.InsertedIDs)}, err
}
