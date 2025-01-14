package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go-admin/app/admin/models"
	"go-admin/app/labeler/model"
	"go-admin/common/actions"
	"go-admin/common/dto"
	"go-admin/common/log"
	"go-admin/common/util"
)

type UploadTask5Req struct {
	Tasks5    []model.Task5
	ProjectID primitive.ObjectID
	Name      []string
}

type UploadTask5Resp struct {
	UploadCount int `json:"uploadCount"`
}

func (svc *LabelerService) UploadTask5(ctx context.Context, req UploadTask5Req) (UploadTask5Resp, error) {
	var project5 model.Project5
	if err := svc.CollectionProject5.FindOne(ctx, bson.M{"_id": req.ProjectID}).Decode(&project5); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return UploadTask5Resp{}, errors.New("项目不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask5Resp{}, err
	}
	var folder5 model.Folder
	if err := svc.CollectionFolder5.FindOne(ctx, bson.M{"_id": project5.FolderID}).Decode(&folder5); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return UploadTask5Resp{}, errors.New("文件夹不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask5Resp{}, err
	}
	//判断是否重复上传
	var sessionIDs []string
	var names []string
	var oldTask5s []model.Task5
	filter := bson.M{
		"projectId": req.ProjectID,
	}
	cursor, err := svc.CollectionTask5.Find(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask5Resp{}, err
	}
	if err := cursor.All(ctx, &oldTask5s); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask5Resp{}, err
	}
	//得到老表中sessionID列表
	for _, oneOldTask5 := range oldTask5s {
		sessionIDs = append(sessionIDs, oneOldTask5.Dialog[0].SessionID)
	}
	//req.Task5中并没有文件名，得把req.Name也带过去
	names = repeatingTask5s(sessionIDs, req.Tasks5, req.Name)
	if len(names) > 0 {
		nameString := "上传失败，重复文件："
		for _, name := range names {
			nameString = nameString + name + " "
		}
		//存在任何重复文件直接返回，任何文件都不会上传
		return UploadTask5Resp{}, errors.New(nameString)
	}

	insertTasks := make([]any, len(req.Tasks5))

	for i, oneTask5 := range req.Tasks5 {
		var wordCount int
		for j, oneDialog := range oneTask5.Dialog {
			oneTask5.Dialog[j].Priority = oneDialog.Version * 5
			if len(oneDialog.Actions) != len(oneDialog.ModelOutputs) {
				return UploadTask5Resp{}, errors.New(oneTask5.Name + "数据错误")
			}
			for n, action := range oneDialog.Actions {
				if action.ActionName == "提供思路、心理作业" {
					oneTask5.Dialog[j].Actions[n].ActionListNode = action.SolutionMethod
				} else {
					oneTask5.Dialog[j].Actions[n].ActionListNode = action.ActionName
				}
			}

			oneTask5.Dialog[j].NewAction = oneDialog.Actions
			oneTask5.Dialog[j].NewOutputs = oneDialog.ModelOutputs
			for k, entity := range oneDialog.Entities {
				numString := strconv.Itoa(entity.Num)
				oneTask5.Dialog[j].Entities[k].ClassType = entity.Class + numString + "[" + entity.Type + "]"
			}
			for i2, action := range oneDialog.Actions {
				if len(action.ActionObject) == 0 {
					oneTask5.Dialog[j].Actions[i2].ActionObject = append(oneTask5.Dialog[j].Actions[i2].ActionObject, model.Object{})
				}
			}
			//使用actions.action_object添加entity
			for _, action := range oneDialog.Actions {
				var insertOneEntity = model.EntityOption{}
				insertOneEntity = model.EntityOption{
					ObjectSummary: action.ActionObject[0].ObjectSummary,
					ClassType:     action.ActionObject[0].ObjectName,
				}
				//判断ObjectSummary是否为空，为空直接不添加
				if insertOneEntity.ObjectSummary != "" {
					oneTask5.Dialog[j].Entities = append(oneTask5.Dialog[j].Entities, insertOneEntity)
				}
			}

			uniqueEntities := make(map[string]model.EntityOption)
			for _, v := range oneTask5.Dialog[j].Entities {
				if existingEntity, ok := uniqueEntities[v.ObjectSummary]; ok {
					// 如果已存在相同ID的记录，则比较B字段的值
					if len(v.ClassType) < len(existingEntity.ClassType) {
						uniqueEntities[v.ObjectSummary] = v
					}
				} else {
					uniqueEntities[v.ObjectSummary] = v
				}
			}
			uniqueEntitiesArray := make([]model.EntityOption, 0, len(uniqueEntities))
			for _, v := range uniqueEntities {
				uniqueEntitiesArray = append(uniqueEntitiesArray, v)
			}
			oneTask5.Dialog[j].Entities = uniqueEntitiesArray
			wordCount = utf8.RuneCountInString(oneDialog.UserContent) + utf8.RuneCountInString(oneDialog.BotResponse) + wordCount
		}

		insertTasks[i] = model.Task5{
			ID:          primitive.NewObjectID(),
			Name:        req.Name[i],
			FullName:    folder5.Name + "/" + project5.Name + "/" + req.Name[i],
			ProjectID:   req.ProjectID,
			Status:      model.TaskStatusAllocate,
			Permissions: model.Permissions{},
			UpdateTime:  util.Datetime(time.Now()),
			Dialog:      oneTask5.Dialog,
			WordCount:   wordCount,
		}
	}
	result, err := svc.CollectionTask5.InsertMany(ctx, insertTasks)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return UploadTask5Resp{}, err
	}
	return UploadTask5Resp{UploadCount: len(result.InsertedIDs)}, err
}

type SearchTask5Req = SearchTaskReq

type SearchTask5Resp struct {
	ID             primitive.ObjectID  `json:"id"`
	ProjectID      primitive.ObjectID  `json:"projectId"`
	Name           string              `json:"name"`
	Status         string              `json:"status"`
	Labeler        string              `json:"labeler"`
	Checker        string              `json:"checker"`
	UpdateTime     util.Datetime       `json:"updateTime"`
	SubmittedTime  util.Datetime       `bson:"submittedTime" json:"submittedTime"`
	ApprovedTime   util.Datetime       `bson:"approvedTime" json:"approvedTime"`
	UnsanctionTime util.Datetime       `bson:"unsanctionTime" json:"unsanctionTime"`
	Remark         bool                `json:"remark"`
	RemarkLen      int                 `bson:"remarkLen" json:"remarkLen"`
	WordCount      int                 `bson:"wordCount" json:"wordCount"`
	EditQuantity   int                 `bson:"editQuantity" json:"editQuantity"`
	WorkQuantity   int                 `bson:"workQuantity" json:"workQuantity"`
	Dialog         []model.ContentText `json:"dialog"`
}

