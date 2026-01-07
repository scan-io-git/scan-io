package sarifissues

import (
	"fmt"
	"strings"
)

// validate validates the RunOptions for the sarif-issues command.
func validate(o *RunOptions) error {
	if o.Namespace == "" {
		return fmt.Errorf("--namespace is required")
	}
	if o.Repository == "" {
		return fmt.Errorf("--repository is required")
	}
	if strings.TrimSpace(o.SarifPath) == "" {
		return fmt.Errorf("--sarif is required")
	}
	return nil
}
