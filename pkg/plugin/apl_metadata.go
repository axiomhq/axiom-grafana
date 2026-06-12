package plugin

import (
	"fmt"
	"strings"

	axiQuery "github.com/axiomhq/axiom-go/axiom/query"
	"github.com/axiomhq/axiom-grafana/pkg/axiomapi"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func fieldMetaByNameForResponse(result axiomapi.APLQueryResponse) map[string]axiomapi.APLFieldMetaMap {
	fieldMetaByName := map[string]axiomapi.APLFieldMetaMap{}

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

func applyAPLFieldMetadata(field *data.Field, f axiQuery.Field, fieldMetaByName map[string]axiomapi.APLFieldMetaMap) {
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

func lookupAPLFieldMeta(f axiQuery.Field, fieldMetaByName map[string]axiomapi.APLFieldMetaMap) (axiomapi.APLFieldMetaMap, bool) {
	if len(fieldMetaByName) == 0 {
		return axiomapi.APLFieldMetaMap{}, false
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

	return axiomapi.APLFieldMetaMap{}, false
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

func applyPreferredVisualization(frame *data.Frame, visualization data.VisType) {
	if frame == nil {
		return
	}
	if frame.Meta == nil {
		frame.Meta = &data.FrameMeta{}
	}
	if frame.Meta.PreferredVisualization == "" {
		frame.Meta.PreferredVisualization = visualization
	}
}

func aplQueryStats(status axiomapi.APLQueryStatus) []data.QueryStat {
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

func aplQueryNotices(status axiomapi.APLQueryStatus) []data.Notice {
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
