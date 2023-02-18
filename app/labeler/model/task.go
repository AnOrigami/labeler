package model

type Task struct {
	ID       string
	Name     string
	Status   string
	Document string
	Contents []Content
}

type Content struct {
	ID      string
	Raw     Group
	Results []Group
}

type Group struct {
	ID       string
	Type     string
	Status   string
	Entities []Entity
}

type Entity struct {
	ID        string
	Sentences []Sentence
}

type Sentence struct {
	Text   string
	Source string
	Span   Span
}

type Span struct {
	Left  int
	Right int
}
