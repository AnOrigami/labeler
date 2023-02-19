package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Folder struct {
	ID       primitive.ObjectID `bson:"_id" json:"id"`
	Name     string             `bson:"name" json:"name"`
	ParentID primitive.ObjectID `bson:"parentID" json:"parentID"`
	Children []*Folder          `bson:"-" json:"children,omitempty"`
}
