package sarif

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/owenrumney/go-sarif/v2/sarif"
	"github.com/scan-io-git/scan-io/internal/git"
)

// MessageFormatOptions contains the configuration needed to format SARIF messages with GitHub links
type MessageFormatOptions struct {
	Namespace    string
	Repository   string
	Ref          string
	SourceFolder string
}

// FormatResultMessage is the main entry point for formatting SARIF result messages
// It processes the message template, substitutes arguments, and converts location references to GitHub hyperlinks
func FormatResultMessage(result *sarif.Result, repoMetadata *git.RepositoryMetadata, options MessageFormatOptions) string {
	// Extract locations for reference lookup
	locations := extractLocationsForFormatting(result)
	if len(locations) == 0 {
		// No locations available, return plain text message
		if result.Message.Text != nil {
			return *result.Message.Text
		}
		return ""
	}

	// Check if this is a CodeQL-style message (direct markdown in text field)
	if result.Message.Text != nil && result.Message.Markdown == nil && len(result.Message.Arguments) == 0 {
		return formatCodeQLStyleMessage(*result.Message.Text, result, repoMetadata, options)
	}

	// Format the message with arguments and location links (Snyk style)
	formatted := formatMessageWithArguments(&result.Message, locations, repoMetadata, options)
	if formatted != "" {
		return formatted
	}

	// Fallback to plain text
	if result.Message.Text != nil {
		return *result.Message.Text
	}

	return ""
}

// formatCodeQLStyleMessage handles CodeQL-style messages where the text contains direct markdown links
// Example: "This template construction depends on a [user-provided value](1)."
func formatCodeQLStyleMessage(text string, result *sarif.Result, repoMetadata *git.RepositoryMetadata, options MessageFormatOptions) string {
	// Pattern to match [text](id) where id is a number
	pattern := regexp.MustCompile(`\[([^\]]+)\]\((\d+)\)`)

	return pattern.ReplaceAllStringFunc(text, func(match string) string {
		matches := pattern.FindStringSubmatch(match)
		if len(matches) != 3 {
			return match // Return original if pattern doesn't match
		}

		linkText := matches[1]
		idStr := matches[2]

		// Convert id to integer (CodeQL uses 1-based indexing)
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return match // Return original if id is not a number
		}

		// Find the relatedLocation with matching id
		var targetLocation *sarif.Location
		for _, relLoc := range result.RelatedLocations {
			if relLoc.Id != nil && *relLoc.Id == uint(id) {
				// Create a Location from RelatedLocation
				targetLocation = &sarif.Location{
					PhysicalLocation: relLoc.PhysicalLocation,
				}
				break
			}
		}

		if targetLocation != nil {
			link := buildLocationLink(targetLocation, repoMetadata, options)
			if link != "" {
				return fmt.Sprintf("[%s](%s)", linkText, link)
			}
		}

		// If we can't build a link, return the original text without the reference
		return linkText
	})
}

// extractLocationsForFormatting extracts locations from SARIF result in priority order:
// 1) relatedLocations, 2) codeFlows[0].threadFlows[0].locations, 3) empty array
func extractLocationsForFormatting(result *sarif.Result) []*sarif.Location {
	var locations []*sarif.Location

	// Priority 1: relatedLocations
	if len(result.RelatedLocations) > 0 {
		for _, relLoc := range result.RelatedLocations {
			if relLoc != nil {
				locations = append(locations, relLoc)
			}
		}
		return locations
	}

	// Priority 2: codeFlows[0].threadFlows[0].locations
	if len(result.CodeFlows) > 0 && len(result.CodeFlows[0].ThreadFlows) > 0 {
		threadFlow := result.CodeFlows[0].ThreadFlows[0]
		for _, threadLoc := range threadFlow.Locations {
			if threadLoc.Location != nil {
				locations = append(locations, threadLoc.Location)
			}
		}
		return locations
	}

	// Priority 3: empty array (fallback)
	return locations
}

// formatMessageWithArguments processes the message template, substitutes placeholders, and converts location references
func formatMessageWithArguments(message *sarif.Message, locations []*sarif.Location, repoMetadata *git.RepositoryMetadata, options MessageFormatOptions) string {
	// Use markdown template if available, otherwise fall back to text
	template := ""
	if message.Markdown != nil {
		template = *message.Markdown
	} else if message.Text != nil {
		template = *message.Text
	} else {
		return ""
	}

	// If no arguments, return template as-is
	if len(message.Arguments) == 0 {
		return template
	}

	// Process each argument and substitute placeholders
	result := template
	for i, arg := range message.Arguments {
		placeholder := fmt.Sprintf("{%d}", i)

		// Parse the argument to extract text and location references
		text, refs := parseLocationReference(arg)

		// Convert location references to hyperlinks
		formattedArg := formatLocationReferences(text, refs, locations, repoMetadata, options)

		// Substitute the placeholder
		result = strings.ReplaceAll(result, placeholder, formattedArg)
	}

	return result
}