func (svc *LabelerService) SearchTask5(ctx context.Context, req SearchTask5Req) ([]SearchTask5Resp, int, error) {
	filter, err := buildFilter(req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	cursor, err := svc.CollectionLabeledTask5.Find(ctx, filter, buildOptions(req))
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	var tasks []model.Task5
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	results := svc.tasksToSearchTask5Resp(ctx, tasks)

	count, err := svc.CollectionLabeledTask5.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	return results, int(count), nil
}

func (svc *LabelerService) tasksToSearchTask5Resp(ctx context.Context, tasks []model.Task5) []SearchTask5Resp {
	ids := make([]string, 0)
	for _, task := range tasks {
		if task.Permissions.Labeler != nil {
			ids = append(ids, task.Permissions.Labeler.ID)
		}
		if task.Permissions.Checker != nil {
			ids = append(ids, task.Permissions.Checker.ID)
		}
	}

	res := make([]SearchTask5Resp, len(tasks))

	var users []models.SysUser
	if len(ids) > 0 {
		db := svc.GormDB.WithContext(ctx).Find(&users).Select("user_id, nick_name").Where("user_id in ?", ids)
		if err := db.Error; err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
		}
	}
	userMap := make(map[string]string)
	for _, v := range users {
		userMap[strconv.Itoa(v.UserId)] = v.NickName
	}
	for i, task := range tasks {
		var labeler, checker string
		if task.Permissions.Labeler != nil {
			labeler = userMap[task.Permissions.Labeler.ID]
		}
		if task.Permissions.Checker != nil {
			checker = userMap[task.Permissions.Checker.ID]
		}
		if task.Remark != "" {
			res[i] = SearchTask5Resp{
				ID:             task.ID,
				ProjectID:      task.ProjectID,
				Name:           task.Name,
				Status:         task.Status,
				Labeler:        labeler,
				Checker:        checker,
				UpdateTime:     task.UpdateTime,
				Dialog:         task.Dialog,
				WordCount:      task.WordCount,
				EditQuantity:   task.EditQuantity,
				WorkQuantity:   task.WorkQuantity,
				Remark:         true,
				RemarkLen:      task.RemarkLen,
				SubmittedTime:  task.SubmittedTime,
				ApprovedTime:   task.ApprovedTime,
				UnsanctionTime: task.UnsanctionTime,
			}
		} else {
			res[i] = SearchTask5Resp{
				ID:             task.ID,
				ProjectID:      task.ProjectID,
				Name:           task.Name,
				Status:         task.Status,
				Labeler:        labeler,
				Checker:        checker,
				UpdateTime:     task.UpdateTime,
				Dialog:         task.Dialog,
				WordCount:      task.WordCount,
				EditQuantity:   task.EditQuantity,
				WorkQuantity:   task.WorkQuantity,
				Remark:         false,
				RemarkLen:      task.RemarkLen,
				SubmittedTime:  task.SubmittedTime,
				ApprovedTime:   task.ApprovedTime,
				UnsanctionTime: task.UnsanctionTime,
			}
		}
	}
	return res
}

type AllocOneTaskReq struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	UserId    string             `json:"-"`
}

func (svc *LabelerService) AllocOneTask5(ctx context.Context, req AllocOneTaskReq) (model.Task5, error) {

	//查询是否有待标注任务，如果有直接返回这个待标注的，不进行新的任务分配
	fileLabeling := bson.M{
		"projectId":              req.ProjectID,
		"status":                 model.TaskStatusLabeling,
		"permissions.labeler.id": req.UserId,
	}
	var oneLabelingTask5 model.Task5
	err := svc.CollectionLabeledTask5.FindOne(ctx, fileLabeling).Decode(&oneLabelingTask5)
	if err == nil {
		// 存在待标注任务，只能有一个待标注任务，所以返回这个存在的待标注任务
		return oneLabelingTask5, errors.New("存在未标注任务")
	}
	if err != mongo.ErrNoDocuments {
		// 查询操作出错
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task5{}, err
	}
	//不存在待标注任务，err==mongo.ErrNoDocuments
	//查询所有存活
	var project5List []model.Project5
	cursor, err := svc.CollectionProject5.Find(ctx, bson.M{})
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task5{}, err
	}
	if err := cursor.All(ctx, &project5List); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task5{}, err
	}
	var project5Ids []primitive.ObjectID
	for _, project5 := range project5List {
		project5Ids = append(project5Ids, project5.ID)
	}

	//当前项目下不存在待标注的，进行新的任务分配
	filterLabeledTask5 := bson.M{
		"permissions.labeler.id": req.UserId,
		//删除project时没有删除project下的task，所以过滤只在存在的project下的task
		"projectId": bson.M{
			"$in": project5Ids,
		},
	}
	cursor, err = svc.CollectionLabeledTask5.Find(ctx, filterLabeledTask5)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task5{}, err
	}
	var tasks []model.Task5
	//在LabeledTask5中查询出当前用户所有标注过的数据，从而得到session的集合
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task5{}, err
	}
	personSessionID := make([]string, 0)
	for _, oneTask5 := range tasks {
		personSessionID = append(personSessionID, oneTask5.Dialog[0].SessionID)
	}

	filter := bson.M{
		"projectId": req.ProjectID,
		"dialog.0.sessionId": bson.M{
			"$nin": personSessionID,
		},
		"dialog.0.priority": bson.M{
			"$gt": 0,
		},
	}
	//priority优先级字段最大的排在最前面
	sortTask := bson.D{{"dialog.0.priority", -1}}

	var resp model.Task5
	optionsTask := options.FindOne().SetSort(sortTask)

	err = svc.CollectionTask5.FindOne(ctx, filter, optionsTask).Decode(&resp)
	if err != mongo.ErrNoDocuments && err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task5{}, err
	} else if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task5{}, errors.New("当前没有可领取的新任务")
	}

	// 更新优先级 更新RequireScore字段
	var newDialog []model.ContentText
	for _, dialog := range resp.Dialog {
		dialog.Priority = dialog.Priority - 1
		newDialog = append(newDialog, dialog)
	}
	updateFilter := bson.M{"_id": resp.ID}
	var update bson.M

	if resp.RequireScore == 1 {
		update = bson.M{"$set": bson.M{"dialog": newDialog, "requireScore": 2}}
	} else {
		update = bson.M{"$set": bson.M{"dialog": newDialog}}
	}

	_, err = svc.CollectionTask5.UpdateOne(ctx, updateFilter, update)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task5{}, err
	}

	//将分配人员信息插入allocTask5
	labeler := &model.Person{
		ID: req.UserId,
	}
	resp.Permissions = model.Permissions{
		Labeler: labeler,
		Checker: nil,
	}

	//为分配出来的task5创建新的ID，以便insert进新表
	resp.ID = primitive.NewObjectID()
	resp.Status = model.TaskStatusLabeling
	for i := range resp.Dialog {
		resp.Dialog[i].UserMessages.UserWant = "无相关信息"
		resp.Dialog[i].UserMessages.UserImportant = "无相关信息"
		resp.Dialog[i].UserMessages.UserAbility = "无相关信息"
	}

	_, err = svc.CollectionLabeledTask5.InsertOne(ctx, resp)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task5{}, err
	}

	//返回的task5是优先级字段没有减去1的
	return resp, nil
}

type ResetTasks5Req struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Persons   []string           `json:"persons"`
	Statuses  []string           `json:"statuses"`
	ResetType int64              `json:"resetType"`
}

