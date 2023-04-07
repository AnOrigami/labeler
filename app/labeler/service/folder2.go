package service

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"go-admin/app/labeler/model"
	"go-admin/common/log"
)

func (svc *LabelerService) GetFolders2(ctx context.Context) ([]*model.Folder, error) {
	cursor, err := svc.CollectionFolder2.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{"createTime", -1}}))
	if err != nil {
		log.Logger().WithContext(ctx).Error("get folders: ", err.Error())
		return nil, ErrDatabase
	}

	var folders []*model.Folder
	if err = cursor.All(ctx, &folders); err != nil {
		log.Logger().WithContext(ctx).Error("get folders: ", err.Error())
		return nil, ErrDatabase
	}

	return FolderTree(folders), nil
}

func (svc *LabelerService) CreateFolder2(ctx context.Context, req model.Folder) (model.Folder, error) {
	InitObjectID(&req.ID)
	if _, err := svc.CollectionFolder2.InsertOne(ctx, req); err != nil {
		log.Logger().WithContext(ctx).Error("create folder: ", err.Error())
		return model.Folder{}, ErrDatabase
	}

	return req, nil
}

func (svc *LabelerService) UpdateFolder2(ctx context.Context, req model.Folder) (model.Folder, error) {
	data := bson.M{"$set": bson.M{"name": req.Name}}
	if _, err := svc.CollectionFolder2.UpdateByID(ctx, req.ID, data); err != nil {
		log.Logger().WithContext(ctx).Error("update folder: ", err.Error())
		return model.Folder{}, ErrDatabase
	}

	return req, nil
}

func (svc *LabelerService) DeleteFolder2(ctx context.Context, id primitive.ObjectID) error {
	if _, err := svc.CollectionFolder2.DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		log.Logger().WithContext(ctx).Error("delete folder: ", err.Error())
		return ErrDatabase
	}

	log.Logger().WithContext(ctx).Warnf("delete folder:%s", id.Hex())

	return nil
}
