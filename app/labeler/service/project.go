package service

import (
	"context"
	"go-admin/app/labeler/model"
	"go-admin/common/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (svc *LabelerService) CreateProject(ctx context.Context, req model.Project) (model.Project, error) {
	InitObjectID(&req.ID)
	_, err := svc.CollectionProject.InsertOne(ctx, &req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Project{}, err
	}
	return req, nil
}

type SearchProjectReq struct {
	//dto.Pagination
	FolderID primitive.ObjectID `json:"folderId"`
}

func (svc *LabelerService) SearchProject(ctx context.Context, req SearchProjectReq) ([]model.Project, int, error) {
	cursor, err := svc.CollectionProject.
		Find(
			ctx,
			bson.M{"folderId": req.FolderID},
			options.Find().SetSort(bson.D{{"_id", -1}}),
		)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	var projects []model.Project
	if err := cursor.All(ctx, &projects); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return nil, 0, err
	}
	return projects, len(projects), nil
}

type ProjectDetailReq struct {
	ID primitive.ObjectID
}

func (svc *LabelerService) ProjectDetail(ctx context.Context, req ProjectDetailReq) (model.Project, error) {
	var project model.Project
	err := svc.CollectionProject.FindOne(ctx, bson.D{{"_id", req.ID}}).Decode(&project)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Project{}, err
	}
	return project, nil
}

func (svc *LabelerService) UpdateProject(ctx context.Context, req model.Project) (model.Project, error) {
	_, err := svc.CollectionProject.ReplaceOne(ctx, bson.D{{"_id", req.ID}}, &req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Project{}, err
	}
	return req, nil
}

type DeleteProjectReq struct {
	ID primitive.ObjectID
}

type DeleteProjectResp struct {
	DeletedCount int64 `json:"deletedCount"`
}

func (svc *LabelerService) DeleteProject(ctx context.Context, req DeleteProjectReq) (DeleteProjectResp, error) {
	result, err := svc.CollectionProject.DeleteOne(ctx, bson.D{{"_id", req.ID}})
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return DeleteProjectResp{}, err
	}
	return DeleteProjectResp{DeletedCount: result.DeletedCount}, nil
}

type ProjectCountReq struct {
	ID primitive.ObjectID
}

type ProjectCountResp struct {
	Total            int64 `json:"total"`
	UnallocatedLabel int64 `json:"unallocatedLabel"`
	AllocatedLabel   int64 `json:"allocatedLabel"`
	Labeling         int64 `json:"labeling"`
	Submit           int64 `json:"submit"`
	UnallocatedCheck int64 `json:"unallocatedCheck"`
	AllocatedCheck   int64 `json:"allocatedCheck"`
	Checking         int64 `json:"checking"`
	Passed           int64 `json:"passed"`
	Failed           int64 `json:"failed"`
}

func (svc *LabelerService) ProjectCount(ctx context.Context, req ProjectCountReq) (ProjectCountResp, error) {
	pipe := mongo.Pipeline{
		bson.D{
			{
				"$match",
				bson.D{{"projectId", req.ID}},
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
	cursor, err := svc.CollectionTask.Aggregate(ctx, pipe)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return ProjectCountResp{}, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return ProjectCountResp{}, err
	}

	var resp ProjectCountResp
	for _, result := range results {
		count := int64(result["count"].(int32))
		switch result["_id"] {
		case model.TaskStatusAllocate:
			resp.UnallocatedLabel = count
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
		resp.Total += count
	}
	resp.AllocatedCheck = resp.Checking + resp.Passed + resp.Failed
	resp.UnallocatedCheck = resp.Submit
	resp.AllocatedLabel = resp.Total - resp.UnallocatedLabel
	return resp, nil
}