type ResetTasks5Resp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) ResetTasks5(ctx context.Context, req ResetTasks5Req) (ResetTasks5Resp, error) {
	filter := bson.M{}
	filter["projectId"] = req.ProjectID
	filter["status"] = bson.M{
		"$in": req.Statuses,
	}

	if req.ResetType == 0 {
		filter["permissions.labeler.id"] = bson.M{
			"$in": req.Persons,
		}
		update := bson.M{
			"$set": bson.M{
				"permissions": model.Permissions{},
				"status":      model.TaskStatusAllocate,
				"updateTime":  util.Datetime(time.Now()),
			},
		}
		result, err := svc.CollectionTask5.UpdateMany(ctx, filter, update)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return ResetTasks5Resp{}, err
		}
		return ResetTasks5Resp{Count: result.ModifiedCount}, nil
	} else {
		filter["permissions.checker.id"] = bson.M{
			"$in": req.Persons,
		}
		update := bson.M{
			"$set": bson.M{
				"permissions.checker": model.Person{},
				"status":              model.TaskStatusSubmit,
				"updateTime":          util.Datetime(time.Now()),
			},
		}
		result, err := svc.CollectionTask5.UpdateMany(ctx, filter, update)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return ResetTasks5Resp{}, err
		}
		return ResetTasks5Resp{Count: result.ModifiedCount}, nil
	}
}

type UpdateTask5Req struct {
	UserID        string              `json:"-"`
	UserDataScope string              `json:"-"`
	ID            primitive.ObjectID  `json:"id"`
	Remark        string              `json:"remark"`
	RemarkOptions int                 `bson:"remarkOptions" json:"remarkOptions"`
	Dialog        []model.ContentText `json:"dialog"`
	Score         model.Scores        `json:"score"`
	HasScore      bool                `json:"hasScore"`
}

func (svc *LabelerService) UpdateTask5(ctx context.Context, req UpdateTask5Req) (model.Task5, error) {
	var task model.Task5

	//req.Score大于5或小于0不合法
	socreType := reflect.TypeOf(req.Score)
	scoreValue := reflect.ValueOf(req.Score)

	for i := 0; i < socreType.NumField(); i++ {
		scoreValue := scoreValue.Field(i).Int()

		if scoreValue > 5 || scoreValue < 0 {
			err := errors.New("评分不合法")
			log.Logger().WithContext(ctx).Error(err.Error())
			return model.Task5{}, err
		}
	}

	if err := svc.CollectionLabeledTask5.FindOne(ctx, bson.M{"_id": req.ID}).Decode(&task); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Task5{}, errors.New("任务不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task5{}, err
	}
	if req.UserDataScope != "1" && req.UserDataScope != "2" && !task.Permissions.IsLabeler(req.UserID) && !task.Permissions.IsChecker(req.UserID) {
		return model.Task5{}, errors.New("权限不足")
	}
	var editQuantity int
	for i, oneDialog := range req.Dialog {
		for j, action := range oneDialog.NewAction {
			if in(action.ActionListNode, specialNodesList) {
				req.Dialog[i].NewAction[j].ActionName = "提供思路、心理作业"
				req.Dialog[i].NewAction[j].SolutionMethod = action.ActionListNode
			} else {
				req.Dialog[i].NewAction[j].ActionName = action.ActionListNode
				req.Dialog[i].NewAction[j].SolutionMethod = ""
			}
			req.Dialog[i].NewOutputs[j].Action = req.Dialog[i].NewAction[j].ActionName
		}
		var newContent, content []string
		for _, v := range oneDialog.ModelOutputs {
			content = append(content, v.Content)
		}
		for _, v := range oneDialog.NewOutputs {
			newContent = append(newContent, v.Content)
		}
		newResultStr := strings.Join(newContent, "")
		resultStr := strings.Join(content, "")
		editQuantity = editDistance(resultStr, newResultStr) + editQuantity
		editQuantity = editDistance(task.Dialog[i].UserMessages.UserWant, oneDialog.UserMessages.UserWant) + editQuantity
		editQuantity = editDistance(task.Dialog[i].UserMessages.UserImportant, oneDialog.UserMessages.UserImportant) + editQuantity
		editQuantity = editDistance(task.Dialog[i].UserMessages.UserAbility, oneDialog.UserMessages.UserAbility) + editQuantity
	}
	runeRemark := []rune(req.Remark)
	remarkLen := len(runeRemark)
	workQuantity := task.WordCount + (editQuantity+remarkLen)*2
	task.Dialog = req.Dialog
	task.UpdateTime = util.Datetime(time.Now())
	update := bson.M{
		"$set": bson.M{
			"editQuantity":  editQuantity,
			"remark":        req.Remark,
			"remarkLen":     remarkLen,
			"remarkOptions": req.RemarkOptions,
			"score":         req.Score,
			"dialog":        task.Dialog,
			"updateTime":    task.UpdateTime,
			"hasScore":      req.HasScore,
			"workQuantity":  workQuantity,
		},
	}
	if _, err := svc.CollectionLabeledTask5.UpdateByID(ctx, req.ID, update); err != nil {
		log.Logger().WithContext(ctx).Error("update task: ", err.Error())
		return model.Task5{}, err
	}
	if err := svc.CollectionLabeledTask5.FindOne(ctx, bson.M{"_id": req.ID}).Decode(&task); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Task5{}, errors.New("任务不存在")
		}
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Task5{}, err
	}
	return task, nil
}

func in(target string, strArray []string) bool {
	sort.Strings(strArray)
	index := sort.SearchStrings(strArray, target)
	if index < len(strArray) && strArray[index] == target {
		return true
	}
	return false
}

var specialNodesList = []string{
	"运动", "音乐疗法", "画画", "家庭关系图", "其他绘画类", "激烈运动", "呐喊", "蹦迪", "极限运动", "唱歌",
	"撕纸", "其他情绪宣泄", "正念", "冥想", "催眠", "呼吸", "肌肉放松", "阅读", "书法", "茶道",
	"花道", "生命意义、人生价值", "香道", "陶艺", "自我暗示", "亲密关系支持", "兴趣爱好小组", "专业性支持", "其他社会性支持", "模拟练习",
	"其他提供思路、心理作业", "现实类问题", "过往经历思考类", "对未来", "当下发生", "过往经历书写类"}

type BatchSetTask5StatusReq struct {
	UserID        string               `json:"-"`
	UserDataScope string               `json:"-"`
	IDs           []primitive.ObjectID `json:"ids"`
	Status        string               `json:"status"`
	WorkType      int64                `json:"workType"`
}

