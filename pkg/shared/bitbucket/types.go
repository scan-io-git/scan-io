package bitbucket

// Limit for Bitbucket v1 API page response
const (
	maxLimitElements = 2000
	startElement     = 0
	maxCommentLength = 32768
)

var (
	changeTypes = []string{"ADD", "MODIFY"}
)

// Project represents a Bitbucket project with minimal fields relevant to the client.
type Project struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Link string `json:"link"`
}

// Repository represents a Bitbucket repository, including its project container.
type Repository struct {
	Name     string  `json:"name"`
	Project  Project `json:"project"`
	HTTPLink string  `json:"http_link"`
	SSHLink  string  `json:"ssh_link"`
}

// PullRequest defines the basic structure of a pull request within Bitbucket.
type PullRequest struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	State       string    `json:"state"`
	Author      User      `json:"author"`
	SelfLink    string    `json:"self_link"`
	Source      Reference `json:"source"`
	Destination Reference `json:"destination"`
	CreatedDate int64     `json:"created_date"`
	UpdatedDate int64     `json:"updated_date"`
}

// User represents a user in Bitbucket.
type User struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

// Reference represents a repository reference.
type Reference struct {
	ID           string `json:"id"`
	DisplayId    string `json:"display_id"`
	LatestCommit string `json:"latest_commit"`
}

// --------------------------------------------------

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
