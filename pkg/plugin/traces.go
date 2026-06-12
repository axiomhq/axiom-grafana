package plugin

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	axiQuery "github.com/axiomhq/axiom-go/axiom/query"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

var requiredTraceFields = map[string]struct{}{
	"traceID":       {},
	"spanID":        {},
	"operationName": {},
	"serviceName":   {},
	"startTime":     {},
	"duration":      {},
}

var traceFieldAliases = map[string]string{
	"traceID":  "traceID",
	"traceId":  "traceID",
	"trace_id": "traceID",
	"trace.id": "traceID",
	"traceid":  "traceID",

	"spanID":  "spanID",
	"spanId":  "spanID",
	"span_id": "spanID",
	"span.id": "spanID",
	"spanid":  "spanID",

	"parentSpanID":   "parentSpanID",
	"parentSpanId":   "parentSpanID",
	"parent_span_id": "parentSpanID",
	"parent.span.id": "parentSpanID",

	"name":           "operationName",
	"operationName":  "operationName",
	"operation_name": "operationName",
	"span.name":      "operationName",

	"serviceName":           "serviceName",
	"service_name":          "serviceName",
	"service.name":          "serviceName",
	"resource.service.name": "serviceName",

	"_time":       "startTime",
	"startTime":   "startTime",
	"start_time":  "startTime",
	"start.time":  "startTime",
	"timestamp":   "startTime",
	"timestampNs": "startTime",

	"duration":    "duration",
	"durationMs":  "duration",
	"duration_ms": "duration",
	"durationNs":  "duration",
	"duration_ns": "duration",

	"serviceTags":  "serviceTags",
	"service_tags": "serviceTags",
	"tags":         "tags",
	"attributes":   "tags",
	"logs":         "logs",
	"events":       "logs",
}

type traceColumn struct {
	index int
	name  string
}

type aplTraceFrameBuilder struct{}

func (aplTraceFrameBuilder) Build(ctx context.Context, result *axiQuery.Table, opts aplFrameOptions) (*data.Frame, error) {
	frame, err := buildTraceFrame(ctx, result)
	if err != nil {
		return nil, err
	}
	applyAPLFrameMetadata(frame, opts)
	return frame, nil
}

func fieldsMatchTrace(ctx context.Context, fields []axiQuery.Field) bool {
	found := make(map[string]struct{}, len(requiredTraceFields))
	for _, field := range fields {
		canonicalName, ok := traceFieldAliases[field.Name]
		if !ok {
			continue
		}
		found[canonicalName] = struct{}{}
	}

	for requiredField := range requiredTraceFields {
		if _, ok := found[requiredField]; !ok {
			return false
		}
	}

	return true
}

func buildTraceFrame(ctx context.Context, result *axiQuery.Table) (*data.Frame, error) {
	logger := log.DefaultLogger.FromContext(ctx)
	columns := traceColumns(result.Fields)
	rowCount := traceRowCount(result.Columns)

	traceIDField := data.NewField("traceID", nil, []*string{})
	spanIDField := data.NewField("spanID", nil, []*string{})
	parentSpanIDField := data.NewField("parentSpanID", nil, []*string{})
	operationNameField := data.NewField("operationName", nil, []*string{})
	serviceNameField := data.NewField("serviceName", nil, []*string{})
	serviceTagsField := data.NewField("serviceTags", nil, []*json.RawMessage{})
	startTimeField := data.NewField("startTime", nil, []*float64{})
	durationField := data.NewField("duration", nil, []*float64{})
	logsField := data.NewField("logs", nil, []*json.RawMessage{})
	tagsField := data.NewField("tags", nil, []*json.RawMessage{})

	for row := 0; row < rowCount; row++ {
		traceIDField.Append(stringPtr(traceValueString(traceColumnValue(result, columns, "traceID", row))))
		spanIDField.Append(stringPtr(traceValueString(traceColumnValue(result, columns, "spanID", row))))
		parentSpanIDField.Append(nullableStringPtr(traceValueString(traceColumnValue(result, columns, "parentSpanID", row))))
		operationNameField.Append(stringPtr(traceValueString(traceColumnValue(result, columns, "operationName", row))))

		serviceName := traceValueString(traceColumnValue(result, columns, "serviceName", row))
		serviceNameField.Append(stringPtr(serviceName))
		serviceTagsField.Append(traceJSONPtr(traceServiceTagsValue(result, columns, serviceName, row)))

		startTime, ok := traceStartTimeMillis(traceColumnValue(result, columns, "startTime", row), columns["startTime"].name)
		if !ok {
			logger.Warn("failed to parse trace start time", "row", row, "value", debugValuePreview(traceColumnValue(result, columns, "startTime", row)))
			startTime = 0
		}
		startTimeField.Append(&startTime)

		duration, ok := traceDurationMillis(traceColumnValue(result, columns, "duration", row), columns["duration"].name)
		if !ok {
			logger.Warn("failed to parse trace duration", "row", row, "value", debugValuePreview(traceColumnValue(result, columns, "duration", row)))
			duration = 0
		}
		durationField.Append(&duration)

		logsField.Append(traceJSONPtr(traceLogsValue(traceColumnValue(result, columns, "logs", row), startTime)))
		tagsField.Append(traceJSONPtr(traceTagsValue(result, columns, row)))
	}

	return data.NewFrame(
		"Trace",
		traceIDField,
		spanIDField,
		parentSpanIDField,
		operationNameField,
		serviceNameField,
		serviceTagsField,
		startTimeField,
		durationField,
		logsField,
		tagsField,
	).SetMeta(&data.FrameMeta{
		PreferredVisualization: data.VisTypeTrace,
	}), nil
}

