package service

import (
	"context"
	"go-admin/app/labeler/model"
	"go-admin/common/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (svc *LabelerService) GetFolders(ctx context.Context) ([]*model.Folder, error) {
	cursor, err := svc.CollectionFolder.Find(ctx, bson.D{}, options.Find().SetSort(bson.D{{"createTime", -1}}))
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

func FolderTree(folders []*model.Folder) []*model.Folder {
	var tree []*model.Folder
	foldersMap := make(map[primitive.ObjectID]*model.Folder, len(folders))
	for _, f := range folders {
		if f.ParentID == nil {
			tree = append(tree, f)
		}

		foldersMap[f.ID] = f
	}

	for _, f := range folders {
		if f.ParentID != nil {
			if parent, exists := foldersMap[*f.ParentID]; exists {
				parent.Children = append(parent.Children, f)
			}
		}
	}

	return tree
}

func (svc *LabelerService) CreateFolder(ctx context.Context, req model.Folder) (model.Folder, error) {
	InitObjectID(&req.ID)
	if _, err := svc.CollectionFolder.InsertOne(ctx, req); err != nil {
		log.Logger().WithContext(ctx).Error("create folder: ", err.Error())
		return model.Folder{}, ErrDatabase
	}

	return req, nil
}

func (svc *LabelerService) UpdateFolder(ctx context.Context, req model.Folder) (model.Folder, error) {
	data := bson.M{"$set": bson.M{"name": req.Name}}
	if _, err := svc.CollectionFolder.UpdateByID(ctx, req.ID, data); err != nil {
		log.Logger().WithContext(ctx).Error("update folder: ", err.Error())
		return model.Folder{}, ErrDatabase
	}

	return req, nil
}

func (svc *LabelerService) DeleteFolder(ctx context.Context, id primitive.ObjectID) error {
	if _, err := svc.CollectionFolder.DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		log.Logger().WithContext(ctx).Error("delete folder: ", err.Error())
		return ErrDatabase
	}

	log.Logger().WithContext(ctx).Warnf("delete folder:%s", id.Hex())

	return nil
}
