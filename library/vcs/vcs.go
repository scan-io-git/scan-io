package vcs

type RepositoryParams struct {
	Name     string
	HttpLink string
	SshLink  string
}

type ProjectParams struct {
	Key  string
	Name string
	Link string
}

type ListFuncResult struct {
	Result  []RepositoryParams
	Status  string
	Message string
}

// type error interface {
// 	Error() string
// }