func traceColumns(fields []axiQuery.Field) map[string]traceColumn {
	columns := make(map[string]traceColumn, len(fields))
	for i, field := range fields {
		canonicalName, ok := traceFieldAliases[field.Name]
		if !ok {
			continue
		}
		if _, exists := columns[canonicalName]; exists {
			continue
		}
		columns[canonicalName] = traceColumn{index: i, name: field.Name}
	}

	return columns
}

func traceRowCount(columns []axiQuery.Column) int {
	rowCount := 0
	for _, column := range columns {
		if len(column) > rowCount {
			rowCount = len(column)
		}
	}

	return rowCount
}

func traceColumnValue(result *axiQuery.Table, columns map[string]traceColumn, canonicalName string, row int) any {
	column, ok := columns[canonicalName]
	if !ok || column.index >= len(result.Columns) || row >= len(result.Columns[column.index]) {
		return nil
	}

	return result.Columns[column.index][row]
}

func traceValueString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return stringifyFrameValue(v)
	}
}

func traceServiceTagsValue(result *axiQuery.Table, columns map[string]traceColumn, serviceName string, row int) any {
	if value := traceColumnValue(result, columns, "serviceTags", row); value != nil {
		return traceKeyValuePairs(value, "serviceTags")
	}
	tags := make([]map[string]any, 0, 1)
	if serviceName == "" {
		return tags
	}
	tags = append(tags, map[string]any{"key": "service.name", "value": serviceName})

	return tags
}

func traceTagsValue(result *axiQuery.Table, columns map[string]traceColumn, row int) any {
	tags := make([]map[string]any, 0)
	for fieldIndex, field := range result.Fields {
		canonicalName, isTraceField := traceFieldAliases[field.Name]
		if isTraceField && canonicalName != "tags" {
			continue
		}
		if fieldIndex >= len(result.Columns) || row >= len(result.Columns[fieldIndex]) {
			continue
		}
		value := result.Columns[fieldIndex][row]
		if value == nil {
			continue
		}
		tags = append(tags, traceKeyValuePairs(value, field.Name)...)
	}

	return tags
}

func traceJSONPtr(value any) *json.RawMessage {
	b, err := json.Marshal(value)
	if err != nil {
		b = []byte("[]")
	}
	raw := json.RawMessage(b)

	return &raw
}

func traceLogsValue(value any, fallbackTimestamp float64) []map[string]any {
	logs := traceLogEntries(value, fallbackTimestamp)
	if logs == nil {
		return []map[string]any{}
	}

	return logs
}

func traceLogEntries(value any, fallbackTimestamp float64) []map[string]any {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		var decoded any
		if err := json.Unmarshal([]byte(v), &decoded); err == nil {
			return traceLogEntries(decoded, fallbackTimestamp)
		}

		return []map[string]any{traceLogValueFromFields(fallbackTimestamp, "", []map[string]any{{"key": "message", "value": v}})}
	case []any:
		logs := make([]map[string]any, 0, len(v))
		for _, item := range v {
			logs = append(logs, traceLogEntries(item, fallbackTimestamp)...)
		}

		return logs
	case []map[string]any:
		logs := make([]map[string]any, 0, len(v))
		for _, item := range v {
			logs = append(logs, traceLogFromMap(item, fallbackTimestamp))
		}

		return logs
	case map[string]any:
		return []map[string]any{traceLogFromMap(v, fallbackTimestamp)}
	default:
		return []map[string]any{traceLogValueFromFields(fallbackTimestamp, "", []map[string]any{{"key": "value", "value": v}})}
	}
}

func traceLogFromMap(value map[string]any, fallbackTimestamp float64) map[string]any {
	timestamp := fallbackTimestamp
	for _, key := range []string{"timestamp", "time", "_time"} {
		if parsed, ok := traceLogTimestampMillis(value[key], key); ok {
			timestamp = parsed
			break
		}
	}

	name := ""
	for _, key := range []string{"name", "event.name"} {
		if value[key] != nil {
			name = traceValueString(value[key])
			break
		}
	}

	fields := make([]map[string]any, 0)
	if fieldValue, ok := value["fields"]; ok {
		fields = append(fields, traceKeyValuePairs(fieldValue, "fields")...)
	}
	if len(fields) == 0 {
		fields = append(fields, traceLogFieldsFromMap(value)...)
	}

	return traceLogValueFromFields(timestamp, name, fields)
}