type BatchSetTask5StatusResp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) BatchSetTask5Status(ctx context.Context, req BatchSetTask5StatusReq) (BatchSetTask5StatusResp, error) {
	if len(req.IDs) == 0 {
		return BatchSetTask5StatusResp{}, errors.New("什么也没有发生")
	}
	filter := bson.M{
		"_id": bson.M{
			"$in": req.IDs,
		},
	}
	if req.WorkType != 0 {
		filter["$or"] = bson.A{
			bson.M{"permissions.labeler.id": req.UserID},
			bson.M{"permissions.checker.id": req.UserID},
		}
	}
	normalStatusMap := map[string][]string{
		model.TaskStatusFailed:   {model.TaskStatusChecking, model.TaskStatusPassed, model.TaskStatusFailed},
		model.TaskStatusPassed:   {model.TaskStatusChecking, model.TaskStatusPassed, model.TaskStatusFailed},
		model.TaskStatusChecking: {model.TaskStatusSubmit, model.TaskStatusFailed},
		model.TaskStatusSubmit:   {model.TaskStatusLabeling, model.TaskStatusSubmit},
	}
	specialStatusMap := map[string][]string{
		model.TaskStatusFailed:   {model.TaskStatusChecking, model.TaskStatusPassed, model.TaskStatusFailed},
		model.TaskStatusPassed:   {model.TaskStatusChecking, model.TaskStatusPassed, model.TaskStatusFailed},
		model.TaskStatusChecking: {},
		model.TaskStatusSubmit:   {},
	}

	//任务状态为{待标注}，管理员点击进入之后为标注页面
	//任务状态为{已提交}，管理员点击进入之后为标注页面

	//任务状态为{待审核}，管理员点击进入之后为审核页面，点击审核通过之后任务状态变更为已审核，点击审核不通过之后任务状态变更为审核不通过
	//任务状态为{已审核}，管理员点击进入之后为审核页面，点击审核通过之后任务状态变更为已审核，点击审核不通过之后任务状态变更为审核不通过
	//任务状态为{审核不通过}，管理员点击进入之后为审核页面，点击审核通过之后任务状态变更为已审核，点击审核不通过之后任务状态变更为审核不通过
	////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//任务状态为{待标注}，标注员点击进入之后为标注页面，点击提交之后任务状态变更为已提交
	//任务状态为{已提交}，标注员点击进入之后为标注页面，点击提交之后任务状态变更为已提交

	//任务状态为{待审核}，审核员点击进入之后为审核页面，点击审核通过之后任务状态变更为已审核，点击审核不通过之后任务状态变更为审核不通过
	//任务状态为{已审核}，审核员点击进入之后为审核页面，点击审核通过之后任务状态变更为已审核，点击审核不通过之后任务状态变更为审核不通过
	//任务状态为{审核不通过}，审核员点击进入之后为审核页面，点击审核通过之后任务状态变更为已审核，点击审核不通过之后任务状态变更为审核不通过
	//                    标注员点击进入之后为标注页面，点击提交之后任务状态变更为待审核

	update := bson.M{}
	if req.Status == model.TaskStatusSubmit {
		update = bson.M{
			"$set": bson.M{
				"status":        req.Status,
				"updateTime":    util.Datetime(time.Now()),
				"submittedTime": util.Datetime(time.Now()),
			},
		}
	} else if req.Status == model.TaskStatusFailed {
		update = bson.M{
			"$set": bson.M{
				"status":         req.Status,
				"updateTime":     util.Datetime(time.Now()),
				"unsanctionTime": util.Datetime(time.Now()),
			},
		}
	} else if req.Status == model.TaskStatusPassed {
		update = bson.M{
			"$set": bson.M{
				"status":       req.Status,
				"updateTime":   util.Datetime(time.Now()),
				"approvedTime": util.Datetime(time.Now()),
			},
		}
	} else {
		update = bson.M{
			"$set": bson.M{
				"status":     req.Status,
				"updateTime": util.Datetime(time.Now()),
			},
		}
	}

	if req.WorkType != 0 {
		if validSourceStatus := normalStatusMap[req.Status]; validSourceStatus != nil {
			filter["status"] = bson.M{
				"$in": validSourceStatus,
			}
		}
	} else {
		if validSourceStatus := specialStatusMap[req.Status]; validSourceStatus != nil {
			filter["status"] = bson.M{
				"$in": validSourceStatus,
			}
		}
	}

	result, err := svc.CollectionLabeledTask5.UpdateMany(ctx, filter, update)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return BatchSetTask5StatusResp{}, err
	}
	if int(result.ModifiedCount) < len(req.IDs) {
		if req.Status == model.TaskStatusSubmit {
			return BatchSetTask5StatusResp{}, errors.New("提交失败：任务已被分配审核")
		}
		return BatchSetTask5StatusResp{}, errors.New("部分任务状态没有修改")
	}
	return BatchSetTask5StatusResp{Count: result.ModifiedCount}, err
}

type SearchMyTask5Req struct {
	ID       primitive.ObjectID `json:"id"`
	UserID   string
	Status   []string `json:"status"`
	TaskType string   `json:"taskType"`
	dto.Pagination
}

func (svc *LabelerService) SearchMyTask5(ctx context.Context, req SearchMyTask5Req) ([]SearchTask5Resp, int, error) {
	filter := bson.M{
		"projectId": req.ID,
	}
	if len(req.Status) != 0 {
		filter["status"] = bson.M{
			"$in": req.Status,
		}
	}
	if req.TaskType == "标注" {
		filter["permissions.labeler.id"] = req.UserID
	} else if req.TaskType == "审核" {
		filter["permissions.checker.id"] = req.UserID
	} else {
		filter["$or"] = []bson.M{
			{"permissions.labeler.id": req.UserID},
			{"permissions.checker.id": req.UserID},
		}
	}
	cursor, err := svc.CollectionLabeledTask5.Find(ctx, filter, buildOptions5(req))
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	var tasks []model.Task5
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	results := svc.tasksToSearchTask5Resp(ctx, tasks)

	count, err := svc.CollectionLabeledTask5.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}

	return results, int(count), nil
}

func buildOptions5(req SearchMyTask5Req) *options.FindOptions {
	if req.PageIndex < 1 {
		req.PageIndex = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 10
	}
	opts := options.Find().
		SetLimit(int64(req.PageSize)).
		SetSkip(int64((req.PageIndex - 1) * req.PageSize))
	return opts
}

func (svc *LabelerService) DeleteTask5(ctx context.Context, id primitive.ObjectID) error {

	var delTask5 model.Task5
	err := svc.CollectionLabeledTask5.FindOne(ctx, bson.M{"_id": id}).Decode(&delTask5)
	if err != nil {
		log.Logger().WithContext(ctx).Error("find delete task: ", err.Error())
		return err
	}
	if delTask5.Status == model.TaskStatusLabeling {
		var oldTask5 model.Task5
		err := svc.CollectionTask5.FindOne(ctx, bson.M{
			"projectId":          delTask5.ProjectID,
			"dialog.0.sessionId": delTask5.Dialog[0].SessionID}).Decode(&oldTask5)
		if err != nil {
			log.Logger().WithContext(ctx).Error("find delete task in old table: ", err.Error())
			return err
		}
		for i := range oldTask5.Dialog {
			oldTask5.Dialog[i].Priority += 1
		}

		updateFilter := bson.M{
			"_id": oldTask5.ID,
		}
		update := bson.M{"$set": bson.M{"dialog": oldTask5.Dialog}}

		_, err = svc.CollectionTask5.UpdateOne(ctx, updateFilter, update)
		if err != nil {
			log.Logger().WithContext(ctx).Error("update task priority: ", err.Error())
			return err
		}
	}
	if _, err := svc.CollectionLabeledTask5.DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		log.Logger().WithContext(ctx).Error("delete task: ", err.Error())
		return err
	}
	return nil
}

type DownloadTask5Req struct {
	ProjectID          primitive.ObjectID `json:"projectId"`
	Status             []string           `json:"status"`
	UpdateTimeStart    string             `json:"updateTimeStart"`
	UpdateTimeEnd      string             `json:"updateTimeEnd"`
	SubmittedTimeStart string             `json:"submittedTimeStart"`
	SubmittedTimeEnd   string             `json:"submittedTimeEnd"`
}

type DownloadTask5Resp struct {
	File     *string `json:"file"`
	FileName string  `json:"filename"`
}

