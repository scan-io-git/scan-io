package shared

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/pflag"
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