func traceLogFieldsFromMap(value map[string]any) []map[string]any {
	keys := make([]string, 0, len(value))
	for key := range value {
		if traceLogReservedField(key) {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	fields := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		fields = append(fields, traceKeyValuePairs(value[key], key)...)
	}

	return fields
}

func traceLogReservedField(key string) bool {
	switch key {
	case "timestamp", "time", "_time", "name", "event.name", "fields":
		return true
	default:
		return false
	}
}

func traceLogValueFromFields(timestamp float64, name string, fields []map[string]any) map[string]any {
	if fields == nil {
		fields = []map[string]any{}
	}
	logEntry := map[string]any{
		"timestamp": timestamp,
		"fields":    fields,
	}
	if name != "" {
		logEntry["name"] = name
	}

	return logEntry
}

func traceLogTimestampMillis(value any, sourceName string) (float64, bool) {
	if value == nil {
		return 0, false
	}

	if parsed, ok := traceStartTimeMillis(value, sourceName); ok {
		return parsed, true
	}

	return 0, false
}

func traceKeyValuePairs(value any, defaultKey string) []map[string]any {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		var decoded any
		if err := json.Unmarshal([]byte(v), &decoded); err == nil {
			return traceKeyValuePairs(decoded, defaultKey)
		}
		if defaultKey == "" {
			return nil
		}

		return []map[string]any{{"key": defaultKey, "value": v}}
	case []any:
		pairs := make([]map[string]any, 0, len(v))
		for _, item := range v {
			itemPairs := traceKeyValuePairs(item, defaultKey)
			if len(itemPairs) == 1 && traceValueString(itemPairs[0]["key"]) == defaultKey {
				pairs = append(pairs, map[string]any{"key": defaultKey, "value": item})
				continue
			}
			pairs = append(pairs, itemPairs...)
		}

		return pairs
	case map[string]any:
		if key, ok := v["key"]; ok {
			pair := map[string]any{
				"key":   traceValueString(key),
				"value": v["value"],
			}
			if tagType, ok := v["type"]; ok {
				pair["type"] = tagType
			}

			return []map[string]any{pair}
		}

		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		pairs := make([]map[string]any, 0, len(keys))
		for _, key := range keys {
			pairs = append(pairs, map[string]any{"key": traceTagKey(defaultKey, key), "value": v[key]})
		}

		return pairs
	default:
		if defaultKey == "" {
			return nil
		}

		return []map[string]any{{"key": defaultKey, "value": v}}
	}
}

func traceTagKey(parentKey, key string) string {
	if parentKey == "" || parentKey == "tags" || parentKey == "attributes" {
		return key
	}

	return parentKey + "." + key
}

func traceStartTimeMillis(value any, sourceName string) (float64, bool) {
	switch v := value.(type) {
	case time.Time:
		return float64(v.UnixNano()) / float64(time.Millisecond), true
	case string:
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return float64(t.UnixNano()) / float64(time.Millisecond), true
		}
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			return timestampNumberToMillis(n, sourceName), true
		}
	case float64:
		return timestampNumberToMillis(v, sourceName), true
	}

	return 0, false
}

func timestampNumberToMillis(value float64, sourceName string) float64 {
	name := strings.ToLower(sourceName)
	switch {
	case strings.Contains(name, "ns") || strings.Contains(name, "nano"):
		return value / 1e6
	case strings.Contains(name, "us") || strings.Contains(name, "micro"):
		return value / 1e3
	case strings.Contains(name, "ms") || strings.Contains(name, "milli"):
		return value
	}

	absValue := math.Abs(value)
	switch {
	case absValue >= 1e17:
		return value / 1e6
	case absValue >= 1e14:
		return value / 1e3
	case absValue >= 1e11:
		return value
	default:
		return value * 1e3
	}
}

func traceDurationMillis(value any, sourceName string) (float64, bool) {
	switch v := value.(type) {
	case time.Duration:
		return float64(v) / float64(time.Millisecond), true
	case string:
		if d, err := time.ParseDuration(v); err == nil {
			return float64(d) / float64(time.Millisecond), true
		}
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			return durationNumberToMillis(n, sourceName), true
		}
	case float64:
		return durationNumberToMillis(v, sourceName), true
	}

	return 0, false
}

func durationNumberToMillis(value float64, sourceName string) float64 {
	name := strings.ToLower(sourceName)
	switch {
	case strings.Contains(name, "ms") || strings.Contains(name, "milli"):
		return value
	case strings.Contains(name, "us") || strings.Contains(name, "micro"):
		return value / 1e3
	case strings.Contains(name, "ns") || strings.Contains(name, "nano"), name == "duration":
		return value / 1e6
	case strings.Contains(name, "sec") || strings.HasSuffix(name, "_s"):
		return value * 1e3
	default:
		return value
	}
}
