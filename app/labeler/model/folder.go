package model

type Folder struct {
	ID       string
	Name     string
	Children []Folder
}
