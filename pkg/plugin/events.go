package plugin

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	axiQuery "github.com/axiomhq/axiom-go/axiom/query"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// queryEvents executes an APL query against the configured endpoint
// (edge or legacy apiHost, depending on configuration).
func (d *Datasource) queryEvents(ctx context.Context, q *queryModel, startTime, endTime time.Time) (*backend.DataResponse, error) {
	logger := log.DefaultLogger.FromContext(ctx)

	reqBody := APLQueryRequest{
		APL:       q.Query,
		StartTime: startTime,
		EndTime:   endTime,
	}

	result, err := d.api.QueryAPL(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	var response backend.DataResponse

	if len(result.Tables) == 0 {
		return nil, fmt.Errorf("query returned no tables")
	}

	var frame *data.Frame
	var newFrame *data.Frame
	frameOptions := aplFrameOptions{
		FieldMetaByName: fieldMetaByNameForResponse(result),
		Status:          result.Status,
	}
	if q.Query != nil {
		frameOptions.Query = *q.Query
	}
	if len(result.Tables) > 1 {
		if q.Totals {
			frame, err = buildFrame(ctx, &result.Tables[1], frameOptions)
		} else {
			frame, err = buildFrame(ctx, &result.Tables[0], frameOptions)
		}
		if err != nil {
			return nil, err
		}

		newFrame, err = longToWideFrame(frame)
		if err != nil {
			logger.Error("transformation from long to wide failed", "error", err.Error())
		}
		if newFrame != nil {
			if newFrame.Meta == nil {
				newFrame.Meta = &data.FrameMeta{}
			}
			if newFrame.Meta.PreferredVisualization == "" {
				newFrame.Meta.PreferredVisualization = data.VisTypeGraph
			}
		}
	} else {
		frame, err = buildFrame(ctx, &result.Tables[0], frameOptions)
		if err != nil {
			return nil, err
		}
		if frame != nil {
			if frame.Meta == nil {
				frame.Meta = &data.FrameMeta{}
			}
			if frame.Meta.PreferredVisualization == "" {
				frame.Meta.PreferredVisualization = data.VisTypeLogs
			}
		}
	}

	if newFrame != nil {
		response.Frames = append(response.Frames, newFrame)
	} else {
		response.Frames = append(response.Frames, frame)
	}

	return &response, nil
}

func longToWideFrame(frame *data.Frame) (*data.Frame, error) {
	wideFrame, err := data.LongToWide(frame, &data.FillMissing{
		Mode: data.FillModePrevious,
	})
	if err != nil {
		return nil, err
	}

	applyWideFieldConfigs(wideFrame, frame)
	applyLabelDisplayNames(wideFrame)
	return wideFrame, nil
}

func applyWideFieldConfigs(wideFrame, longFrame *data.Frame) {
	configsByName := make(map[string]*data.FieldConfig, len(longFrame.Fields))
	for _, field := range longFrame.Fields {
		if field.Config != nil {
			configsByName[field.Name] = field.Config
		}
	}

	for _, field := range wideFrame.Fields {
		config := configsByName[field.Name]
		if config == nil {
			continue
		}
		field.Config = cloneFieldConfig(config)
	}
}

func cloneFieldConfig(config *data.FieldConfig) *data.FieldConfig {
	if config == nil {
		return nil
	}

	clone := *config
	if config.Custom != nil {
		clone.Custom = make(map[string]interface{}, len(config.Custom))
		for key, value := range config.Custom {
			clone.Custom[key] = value
		}
	}
	return &clone
}

func applyLabelDisplayNames(frame *data.Frame) {
	for _, field := range frame.Fields {
		if len(field.Labels) == 0 {
			continue
		}

		if field.Config == nil {
			field.Config = &data.FieldConfig{}
		}
		field.Config.DisplayNameFromDS = labelsDisplayName(field.Labels)
	}
}

func labelsDisplayName(labels data.Labels) string {
	if len(labels) == 1 {
		for _, value := range labels {
			return value
		}
	}

	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, labels[key]))
	}

	return strings.Join(parts, ", ")
}

func fieldMetaByNameForResponse(result APLQueryResponse) map[string]APLFieldMetaMap {
	fieldMetaByName := map[string]APLFieldMetaMap{}

	for _, datasetName := range result.DatasetNames {
		for _, fieldMeta := range result.FieldsMetaMap[datasetName] {
			if _, exists := fieldMetaByName[fieldMeta.Name]; !exists {
				fieldMetaByName[fieldMeta.Name] = fieldMeta
			}
		}
	}

	for _, fieldsMeta := range result.FieldsMetaMap {
		for _, fieldMeta := range fieldsMeta {
			if _, exists := fieldMetaByName[fieldMeta.Name]; !exists {
				fieldMetaByName[fieldMeta.Name] = fieldMeta
			}
		}
	}

	return fieldMetaByName
}

