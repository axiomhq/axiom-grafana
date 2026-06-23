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

func firstDebugValue(column []any) (any, int, bool) {
	for i, value := range column {
		if value != nil {
			return value, i, true
		}
	}
	if len(column) > 0 {
		return column[0], 0, true
	}

	return nil, -1, false
}

func debugValueType(value any) string {
	if value == nil {
		return "<nil>"
	}

	return fmt.Sprintf("%T", value)
}

func debugValuePreview(value any) (preview string) {
	defer func() {
		if r := recover(); r != nil {
			preview = fmt.Sprintf("<failed to render value: %v>", r)
		}
	}()

	if value == nil {
		return "<nil>"
	}

	b, err := json.Marshal(value)
	if err != nil {
		preview = fmt.Sprintf("%v", value)
	} else {
		preview = string(b)
	}
	if len(preview) > 512 {
		return preview[:512] + "...(truncated)"
	}

	return preview
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
