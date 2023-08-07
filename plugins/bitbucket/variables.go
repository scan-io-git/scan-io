package main

// Limit for Bitbucket v1 API page response
const (
	maxLimitElements = 2000
	startElement     = 0
)

// Known change scopes in PR
const (
	ChangeScopeUnreviewed = "UNREVIEWED"
)

// Known change types in PR
const (
	ChangeTypeAdd     = "ADD"     // A new file was added.
	ChangeTypeCopy    = "COPY"    //An existing file was copied to create a new file.
	ChangeTypeDelete  = "DELETE"  // An existing file was deleted.
	ChangeTypeModify  = "MODIFY"  // 	An existing file was modified.
	ChangeTypeMove    = "MOVE"    // An existing file was moved to a new path.
	ChangeTypeUnknown = "UNKNOWN" // An SCM-specific change has occurred for which there is no known generic alias.
)

// Known segment types in PR diff
const (
	DiffSegmentTypeAdded   = "ADDED"   // Indicates the lines in the segment were added in the destination file.
	DiffSegmentTypeContext = "CONTEXT" // 	Indicates the lines in the segment are context, existing unchanged in both the source and destination.
	DiffSegmentTypeRemoved = "REMOVED" // 	Indicates the lines in the segment were removed in the destination file.
)

// Known node types in PR
const (
	NodeTypeFile = "FILE"
)

// Known user roles in PR
const (
	RoleReviewer = "REVIEWER"
)

// Known approval statuses
const (
	StatusApproved   = "APPROVED"
	StatusNeedsWork  = "NEEDS_WORK"
	StatusUnapproved = "UNAPPROVED"
)

const (
	maxCommentLength = 32768
)