// parseLocationReference parses SARIF message arguments to extract text and location reference numbers
// Examples:
//
//	"[user-provided value](1)" -> text="user-provided value", refs=[1]
//	"[flows](1),(2),(3),(4),(5),(6)" -> text="flows", refs=[1,2,3,4,5,6]
func parseLocationReference(arg string) (text string, refs []int) {
	// Pattern to match [text](ref1),(ref2),...
	// This handles both single references and multiple references
	pattern := regexp.MustCompile(`^\[([^\]]+)\]\((.+)\)$`)
	matches := pattern.FindStringSubmatch(arg)

	if len(matches) != 3 {
		// Malformed argument, return as-is
		return arg, nil
	}

	text = matches[1]
	refsStr := matches[2]

	// Parse reference numbers (handle both single and multiple)
	// The format is like "1),(2),(3),(4),(5),(6" - we need to extract numbers
	refParts := strings.Split(refsStr, "),(")
	for _, part := range refParts {
		part = strings.TrimSpace(part)
		// Remove any remaining parentheses
		part = strings.Trim(part, "()")
		if refNum, err := strconv.Atoi(part); err == nil {
			refs = append(refs, refNum)
		}
	}

	return text, refs
}

// formatLocationReferences converts location reference numbers to GitHub hyperlinks
func formatLocationReferences(text string, refs []int, locations []*sarif.Location, repoMetadata *git.RepositoryMetadata, options MessageFormatOptions) string {
	if len(refs) == 0 {
		return text
	}

	// Build links for each reference
	var links []string
	for _, ref := range refs {
		if ref >= 0 && ref < len(locations) {
			link := buildLocationLink(locations[ref], repoMetadata, options)
			if link != "" {
				links = append(links, fmt.Sprintf("[%d](%s)", ref, link))
			} else {
				links = append(links, fmt.Sprintf("%d", ref))
			}
		} else {
			// Invalid reference, use as-is
			links = append(links, fmt.Sprintf("%d", ref))
		}
	}

	// Format based on number of references
	if len(refs) == 1 {
		// Single reference: "[text](link)"
		if refs[0] >= 0 && refs[0] < len(locations) {
			link := buildLocationLink(locations[refs[0]], repoMetadata, options)
			if link != "" {
				return fmt.Sprintf("[%s](%s)", text, link)
			} else {
				return fmt.Sprintf("%s (%d)", text, refs[0])
			}
		} else {
			return fmt.Sprintf("%s (%d)", text, refs[0])
		}
	} else {
		// Multiple references: "text ([1](link1) > [2](link2) > ...)"
		linkChain := strings.Join(links, " > ")
		return fmt.Sprintf("%s (%s)", text, linkChain)
	}
}

// buildLocationLink constructs a GitHub permalink for a SARIF location
func buildLocationLink(location *sarif.Location, repoMetadata *git.RepositoryMetadata, options MessageFormatOptions) string {
	if location.PhysicalLocation == nil || location.PhysicalLocation.ArtifactLocation == nil {
		return ""
	}

	artifact := location.PhysicalLocation.ArtifactLocation
	if artifact.URI == nil {
		return ""
	}

	// Get file path and convert to repository-relative path
	filePath := *artifact.URI
	repoPath := ConvertToRepoRelativePath(filePath, repoMetadata, options.SourceFolder)

	// Get line information
	region := location.PhysicalLocation.Region
	if region == nil {
		return ""
	}

	startLine := 1
	endLine := 1

	if region.StartLine != nil {
		startLine = *region.StartLine
	}
	if region.EndLine != nil {
		endLine = *region.EndLine
	} else {
		endLine = startLine
	}

	// Build GitHub permalink
	// Format: https://github.com/{namespace}/{repo}/blob/{ref}/{file}#L{start}-L{end}
	baseURL := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s",
		options.Namespace, options.Repository, options.Ref, repoPath)

	if startLine == endLine {
		return fmt.Sprintf("%s#L%d", baseURL, startLine)
	} else {
		return fmt.Sprintf("%s#L%d-L%d", baseURL, startLine, endLine)
	}
}
