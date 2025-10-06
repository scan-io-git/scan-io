package cmd

// Mode constants
const (
	ModeSingleURL = "single-url"
	ModeFlags     = "flags"
)

// determineMode determines the mode based on the provided arguments.
func DetermineMode(args []string) string {
	if len(args) > 0 {
		return ModeSingleURL
	}
	return ModeFlags
}
