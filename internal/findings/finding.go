package findings

// Property is a simple name/value pair used for tags, references, or custom metadata.
type Property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CodeFlowStep represents a single step in a data flow.
type CodeFlowStep struct {
	FilePath    string `json:"file_path"`
	StartLine   int    `json:"start_line"`
	StartColumn int    `json:"start_column"`
	EndLine     int    `json:"end_line"`
	EndColumn   int    `json:"end_column"`
	Message     string `json:"message,omitempty"`
}

// CodeFlow contains a sequence of steps describing a flow.
type CodeFlow struct {
	Steps []CodeFlowStep `json:"steps"`
}

// Finding is a minimal internal domain model extracted from SARIF or other scanners.
type Finding struct {
	RuleID      string `json:"rule_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Scanner     string `json:"scanner"`

	FilePath  string `json:"file_path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`

	Tags       []Property `json:"tags,omitempty"`
	References []Property `json:"references,omitempty"`
	Properties []Property `json:"properties,omitempty"`

	CodeFlows []CodeFlow `json:"code_flows,omitempty"`
}
