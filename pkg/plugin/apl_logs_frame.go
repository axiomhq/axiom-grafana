package plugin

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	axiQuery "github.com/axiomhq/axiom-go/axiom/query"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

var logFieldAliases = map[string]logFieldAlias{
	"timestamp": {canonicalName: "timestamp", priority: 0},
	"_time":     {canonicalName: "timestamp", priority: 1},
	"time":      {canonicalName: "timestamp", priority: 2},
	"_systime":  {canonicalName: "timestamp", priority: 3},

	"body":    {canonicalName: "body"},
	"message": {canonicalName: "body"},
	"msg":     {canonicalName: "body"},
	"content": {canonicalName: "body"},
	"_raw":    {canonicalName: "body"},
	"raw":     {canonicalName: "body"},
	"line":    {canonicalName: "body"},
	"log":     {canonicalName: "body"},
	"text":    {canonicalName: "body"},

	"severity":     {canonicalName: "severity"},
	"level":        {canonicalName: "severity"},
	"lvl":          {canonicalName: "severity"},
	"log.level":    {canonicalName: "severity"},
	"severitytext": {canonicalName: "severity"},

	"id":  {canonicalName: "id"},
	"_id": {canonicalName: "id"},
}

type logFieldAlias struct {
	canonicalName string
	priority      int
}

type logColumn struct {
	index    int
	priority int
}

type aplLogsFrameBuilder struct{}

func (aplLogsFrameBuilder) Build(ctx context.Context, result *axiQuery.Table, opts aplFrameOptions) (*data.Frame, error) {
	logger := log.DefaultLogger.FromContext(ctx)
	columns := logColumns(result.Fields)
	timestampColumns := logTimestampColumns(result.Fields)
	rowCount := traceRowCount(result.Columns)

	timestampField := data.NewField("timestamp", nil, []time.Time{})
	bodyField := data.NewField("body", nil, []string{})
	severityField := data.NewField("severity", nil, []string{})
	idField := data.NewField("id", nil, []string{})
	labelsField := data.NewField("labels", nil, []json.RawMessage{})

	for row := 0; row < rowCount; row++ {
		timestamp, ok := logRowTimestamp(result, timestampColumns, row)
		if !ok {
			logger.Warn("failed to parse log timestamp", "row", row)
		}
		timestampField.Append(timestamp)
		bodyField.Append(logValueString(logColumnValue(result, columns, "body", row)))
		severityField.Append(logValueString(logColumnValue(result, columns, "severity", row)))
		idField.Append(logValueString(logColumnValue(result, columns, "id", row)))
		labelsField.Append(logLabelsValue(result, columns, row))
	}

	frame := data.NewFrame(
		"Logs",
		timestampField,
		bodyField,
		severityField,
		idField,
		labelsField,
	).SetMeta(&data.FrameMeta{
		Type:                   data.FrameTypeLogLines,
		TypeVersion:            data.FrameTypeVersion{0, 0},
		PreferredVisualization: data.VisTypeLogs,
	})

	applyAPLFrameMetadata(frame, opts)
	return frame, nil
}

func fieldsMatchLogs(fields []axiQuery.Field) bool {
	columns := logColumns(fields)
	hasTimestamp := len(logTimestampColumns(fields)) > 0
	_, hasBody := columns["body"]

	return hasTimestamp && hasBody
}

func logColumns(fields []axiQuery.Field) map[string]logColumn {
	columns := make(map[string]logColumn, len(fields))
	for i, field := range fields {
		alias, ok := logFieldAliasForName(field.Name)
		if !ok {
			continue
		}
		column, exists := columns[alias.canonicalName]
		if exists && column.priority <= alias.priority {
			continue
		}
		columns[alias.canonicalName] = logColumn{index: i, priority: alias.priority}
	}

	return columns
}

func logTimestampColumns(fields []axiQuery.Field) []logColumn {
	columns := make([]logColumn, 0, 2)
	for i, field := range fields {
		alias, ok := logFieldAliasForName(field.Name)
		if !ok || alias.canonicalName != "timestamp" {
			continue
		}
		columns = append(columns, logColumn{index: i, priority: alias.priority})
	}

	sort.SliceStable(columns, func(i, j int) bool {
		return columns[i].priority < columns[j].priority
	})

	return columns
}

func logColumnValue(result *axiQuery.Table, columns map[string]logColumn, canonicalName string, row int) any {
	column, ok := columns[canonicalName]
	if !ok || column.index >= len(result.Columns) || row >= len(result.Columns[column.index]) {
		return nil
	}

	return result.Columns[column.index][row]
}

func logRowTimestamp(result *axiQuery.Table, columns []logColumn, row int) (time.Time, bool) {
	for _, column := range columns {
		if column.index >= len(result.Columns) || row >= len(result.Columns[column.index]) {
			continue
		}
		timestamp, ok := logTimestamp(result.Columns[column.index][row])
		if ok {
			return timestamp, true
		}
	}

	return time.Time{}, false
}

func logTimestamp(value any) (time.Time, bool) {
	switch v := value.(type) {
	case time.Time:
		return v, true
	case string:
		t, err := time.Parse(time.RFC3339Nano, v)
		if err == nil {
			return t, true
		}
	}

	return time.Time{}, false
}

func logValueString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	default:
		return stringifyFrameValue(v)
	}
}

func logLabelsValue(result *axiQuery.Table, columns map[string]logColumn, row int) json.RawMessage {
	labels := make(map[string]any)
	for fieldIndex, field := range result.Fields {
		if _, isLogField := logFieldAliasForName(field.Name); isLogField {
			continue
		}
		if fieldIndex >= len(result.Columns) || row >= len(result.Columns[fieldIndex]) {
			continue
		}

		value := result.Columns[fieldIndex][row]
		if value == nil {
			continue
		}
		labels[field.Name] = value
	}

	raw, err := json.Marshal(labels)
	if err != nil {
		return json.RawMessage(`{}`)
	}

	return json.RawMessage(raw)
}

func canonicalLogFieldName(name string) (string, bool) {
	alias, ok := logFieldAliasForName(name)
	return alias.canonicalName, ok
}

func logFieldAliasForName(name string) (logFieldAlias, bool) {
	alias, ok := logFieldAliases[strings.ToLower(name)]
	return alias, ok
}