func (svc *LabelerService) DownloadTask5(ctx context.Context, req DownloadTask5Req) (DownloadTask5Resp, error) {
	filter := bson.M{
		"projectId": req.ProjectID,
		"status": bson.M{
			"$in": req.Status,
		},
	}

	if len(req.UpdateTimeStart) > 0 && len(req.UpdateTimeEnd) > 0 {
		startTime, err := time.Parse(util.TimeLayoutDatetime, req.UpdateTimeStart)
		if err != nil {
			return DownloadTask5Resp{}, ErrTimeParse
		}
		endTime, err := time.Parse(util.TimeLayoutDatetime, req.UpdateTimeEnd)
		if err != nil {
			return DownloadTask5Resp{}, ErrTimeParse
		}
		filter["updateTime"] = bson.M{
			"$gte": startTime,
			"$lte": endTime,
		}
	}

	if len(req.SubmittedTimeStart) > 0 && len(req.SubmittedTimeEnd) > 0 {
		startTime, err := time.Parse(util.TimeLayoutDatetime, req.SubmittedTimeStart)
		if err != nil {
			return DownloadTask5Resp{}, ErrTimeParse
		}
		endTime, err := time.Parse(util.TimeLayoutDatetime, req.SubmittedTimeEnd)
		if err != nil {
			return DownloadTask5Resp{}, ErrTimeParse
		}
		filter["submittedTime"] = bson.M{
			"$gte": startTime,
			"$lte": endTime,
		}
	}

	cursor, err := svc.CollectionLabeledTask5.Find(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask5Resp{}, err
	}

	var tasks []*model.Task5
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask5Resp{}, err
	}

	nameStr := time.Now().Format("2006-01-02 15-04-05") + "下载文件.zip"
	buf := new(bytes.Buffer)

	zipWriter := zip.NewWriter(buf)

	for _, task := range tasks {
		data, err := json.Marshal(task)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTask5Resp{}, err
		}
		taskName := strings.Split(task.Name, ".")
		w1, err := zipWriter.Create(taskName[0] + ".json")
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTask5Resp{}, err
		}
		_, err = w1.Write(data)
		if err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return DownloadTask5Resp{}, err
		}
	}
	if err = zipWriter.Close(); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask5Resp{}, err
	}

	res := base64.StdEncoding.EncodeToString(buf.Bytes())
	return DownloadTask5Resp{File: &res, FileName: nameStr}, nil
}

type GetTask5Req struct {
	ID              primitive.ObjectID `json:"id"`
	Name            string             `json:"name"`
	Labeler         string             `json:"labeler"`
	Checker         string             `json:"checker"`
	UpdateTimeStart string             `json:"updateTimeStart"`
	UpdateTimeEnd   string             `json:"updateTimeEnd"`
	Status          []string           `json:"status"`
	WorkType        int64              `json:"workType"`
}

type GetTask5Resp struct {
	Last primitive.ObjectID `json:"last"`
	Next primitive.ObjectID `json:"next"`
	model.Task5
}

func (svc *LabelerService) GetTask5(ctx context.Context, req GetTask5Req, p *actions.DataPermission) (GetTask5Resp, error) {
	var task model.Task5
	var users []models.SysUser
	filter := bson.M{}
	ids := make([]string, 0)
	userID := strconv.Itoa(p.UserId)
	if err := svc.CollectionLabeledTask5.FindOne(ctx, bson.D{{"_id", req.ID}}).Decode(&task); err != nil {
		if err == mongo.ErrNoDocuments {
			return GetTask5Resp{}, ErrNoDoc
		}
		log.Logger().WithContext(ctx).Error("get task: ", err.Error())
		return GetTask5Resp{}, err
	}
	if req.WorkType == 1 {
		if !task.Permissions.IsLabeler(userID) {
			return GetTask5Resp{}, errors.New("任务已被撤回/删除，请刷新任务列表重新进入")
		}
		filter = bson.M{
			"projectId":              task.ProjectID,
			"permissions.labeler.id": userID,
			"status": bson.M{
				"$in": req.Status,
			},
		}
	}
	if req.WorkType == 2 {
		if !task.Permissions.IsChecker(userID) {
			return GetTask5Resp{}, errors.New("任务已被撤回/删除，请刷新任务列表重新进入")
		}
		filter = bson.M{
			"projectId":              task.ProjectID,
			"permissions.checker.id": userID,
			"status": bson.M{
				"$in": req.Status,
			},
		}
	}
	if req.WorkType == 0 {

		filter = buildTask5DetailFilter(req)
		filter["projectId"] = task.ProjectID
	}
	if task.Permissions.Labeler != nil {
		ids = append(ids, task.Permissions.Labeler.ID)
	}
	if task.Permissions.Checker != nil {
		ids = append(ids, task.Permissions.Checker.ID)
	}
	if len(ids) > 0 {
		db := svc.GormDB.WithContext(ctx).Find(&users).Select("user_id, nick_name").Where("user_id in ?", ids)
		if err := db.Error; err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
		}
	}
	userMap := make(map[string]string)
	for _, v := range users {
		userMap[strconv.Itoa(v.UserId)] = v.NickName
	}
	if task.Permissions.Labeler != nil {
		task.Permissions.Labeler.NickName = userMap[task.Permissions.Labeler.ID]
	}
	if task.Permissions.Checker != nil {
		task.Permissions.Checker.NickName = userMap[task.Permissions.Checker.ID]
	}

	var nextTask model.Task5
	var lastTask model.Task5
	filter["_id"] = bson.M{"$gt": task.ID}
	if err := svc.CollectionLabeledTask5.FindOne(ctx, filter).Decode(&nextTask); err != nil {
		if err != mongo.ErrNoDocuments {
			log.Logger().WithContext(ctx).Error("get task: ", err.Error())
			return GetTask5Resp{}, err
		}
	}

	filter["_id"] = bson.M{"$lt": task.ID}
	option := options.FindOne().SetSort(bson.M{"_id": -1})
	if err := svc.CollectionLabeledTask5.FindOne(ctx, filter, option).Decode(&lastTask); err != nil {
		if err != mongo.ErrNoDocuments {
			log.Logger().WithContext(ctx).Error("get task: ", err.Error())
			return GetTask5Resp{}, err
		}
	}
	res := GetTask5Resp{
		Task5: task,
	}
	res.Last = lastTask.ID
	res.Next = nextTask.ID
	return res, nil
}

func buildTask5DetailFilter(req GetTask5Req) bson.M {
	filter := bson.M{}

	if len(req.Status) > 0 {
		filter["status"] = bson.M{
			"$in": req.Status,
		}
	}

	if len(req.Labeler) > 0 {
		filter["permissions.labeler.id"] = req.Labeler
	}
	if len(req.Checker) > 0 {
		filter["permissions.checker.id"] = req.Checker
	}
	if len(req.Name) > 0 {
		filter["name"] = bson.M{
			"$regex": req.Name,
		}
	}
	if len(req.UpdateTimeStart) > 0 {
		t, err := time.Parse(util.TimeLayoutDatetime, req.UpdateTimeStart)
		if err != nil {
			return nil
		}
		filter["updateTime"] = bson.M{
			"$gte": t,
		}
	}
	if len(req.UpdateTimeEnd) > 0 {
		t, err := time.Parse(util.TimeLayoutDatetime, req.UpdateTimeEnd)
		if err != nil {
			return nil
		}
		value, ok := filter["updateTime"]
		if ok {
			value.(bson.M)["$lte"] = t
			filter["updateTime"] = value
		}
	}
	return filter
}

type SearchMyTask5CountReq struct {
	ID       primitive.ObjectID `json:"id"`
	UserID   string
	TaskType string `json:"taskType"`
}

