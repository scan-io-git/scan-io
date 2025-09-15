package shared

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/pflag"

	"github.com/scan-io-git/scan-io/pkg/shared/config"
	"github.com/scan-io-git/scan-io/pkg/shared/files"
)

func ContainsSubstring(target string, substrings []string) bool {
	for _, substring := range substrings {
		if strings.Contains(target, substring) {
			return true
		}
	}
	return false
}

// StructToMap converts a struct to a map[string]string using reflection.
func StructToMap(data interface{}) (map[string]string, error) {
	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected a struct but got %q", val.Kind())
	}

	result := make(map[string]string)
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldName := typ.Field(i).Name
		fieldValue := fmt.Sprintf("%v", field.Interface())
		result[fieldName] = fieldValue
	}

	return result, nil
}

// WriteGenericResult writes the provided result to a JSON file.
func WriteGenericResult(cfg *config.Config, logger hclog.Logger, result GenericLaunchesResult, commandName string) error {
	outputFilePath := fmt.Sprintf("%v/%s.scanio-result", config.GetScanioHome(cfg), commandName)

	resultData, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		return fmt.Errorf("error marshaling the result data: %w", err)
	}

	if err := files.WriteJsonFile(outputFilePath, resultData); err != nil {
		return fmt.Errorf("error writing result to log file: %w", err)
	}
	logger.Info("results saved to file", "path", outputFilePath)

	return nil
}

// IsInList checks if the target string is in the list of strings.
func IsInList(target string, list []string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

// hasFlags checks if any flags have been set.
func HasFlags(flags *pflag.FlagSet) bool {
	hasFlags := false
	flags.VisitAll(func(flag *pflag.Flag) {
		if flag.Changed {
			hasFlags = true
		}
	})
	return hasFlags
}

// PrintResultAsJSON serializes the result as JSON and prints it.
func PrintResultAsJSON(result interface{}) error {
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(resultJSON))
	return nil
}
