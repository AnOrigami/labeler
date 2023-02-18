package service

import (
	"context"
	"go-admin/app/labeler/model"
	"go-admin/common/log"
)

func (svc *LabelerService) CreateProject(ctx context.Context, req model.Project) (model.Project, error) {
	InitObjectID(&req.ID)
	_, err := svc.CollectionProject.InsertOne(ctx, req)
	if err != nil {
		log.Logger().WithContext(ctx).Error(err.Error())
		return model.Project{}, err
	}
	return req, nil
}
