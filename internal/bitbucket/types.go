package bitbucket

var (
	// ChangeTypes defines the possible types of changes in a pull request.
	// TODO: move to global config
	ChangeTypes = []string{"ADD", "MODIFY"}
)

// Response wraps API responses that include pagination details.
type Response[T any] struct {
	NextPageStart int  `json:"nextPageStart"`
	IsLastPage    bool `json:"isLastPage"`
	Limit         int  `json:"limit"`
	Size          int  `json:"size"`
	Start         int  `json:"start"`
	Values        []T  `json:"values"`
}

// ErrorList encapsulates potential API error responses.
type ErrorList struct {
	Errors []Error `json:"errors"`
}

// Error provides detailed information about an error occurred during API interactions.
type Error struct {
	Context       string `json:"context"`
	Message       string `json:"message"`
	ExceptionName string `json:"exceptionName"`
}

// Repository represents a repository in Bitbucket, including its project container and metadata.
type Repository struct {
	Slug          string   `json:"slug"`
	ID            int      `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	HierarchyID   string   `json:"hierarchyId"`
	ScmID         string   `json:"scmId"`
	State         string   `json:"state"`
	StatusMessage string   `json:"statusMessage"`
	Forkable      bool     `json:"forkable,omitempty"`
	Project       *Project `json:"project,omitempty"`
	Public        bool     `json:"public"`
	Archived      bool     `json:"archived,omitempty"`
	Links         Links    `json:"links"`
}

// Project represents a project within Bitbucket, providing a container for repositories.
type Project struct {
	Key         string `json:"key"`
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Public      bool   `json:"public,omitempty"`
	Type        string `json:"type"`
	Links       Links  `json:"links"`
}

// Links stores URLs for accessing related resources.
type Links struct {
	Clone []CloneLink `json:"clone,omitempty"`
	Self  []SelfLink  `json:"self,omitempty"`
}

// CloneLink represents a link to clone the repository.
type CloneLink struct {
	Href string `json:"href"`
	Name string `json:"name"`
}

// SelfLink represents a direct link to the resource itself.
type SelfLink struct {
	Href string `json:"href"`
}

// PullRequest defines the basic structure of a pull request within Bitbucket.
type PullRequest struct {
	client        *Client    `json:"-"`
	ID            int        `json:"id"`
	Version       int        `json:"version"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	State         string     `json:"state"`
	Open          bool       `json:"open,omitempty"`
	Closed        bool       `json:"closed,omitempty"`
	CreatedDate   int64      `json:"createdDate"`
	UpdatedDate   int64      `json:"updatedDate"`
	FromReference Reference  `json:"fromRef"`
	ToReference   Reference  `json:"toRef"`
	Locked        bool       `json:"locked,omitempty"`
	Author        *UserData  `json:"author,omitempty"`
	Reviewers     []UserData `json:"reviewers"`
	Participants  []UserData `json:"participants,omitempty"`
	Properties    struct {
		MergeResult       MergeResult `json:"mergeResult"`
		ResolvedTaskCount int         `json:"resolvedTaskCount"`
		OpenTaskCount     int         `json:"openTaskCount"`
	} `json:"properties"`
	Links Links `json:"links"`
}

// Reference represents a specific state or reference point in a repository.
type Reference struct {
	ID           string     `json:"id"`
	DisplayID    string     `json:"displayId"`
	LatestCommit string     `json:"latestCommit"`
	Repository   Repository `json:"repository"`
}

// UserData holds information about a user.
type UserData struct {
	User               User   `json:"user,omitempty"`
	Role               string `json:"role,omitempty"`
	Approved           bool   `json:"approved,omitempty"`
	Status             string `json:"status,omitempty"`
	LastReviewedCommit string `json:"lastReviewedCommit,omitempty"`
}

// User represents a user within Bitbucket.
type User struct {
	Name         string `json:"name,omitempty"`
	EmailAddress string `json:"emailAddress,omitempty"`
	ID           int    `json:"id,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	Active       bool   `json:"active,omitempty"`
	Slug         string `json:"slug,omitempty"`
	Type         string `json:"type,omitempty"`
	Links        Links  `json:"links,omitempty"`
}

// MergeResult encapsulates the result of a merge attempt in a pull request.
type MergeResult struct {
	Outcome string `json:"outcome"`
	Current bool   `json:"current"`
}

// Changes represents a collection of change details within a git context.
type ChangesResponse[T any] struct {
	Response[T]
	FromHash   string             `json:"fromHash"`
	ToHash     string             `json:"toHash"`
	Properties *ChangesProperties `json:"properties"`
}

// Change details a single file change within a repository, providing metadata about the modification.
type Change struct {
	ContentID        string                   `json:"contentId"`
	FromContentID    string                   `json:"fromContentId"`
	Path             *File                    `json:"path,omitempty"`
	Executable       bool                     `json:"executable,omitempty"`
	Type             string                   `json:"type"`
	NodeType         string                   `json:"nodeType"`
	PercentUnchanged int                      `json:"percentUnchanged,omitempty"`
	Links            Links                    `json:"links,omitempty"`
	Properties       *ChangesPropertiesValues `json:"properties"`
}

// ChangesProperties defines properties associated with a set of changes, typically related to the scope.
type ChangesProperties struct {
	ChangeScope string `json:"changeScope"`
}

// ChangesPropertiesValues holds detailed values for properties associated with a single change.
type ChangesPropertiesValues struct {
	ChangeScope string `json:"changeScope"`
}

// File represents a file within a repository showing its structure and metadata.
type File struct {
	Components []string `json:"components"`
	Parent     string   `json:"parent"`
	Name       string   `json:"name"`
	Extension  string   `json:"extension"`
	ToString   string   `json:"toString"`
}