func applyAPLFieldMetadata(field *data.Field, f axiQuery.Field, fieldMetaByName map[string]APLFieldMetaMap) {
	fieldMeta, ok := lookupAPLFieldMeta(f, fieldMetaByName)
	if !ok {
		return
	}

	if field.Config == nil {
		field.Config = &data.FieldConfig{}
	}
	if fieldMeta.Unit != "" {
		field.Config.Unit = fieldMeta.Unit
	}
	if fieldMeta.Description != "" {
		field.Config.Description = fieldMeta.Description
	}
}

func lookupAPLFieldMeta(f axiQuery.Field, fieldMetaByName map[string]APLFieldMetaMap) (APLFieldMetaMap, bool) {
	if len(fieldMetaByName) == 0 {
		return APLFieldMetaMap{}, false
	}

	if fieldMeta, ok := fieldMetaByName[f.Name]; ok {
		return fieldMeta, true
	}

	if f.Aggregation != nil {
		for _, sourceField := range f.Aggregation.Fields {
			if fieldMeta, ok := fieldMetaByName[sourceField]; ok {
				return fieldMeta, true
			}
		}
	}

	return APLFieldMetaMap{}, false
}

func applyAPLFrameMetadata(frame *data.Frame, opts aplFrameOptions) {
	if opts.Status == nil && opts.Query == "" {
		return
	}

	if frame.Meta == nil {
		frame.Meta = &data.FrameMeta{}
	}
	if opts.Query != "" {
		frame.Meta.ExecutedQueryString = opts.Query
	}
	if opts.Status == nil {
		return
	}

	frame.Meta.Stats = aplQueryStats(*opts.Status)
	frame.Meta.Custom = map[string]any{
		"axiomStatus": opts.Status,
	}
	frame.Meta.Notices = append(frame.Meta.Notices, aplQueryNotices(*opts.Status)...)
}

func aplQueryStats(status APLQueryStatus) []data.QueryStat {
	return []data.QueryStat{
		{FieldConfig: data.FieldConfig{DisplayName: "Elapsed time", Unit: "µs"}, Value: float64(status.ElapsedTime)},
		{FieldConfig: data.FieldConfig{DisplayName: "Blocks examined"}, Value: float64(status.BlocksExamined)},
		{FieldConfig: data.FieldConfig{DisplayName: "Blocks cached"}, Value: float64(status.BlocksCached)},
		{FieldConfig: data.FieldConfig{DisplayName: "Blocks matched"}, Value: float64(status.BlocksMatched)},
		{FieldConfig: data.FieldConfig{DisplayName: "Blocks skipped"}, Value: float64(status.BlocksSkipped)},
		{FieldConfig: data.FieldConfig{DisplayName: "Rows examined"}, Value: float64(status.RowsExamined)},
		{FieldConfig: data.FieldConfig{DisplayName: "Rows matched"}, Value: float64(status.RowsMatched)},
		{FieldConfig: data.FieldConfig{DisplayName: "Groups"}, Value: float64(status.NumGroups)},
		{FieldConfig: data.FieldConfig{DisplayName: "Cache status"}, Value: float64(status.CacheStatus)},
	}
}

func aplQueryNotices(status APLQueryStatus) []data.Notice {
	notices := make([]data.Notice, 0, len(status.Messages)+1)
	if status.IsPartial {
		notices = append(notices, data.Notice{
			Severity: data.NoticeSeverityWarning,
			Text:     "Axiom returned a partial response",
			Inspect:  data.InspectTypeStats,
		})
	}

	for _, message := range status.Messages {
		text := message.Msg
		if message.Count > 1 {
			text = fmt.Sprintf("%s (x%d)", text, message.Count)
		}
		if message.Code != "" {
			text = fmt.Sprintf("%s: %s", message.Code, text)
		}

		notices = append(notices, data.Notice{
			Severity: noticeSeverityForPriority(message.Priority),
			Text:     text,
			Inspect:  data.InspectTypeStats,
		})
	}

	return notices
}

func noticeSeverityForPriority(priority string) data.NoticeSeverity {
	switch strings.ToLower(priority) {
	case "error", "fatal":
		return data.NoticeSeverityError
	case "warn", "warning":
		return data.NoticeSeverityWarning
	default:
		return data.NoticeSeverityInfo
	}
}

