package model

type Schema struct {
	ID         string
	Name       string
	RuleGroups []RuleGroup
	Model      Model
}

type RuleGroup struct {
	Type string
}

type Model struct {
	Source string
	URL    string
}