type SearchMyTask5CountRes struct {
	Labeling int64 `json:"labeling"`
	Submit   int64 `json:"submit"`
	Checking int64 `json:"checking"`
	Passed   int64 `json:"passed"`
	Failed   int64 `json:"failed"`
}

func (svc *LabelerService) SearchMyTask5Count(ctx context.Context, req SearchMyTask5CountReq) (SearchMyTask5CountRes, error) {
	filter := bson.M{
		"projectId": req.ID,
	}
	if req.TaskType == "标注" {
		filter["permissions.labeler.id"] = req.UserID
	} else {
		filter["permissions.checker.id"] = req.UserID
	}
	pipe := mongo.Pipeline{
		bson.D{
			{
				"$match",
				filter,
			},
		},
		bson.D{
			{
				"$group",
				bson.D{
					{"_id", "$status"},
					{"count", bson.D{{"$sum", 1}}},
				},
			},
		},
	}
	cursor, err := svc.CollectionTask5.Aggregate(ctx, pipe)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return SearchMyTask5CountRes{}, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return SearchMyTask5CountRes{}, err
	}

	var resp SearchMyTask5CountRes
	for _, result := range results {
		count := int64(result["count"].(int32))
		switch result["_id"] {
		case model.TaskStatusLabeling:
			resp.Labeling = count
		case model.TaskStatusSubmit:
			resp.Submit = count
		case model.TaskStatusChecking:
			resp.Checking = count
		case model.TaskStatusPassed:
			resp.Passed = count
		case model.TaskStatusFailed:
			resp.Failed = count
		}
	}
	return resp, nil
}

type DownloadScoreReq struct {
	Version   []int              `json:"version"`
	ProjectID primitive.ObjectID `json:"projectId"`
}

func (svc *LabelerService) DownloadScore(ctx context.Context, req DownloadScoreReq) (DownloadTask2Resp, error) {

	filter := bson.M{}
	if len(req.Version) == 0 {
		filter = bson.M{
			"projectId": req.ProjectID,
			"hasScore":  true,
		}
	} else {
		filter = bson.M{
			"projectId": req.ProjectID,
			"dialog.0.version": bson.M{
				"$in": req.Version,
			},
			"hasScore": true,
		}
	}

	cursor, err := svc.CollectionLabeledTask5.Find(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask2Resp{}, err
	}
	var task5 []model.Task5
	if err := cursor.All(ctx, &task5); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask2Resp{}, err
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask2Resp{}, err
	}
	currentTime := time.Now().In(loc)
	currentTimeString := currentTime.Format("2006-01-02 15:04:05")
	ubotVsersion := ""
	for i := 0; i < len(req.Version); i++ {
		if i == len(req.Version)-1 {
			ubotVsersion = ubotVsersion + strconv.Itoa(req.Version[i])
		}
		ubotVsersion = ubotVsersion + strconv.Itoa(req.Version[i]) + ","
	}
	nameStr := currentTimeString + "-Ubot:" + ubotVsersion + "-" + "打分情况"

	idList := make([]string, len(task5))
	for _, task := range task5 {
		idList = append(idList, task.Permissions.Labeler.ID)
	}

	var users []models.SysUser
	db := svc.GormDB.WithContext(ctx).Where("user_id IN (?)", idList).
		Find(&users)

	nicknameList := make(map[string]string, len(users))
	for _, user := range users {
		nicknameList[strconv.Itoa(user.UserId)] = user.NickName
	}

	if err := db.Error; err != nil {
		return DownloadTask2Resp{}, err
	}

	excleData := getTask5ScoreExcle(task5, nicknameList)
	data, filename, err := util.EmbedExcelData(
		nameStr,
		excleData,
		ctx,
	)
	if err != nil {
		return DownloadTask2Resp{}, err
	}
	return DownloadTask2Resp{File: data, FileName: filename}, nil

}

type DownloadWorkloadReq struct {
	PersonList         []string           `json:"personList"`
	WordCount          bool               `json:"wordCount"`
	EditQuantity       bool               `json:"editQuantity"`
	RemarkQuantity     bool               `json:"remarkQuantity"`
	WorkQuantity       bool               `json:"workQuantity"`
	TaskStatus         string             `json:"taskStatus"`
	UpdateTimeStart    string             `json:"updateTimeStart"`
	UpdateTimeEnd      string             `json:"updateTimeEnd"`
	ProjectID          primitive.ObjectID `json:"projectId"`
	SubmittedTimeStart string             `json:"submittedTimeStart"`
	SubmittedTimeEnd   string             `json:"submittedTimeEnd"`
}

func (svc *LabelerService) DownloadWorkload(ctx context.Context, req DownloadWorkloadReq) (DownloadTask2Resp, error) {

	//req.bool全为false时变为全为true
	if req.RemarkQuantity == false && req.WordCount == false && req.EditQuantity == false && req.WorkQuantity == false {
		req.RemarkQuantity = true
		req.WordCount = true
		req.EditQuantity = true
		req.WorkQuantity = true
	}

	var users []models.SysUser
	db := svc.GormDB.WithContext(ctx).Where("user_id IN (?)", req.PersonList).
		Find(&users)
	if err := db.Error; err != nil {
		return DownloadTask2Resp{}, err
	}

	var userMap = make(map[int]string, len(users))
	for _, user := range users {
		userMap[user.UserId] = user.NickName
	}

	filter := bson.M{
		"status": req.TaskStatus,
		"permissions.labeler.id": bson.M{
			"$in": req.PersonList,
		},
		"projectId": req.ProjectID,
	}

	if len(req.UpdateTimeStart) > 0 && len(req.UpdateTimeEnd) > 0 {
		startTime, err := time.Parse(util.TimeLayoutDatetime, req.UpdateTimeStart)
		if err != nil {
			return DownloadTask2Resp{}, ErrTimeParse
		}
		endTime, err := time.Parse(util.TimeLayoutDatetime, req.UpdateTimeEnd)
		if err != nil {
			return DownloadTask2Resp{}, ErrTimeParse
		}
		filter["updateTime"] = bson.M{
			"$gte": startTime,
			"$lte": endTime,
		}
	}

	if len(req.SubmittedTimeStart) > 0 && len(req.SubmittedTimeEnd) > 0 {
		startTime, err := time.Parse(util.TimeLayoutDatetime, req.SubmittedTimeStart)
		if err != nil {
			return DownloadTask2Resp{}, ErrTimeParse
		}
		endTime, err := time.Parse(util.TimeLayoutDatetime, req.SubmittedTimeEnd)
		if err != nil {
			return DownloadTask2Resp{}, ErrTimeParse
		}
		filter["submittedTime"] = bson.M{
			"$gte": startTime,
			"$lte": endTime,
		}
	}

	pipe := mongo.Pipeline{
		bson.D{
			{
				"$match",
				filter,
			},
		},
		bson.D{
			{
				"$group",
				bson.D{
					{"_id", "$permissions.labeler.id"},
					//{"permissions_labeler_id", bson.D{{"$first", "$permissions.labeler.id"}}},
					{"remarkLen", bson.D{{"$sum", "$remarkLen"}}},
					{"wordCount", bson.D{{"$sum", "$wordCount"}}},
					{"editQuantity", bson.D{{"$sum", "$editQuantity"}}},
					{"workQuantity", bson.D{{"$sum", "$workQuantity"}}},
				},
			},
		},
	}

	cursor, err := svc.CollectionLabeledTask5.Aggregate(ctx, pipe)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask2Resp{}, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DownloadTask2Resp{}, err
	}

	nameStr := ""
	if len(req.UpdateTimeStart) > 0 {
		nameStr += "StartTime:" + req.UpdateTimeStart
	}
	if len(req.UpdateTimeEnd) > 0 {
		nameStr += "EndTime:" + req.UpdateTimeEnd
	}
	nameStr += "DownloadTime:"

	columns := []string{"咨询师", "阅读量", "修改量", "点评量", "工作量"}
	excelData := getTask5WorkExcle(results, userMap, req)

	data, filename, err := util.CreateExcelFile(
		excelData,
		columns,
		nameStr,
	)
	if err != nil {
		return DownloadTask2Resp{}, err
	}
	return DownloadTask2Resp{File: data, FileName: filename}, nil
}

