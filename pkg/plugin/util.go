package plugin

import (
	"encoding/json"
	"fmt"
	"time"
)

func stringPtr(value string) *string {
	return &value
}

func nullableStringPtr(value string) *string {
	if value == "" {
		return nil
	}

	return &value
}

func stringifyFrameValue(value any) string {
	if value == nil {
		return ""
	}

	b, err := json.Marshal(value)
	if err == nil {
		return string(b)
	}

	return fmt.Sprintf("%v", value)
}

func inferUnknownFieldType(fieldName string, column []any) string {
	hasValue := false
	allFloat := true
	allBool := true
	allString := true
	allDatetime := true
	allArray := true

	for _, value := range column {
		switch v := value.(type) {
		case nil:
			continue
		case float64:
			hasValue = true
			allBool = false
			allString = false
			allDatetime = false
			allArray = false
		case bool:
			hasValue = true
			allFloat = false
			allString = false
			allDatetime = false
			allArray = false
		case string:
			hasValue = true
			allFloat = false
			allBool = false
			allArray = false
			if _, err := time.Parse(time.RFC3339Nano, v); err != nil {
				allDatetime = false
			}
		case []any, []string, []float64:
			hasValue = true
			allFloat = false
			allBool = false
			allString = false
			allDatetime = false
		default:
			hasValue = true
			allFloat = false
			allBool = false
			allString = false
			allDatetime = false
			allArray = false
		}
	}

	if !hasValue {
		return "string"
	}
	if allFloat {
		return "float"
	}
	if allBool {
		return "bool"
	}
	if allString {
		if allDatetime || fieldName == "_time" {
			return "datetime"
		}

		return "string"
	}
	if allArray {
		return "array"
	}

	return "string"
}
