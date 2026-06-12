package plugin

import (
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

const (
	aplTimePriorityTime = iota
	aplTimePriorityTimestamp
	aplTimePriorityTimeAlias
	aplTimePrioritySysTime
	aplTimePriorityUnknownTime
)

func aplTimeFieldPriority(name string) (int, bool) {
	switch strings.ToLower(name) {
	case "_time":
		return aplTimePriorityTime, true
	case "timestamp":
		return aplTimePriorityTimestamp, true
	case "time":
		return aplTimePriorityTimeAlias, true
	case "_systime":
		return aplTimePrioritySysTime, true
	default:
		return 0, false
	}
}

func aplDataFrameTimeFieldPriority(field *data.Field) (int, bool) {
	if priority, ok := aplTimeFieldPriority(field.Name); ok {
		return priority, true
	}

	switch field.Type() {
	case data.FieldTypeTime, data.FieldTypeNullableTime:
		return aplTimePriorityUnknownTime, true
	default:
		return 0, false
	}
}

func preferredAPLTimeFieldIndex(fields []*data.Field) int {
	bestIndex := -1
	bestPriority := aplTimePriorityUnknownTime + 1

	for i, field := range fields {
		priority, ok := aplDataFrameTimeFieldPriority(field)
		if !ok {
			continue
		}
		if priority < bestPriority {
			bestIndex = i
			bestPriority = priority
		}
	}

	return bestIndex
}
