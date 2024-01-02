package main

// Limit for Bitbucket v1 API page response
const (
	maxLimitElements = 2000
	startElement     = 0
	maxCommentLength = 32768
)

var (
	changeTypes = []string{"ADD", "MODIFY"}
)

type Changes struct {
	FromHash      string             `json:"fromHash"`
	ToHash        string             `json:"toHash"`
	Properties    *ChangesProperties `json:"properties"`
	Values        []*Change          `json:"values"`
	Size          int                `json:"size"`
	IsLastPage    bool               `json:"isLastPage"`
	Start         int                `json:"start"`
	Limit         int                `json:"limit"`
	NextPageStart *int               `json:"nextPageStart"`
}

type Change struct {
	ContentID        string `json:"contentId"`
	FromContentID    string `json:"fromContentId"`
	Path             *File
	Executable       bool                     `json:"executable"`
	Type             string                   `json:"type"`
	NodeType         string                   `json:"nodeType"`
	PercentUnchanged int                      `json:"percentUnchanged"`
	Properties       *ChangesPropertiesValues `json:"properties"`
}

type ChangesProperties struct {
	ChangeScope string `json:"changeScope"`
}

type ChangesPropertiesValues struct {
	ChangeScope string `json:"changeScope"`
}

type File struct {
	Components []string `json:"components"`
	Parent     string   `json:"parent"`
	Name       string   `json:"name"`
	Extension  string   `json:"extension"`
	ToString   string   `json:"toString"`
}