type ProportionalScoringReq struct {
	ProjectID  primitive.ObjectID `json:"projectID"`
	Version    []int              `json:"version"`
	Proportion float32            `json:"proportion"`
}

type ProportionalScoringResp struct {
	MatchedCount  int64 `json:"matchedCount"`
	ModifiedCount int64 `json:"modifiedCount"`
}

func (svc *LabelerService) ProportionalScoring(ctx context.Context, req ProportionalScoringReq) (ProportionalScoringResp, error) {

	if req.Proportion > 1 || req.Proportion <= 0 {
		return ProportionalScoringResp{}, errors.New("概率必须大于0小于等于1")
	}

	var resp ProportionalScoringResp

	filter := bson.M{
		"projectId": req.ProjectID,
		"dialog.0.version": bson.M{
			"$in": req.Version,
		},
		"requireScore": 0,
	}

	var err error
	resp.MatchedCount, err = svc.CollectionTask5.CountDocuments(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return ProportionalScoringResp{}, err
	}

	resp.ModifiedCount = int64(float32(resp.MatchedCount) * req.Proportion)

	pipeline := mongo.Pipeline{
		bson.D{
			{
				"$match",
				filter,
			},
		},
		bson.D{
			{
				"$sample",
				bson.D{
					{"size", resp.ModifiedCount},
				},
			},
		},
	}
	var modifyTask []model.Task5

	cursor, err := svc.CollectionTask5.Aggregate(ctx, pipeline)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return ProportionalScoringResp{}, err
	}
	err = cursor.All(ctx, &modifyTask)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return ProportionalScoringResp{}, err
	}
	modifyIdList := make([]primitive.ObjectID, len(modifyTask))
	for _, task := range modifyTask {
		modifyIdList = append(modifyIdList, task.ID)
	}
	modifyFilter := bson.M{
		"_id": bson.M{
			"$in": modifyIdList,
		},
	}

	update := bson.D{
		{"$set", bson.D{
			{"requireScore", 1},
		}},
	}
	_, err = svc.CollectionTask5.UpdateMany(context.Background(), modifyFilter, update)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return ProportionalScoringResp{}, err
	}
	return resp, nil

}

func getTask5WorkExcle(results []bson.M, user map[int]string, req DownloadWorkloadReq) [][]interface{} {
	var data [][]interface{}
	for _, result := range results {
		s := []interface{}{}
		userId, _ := strconv.Atoi(result["_id"].(string))
		s = append(s, user[userId])
		if req.WordCount {
			wordCount, _ := result["wordCount"].(int32)
			s = append(s, wordCount)
		} else {
			s = append(s, "")
		}
		if req.EditQuantity {
			editQuantity, _ := result["editQuantity"].(int32)
			s = append(s, editQuantity)
		} else {
			s = append(s, "")
		}
		if req.RemarkQuantity {
			remarkQuantity, _ := result["remarkLen"].(int32)
			s = append(s, remarkQuantity)
		} else {
			s = append(s, "")
		}
		if req.WorkQuantity {
			workQuantity, _ := result["workQuantity"].(int32)
			s = append(s, workQuantity)
		} else {
			s = append(s, "")
		}
		data = append(data, s)
	}
	return data
}

func getTask5ScoreExcle(task5 []model.Task5, nicknameList map[string]string) [][]interface{} {
	var data [][]interface{}
	for _, task := range task5 {

		s := []interface{}{}

		s = append(s, task.Dialog[0].Version)
		s = append(s, task.Name)
		s = append(s, nicknameList[task.Permissions.Labeler.ID])
		s = append(s, task.Score.IdentifyRisk)
		s = append(s, task.Score.UnderstandingVisitor)
		s = append(s, task.Score.ExpressingCare)
		s = append(s, task.Score.AcceptFeedback)
		s = append(s, task.Score.AdvanceProcess)
		s = append(s, task.Score.PromoteProcess)
		s = append(s, task.Score.InspirationAssistance)
		s = append(s, task.Score.RelieveEmotions)
		s = append(s, task.Score.VisitorFeedback)
		if task.RemarkOptions == 1 {
			s = append(s, "及格")
		}
		if task.RemarkOptions == 2 {
			s = append(s, "不及格")
		}
		data = append(data, s)
	}
	return data
}

type Node struct {
	Value    string `json:"value"`
	Children []Node `json:"children"`
}

