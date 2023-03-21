package model

import (
	"go-admin/common/util"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Folder struct {
	ID         primitive.ObjectID  `bson:"_id" json:"id"`
	Name       string              `bson:"name" json:"name"`
	ParentID   *primitive.ObjectID `bson:"parentId,omitempty" json:"parentId,omitempty"`
	Children   []*Folder           `bson:"-" json:"children,omitempty"`
	CreateTime util.Datetime       `bson:"createTime,omitempty" json:"createTime,omitempty"`
}