func buildFrame(ctx context.Context, result *axiQuery.Table, opts ...aplFrameOptions) (*data.Frame, error) {
	frameOptions := aplFrameOptions{}
	if len(opts) > 0 {
		frameOptions = opts[0]
	}

	logger := log.DefaultLogger.FromContext(ctx)
	if fieldsMatchTrace(ctx, result.Fields) {
		frame, err := buildTraceFrame(ctx, result)
		if err != nil {
			return nil, err
		}
		applyAPLFrameMetadata(frame, frameOptions)
		return frame, nil
	}

	frame := data.NewFrame("response")

	// define fields
	fields := make([]*data.Field, 0, len(result.Fields))
	fieldTypes := make([]string, 0, len(result.Fields))

	for i, f := range result.Fields {
		f := f
		i := i
		fieldType := f.Type
		var sampleValue any
		sampleRow := -1
		columnLen := 0
		if i < len(result.Columns) {
			columnLen = len(result.Columns[i])
			sampleValue, sampleRow, _ = firstDebugValue(result.Columns[i])
		}
		if f.Name == "_time" {
			fieldType = "datetime"
		} else if fieldType == "unknown" && i < len(result.Columns) {
			fieldType = inferUnknownFieldType(f.Name, result.Columns[i])
			logger.Debug("inferred unknown APL field type", "field", f.Name, "type", fieldType)
		}

		var field *data.Field
		func() {
			var fieldValues any
			defer func() {
				if r := recover(); r != nil {
					logger.Error(
						"panic creating APL data frame field",
						"field", f.Name,
						"fieldIndex", i,
						"declaredType", f.Type,
						"resolvedType", fieldType,
						"columnLength", columnLen,
						"sampleRow", sampleRow,
						"sampleValueType", debugValueType(sampleValue),
						"sampleValue", debugValuePreview(sampleValue),
						"fieldValuesType", debugValueType(fieldValues),
						"panic", fmt.Sprintf("%v", r),
					)
					panic(r)
				}
			}()

			switch fieldType {
			case "datetime":
				fieldValues = []time.Time{}
			case "integer":
				fieldValues = []*float64{}
			case "float":
				fieldValues = []*float64{}
			case "bool":
				fieldValues = []*bool{}
			case "timespan":
				fieldValues = []*string{}
			case "array":
				fieldValues = []*string{}
			default:
				fieldValues = []*string{}
			}

			field = data.NewField(f.Name, nil, fieldValues)
		}()
		applyAPLFieldMetadata(field, f, frameOptions.FieldMetaByName)

		fields = append(fields, field)
		fieldTypes = append(fieldTypes, fieldType)
	}

	for colIndex, col := range result.Columns {
		if colIndex >= len(result.Fields) || colIndex >= len(fields) {
			return nil, fmt.Errorf("table column %d has no matching field metadata", colIndex)
		}

		for i := 0; i < len(col); i++ {
			colIndex := colIndex
			i := i

			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Error(
							"panic appending APL data frame value",
							"field", result.Fields[colIndex].Name,
							"fieldIndex", colIndex,
							"rowIndex", i,
							"declaredType", result.Fields[colIndex].Type,
							"resolvedType", fieldTypes[colIndex],
							"valueType", debugValueType(col[i]),
							"value", debugValuePreview(col[i]),
							"panic", fmt.Sprintf("%v", r),
						)
						panic(r)
					}
				}()

				// check if the value is nil
				// if it is, append nil to the field, skip more processing
				if col[i] == nil {
					fields[colIndex].Append(nil)
					return
				}

				// check the type and parse it accordingly
				switch fieldTypes[colIndex] {
				case "datetime":
					// parse time
					t, err := time.Parse(time.RFC3339, col[i].(string))
					if err != nil {
						logger.Warn("Failed to parse time", "time", col[i])
						fields[colIndex].Append(time.Time{})
						return
					}
					fields[colIndex].Append(t)
				case "integer":
					num := col[i].(float64)
					fields[colIndex].Append(&num)
				case "float":
					num := col[i].(float64)
					fields[colIndex].Append(&num)
				case "string", "unknown":
					txt, ok := col[i].(string)
					if !ok {
						txt = stringifyFrameValue(col[i])
					}
					fields[colIndex].Append(&txt)
				case "bool":
					b := col[i].(bool)
					fields[colIndex].Append(&b)
				case "timespan":
					num := col[i].(string)
					fields[colIndex].Append(&num)
				case "array":
					txt := stringifyFrameValue(col[i])
					fields[colIndex].Append(&txt)
				default:
					txt := stringifyFrameValue(col[i])
					fields[colIndex].Append(&txt)
				}
			}()

		}
	}
	frame.Fields = fields
	applyAPLFrameMetadata(frame, frameOptions)

	return frame, nil
}