var ActionTags = []Node{
	{
		Value: "提问",
		Children: []Node{
			{
				Value: "短焦问句",
				Children: []Node{
					{
						Value:    "提问对来访者来说满分是什么样的",
						Children: nil,
					},
					{
						Value:    "提问如果ta做出改变的话，来访者会有什么不同",
						Children: nil,
					},
					{
						Value:    "以`结果问句`的方式进行提问",
						Children: nil,
					},
					{
						Value:    "以`量尺问句`的方式进行提问",
						Children: nil,
					},
					{
						Value:    "以`例外问句`的方式进行提问",
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
						Value:    "以`假如问句`的方式进行提问",
						Children: nil,
					},
					{
						Value:    "以`差异问句`的方式进行提问",
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
						Value:    "提问来访者当下的情绪或感受",
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
						Value:    "提问来访者最近的生活情况",
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
					{
						Value:    "提问来访者做某件事的理由",
						Children: nil,
					},
					{
						Value:    "提问来访者对某人物的看法/评价？",
						Children: nil,
					},
					{
						Value:    "提问来访者的自我评价",
						Children: nil,
					},
					{
						Value:    "提问来访者的行为/反应",
						Children: nil,
					},
					{
						Value:    "提问历史",
						Children: nil,
					},
					{
						Value:    "提问表现",
						Children: nil,
					},
					{
						Value:    "提问关系",
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
				Value:    "总结",
				Children: nil,
			},
			{
				Value:    "共情/同理",
				Children: nil,
			},
			{
				Value:    "澄清",
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
			{
				Value:    "使用关键词",
				Children: nil,
			},
			{
				Value:    "确认",
				Children: nil,
			},
			{
				Value:    "语意重复",
				Children: nil,
			},
			{
				Value:    "重构",
				Children: nil,
			},
			{
				Value:    "温和面质",
				Children: nil,
			},
			{
				Value:    "达成一致性理解",
				Children: nil,
			},
			{
				Value:    "比喻",
				Children: nil,
			},
			{
				Value:    "隐喻",
				Children: nil,
			},
		},
	},
	{
		Value:    "解释、分析",
		Children: nil,
	},
	{
		Value: "提供思路、心理作业",
		Children: []Node{
			{
				Value: "思考类",
				Children: []Node{
					{
						Value:    "生命意义、人生价值",
						Children: nil,
					},
					{
						Value:    "现实类的问题",
						Children: nil,
					},
					{
						Value:    "过往经历思考类",
						Children: nil,
					},
				},
			},
			{
				Value: "书写类",
				Children: []Node{
					{
						Value:    "对未来",
						Children: nil,
					},
					{
						Value:    "当下发生",
						Children: nil,
					},
					{
						Value:    "过往经历书写类",
						Children: nil,
					},
				},
			},
			{
				Value: "行为类",
				Children: []Node{
					{
						Value:    "运动",
						Children: nil,
					},
					{
						Value:    "音乐疗法",
						Children: nil,
					},
					{
						Value: "绘画类",
						Children: []Node{
							{
								Value:    "画画",
								Children: nil,
							},
							{
								Value:    "家庭关系图",
								Children: nil,
							},
							{
								Value:    "其他绘画类",
								Children: nil,
							},
						},
					},
					{
						Value: "情绪宣泄",
						Children: []Node{
							{
								Value:    "激烈运动",
								Children: nil,
							},
							{
								Value:    "呐喊",
								Children: nil,
							},
							{
								Value:    "蹦迪",
								Children: nil,
							},
							{
								Value:    "极限运动",
								Children: nil,
							},
							{
								Value:    "唱歌",
								Children: nil,
							},
							{
								Value:    "撕纸",
								Children: nil,
							},
							{
								Value:    "其他情绪宣泄",
								Children: nil,
							},
						},
					},
					{
						Value: "身心疗愈",
						Children: []Node{
							{
								Value:    "正念",
								Children: nil,
							},
							{
								Value:    "冥想",
								Children: nil,
							},
							{
								Value:    "催眠",
								Children: nil,
							},
							{
								Value:    "呼吸",
								Children: nil,
							},
							{
								Value:    "肌肉放松",
								Children: nil,
							},
						},
					},
					{
						Value: "艺术疗愈",
						Children: []Node{
							{
								Value:    "阅读",
								Children: nil,
							},
							{
								Value:    "书法",
								Children: nil,
							},
							{
								Value:    "茶道",
								Children: nil,
							},
							{
								Value:    "花道",
								Children: nil,
							},
							{
								Value:    "香道",
								Children: nil,
							},
							{
								Value:    "陶艺",
								Children: nil,
							},
						},
					},
					{
						Value:    "自我暗示",
						Children: nil,
					},
					{
						Value: "寻求他人帮助",
						Children: []Node{
							{
								Value:    "亲密关系支持",
								Children: nil,
							},
							{
								Value:    "兴趣爱好小组",
								Children: nil,
							},
							{
								Value:    "专业性支持",
								Children: nil,
							},
							{
								Value:    "其他社会性支持",
								Children: nil,
							},
						},
					},
					{
						Value:    "模拟练习",
						Children: nil,
					},
				},
			},
			{
				Value:    "其他提供思路、心理作业",
				Children: nil,
			},
		},
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
		Value:    "其他动作",
		Children: nil,
	},
}

func minThree(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= a && b <= c {
		return b
	}
	return c
}

func editDistance(s1, s2 string) int {
	r1 := []rune(s1)
	r2 := []rune(s2)

	m := len(r1)
	n := len(r2)

	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for j := 0; j <= n; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if r1[i-1] == r2[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				dp[i][j] = minThree(dp[i-1][j], dp[i][j-1]+1, dp[i-1][j-1]+1)
			}
		}
	}

	return dp[m][n]
}

func repeatingTask5s(sessionIDs []string, task5s []model.Task5, filename []string) []string {
	names := make([]string, 0)
	for i, oneTask5 := range task5s {
		for _, s := range sessionIDs {
			if s == oneTask5.Dialog[0].SessionID {
				names = append(names, filename[i])
				break
			}
		}
	}
	return names
}

type Req struct {
	ProjectID primitive.ObjectID `json:"projectId"`
}

func (svc *LabelerService) SearchTask5Count(ctx context.Context, req Req) (int, error) {
	filter := bson.M{
		"projectId": req.ProjectID,
	}

	cursor, err := svc.CollectionLabeledTask5.Find(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return 0, err
	}

	var tasks []model.Task5
	if err := cursor.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return 0, err
	}
	var updateCount int
	for _, task := range tasks {
		var isUpdate bool
		parts := strings.Split(task.Name, ".")
		sessionId := parts[0]
		for j, v := range task.Dialog {
			if v.SessionID == "" {
				task.Dialog[j].SessionID = sessionId
				isUpdate = true
			} else {
				break
			}
		}
		update := bson.M{
			"$set": bson.M{
				"dialog": task.Dialog,
			},
		}
		if isUpdate {
			if _, err := svc.CollectionLabeledTask5.UpdateByID(ctx, task.ID, update); err != nil {
				log.Logger().WithContext(ctx).Error("update task: ", err.Error())
				return 0, err
			}
			updateCount = updateCount + 1
		}
	}

	return updateCount, nil
}

type Task5BatchAllocCheckerReq struct {
	ProjectID primitive.ObjectID `json:"projectId"`
	Persons   []string           `json:"persons"`
	Number    int64              `json:"number"`
}

type Task5BatchAllocCheckerResp struct {
	Count int64 `json:"count"`
}

func (svc *LabelerService) Task5BatchAllocChecker(ctx context.Context, req Task5BatchAllocCheckerReq) error {
	if req.Number <= 0 {
		return errors.New("分配任务数量不合法")
	}
	if len(req.Persons) == 0 {
		return errors.New("分配人员数量不能为0")
	}
	filter := bson.M{
		"projectId": req.ProjectID,
		"status":    model.TaskStatusSubmit,
	}
	maxCount := int(req.Number) / len(req.Persons)
	if maxCount < 1 {
		maxCount = 1
	}
	personMap := make(map[string]int, len(req.Persons))
	for _, id := range req.Persons {
		personMap[id] = 0
	}

	result, err := svc.CollectionLabeledTask5.Find(ctx, filter)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return err
	}
	var tasks []model.Task5
	if err = result.All(ctx, &tasks); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return err
	}
	if len(tasks) == 0 {
		return errors.New("当前无可分配任务")
	}
	nowTime := util.Datetime(time.Now())
	var totalCount int
	for _, task := range tasks {
		var minCount = math.MaxInt
		var minID string
		for i, v := range personMap {
			if task.Permissions.Labeler != nil {
				if i == task.Permissions.Labeler.ID {
					continue
				}
			}
			if v == maxCount {
				continue
			}
			if v < minCount {
				minCount = v
				minID = i
			}
		}
		if minID == "" {
			continue
		}
		personMap[minID]++
		ft := bson.M{
			"_id": task.ID,
		}
		update := bson.M{
			"$set": bson.M{
				"permissions.checker": model.Person{ID: minID},
				"status":              model.TaskStatusChecking,
				"updateTime":          nowTime,
			},
		}
		if _, err := svc.CollectionLabeledTask5.UpdateOne(ctx, ft, update); err != nil {
			log.Logger().WithContext(ctx).Error(err.Error())
			return err
		}
		totalCount++
		if totalCount == int(req.Number) {
			break
		}
	}
	if totalCount == 0 {
		return errors.New("分配失败：标注员和审核员不能是同一人")
	}
	return nil
}
