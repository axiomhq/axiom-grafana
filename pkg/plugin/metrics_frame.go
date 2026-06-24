package plugin

import (
	"sort"
	"strings"
	"time"

	"github.com/axiomhq/axiom-grafana/pkg/axiomapi"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

const metricsDisplayLabelTag = "__label"

type metricsFrameBuilder struct {
	metadata axiomapi.MetricsQueryMetadata
	refID    string
}

type metricsTableRow struct {
	tags   map[string]string
	values map[string]*float64
}

func newMetricsFrameBuilder(metadata axiomapi.MetricsQueryMetadata, refID string) metricsFrameBuilder {
	return metricsFrameBuilder{
		metadata: metadata,
		refID:    refID,
	}
}

func (b metricsFrameBuilder) Build(group axiomapi.MetricsQuerySeries) *data.Frame {
	frameName := group.Metric
	if frameName == "" {
		frameName = "value"
	}

	labels := data.Labels{}
	for key, value := range group.Tags {
		labels[key] = value
	}

	fieldName := metricsSeriesFieldName(frameName, group.Tags)

	frame := data.NewFrame(frameName)
	frame.RefID = b.refID
	frame.Meta = &data.FrameMeta{
		Type:                   data.FrameTypeTimeSeriesMulti,
		TypeVersion:            data.FrameTypeVersion{0, 1},
		PreferredVisualization: data.VisTypeGraph,
		// TODO: fix notices
		// Notices: []data.Notice{res.Metadata.Warnings},
	}
	timeField := data.NewField("Time", nil, []time.Time{})
	applyMetricsTimeFieldMetadata(timeField, group)
	frame.Fields = append(frame.Fields, timeField)
	valueField := data.NewField(fieldName, labels, []*float64{})
	applyMetricsDisplayName(valueField, b.refID, group.Tags)
	applyMetricsFieldMetadata(valueField, b.metadata)
	frame.Fields = append(frame.Fields, valueField)

	// add values
	for i, value := range group.Data {
		_time := time.Unix(group.Start+int64(i*group.Resolution), 0)
		frame.Fields[0].Append(_time)
		frame.Fields[1].Append(value)
	}

	return frame
}

func (b metricsFrameBuilder) BuildTable(series []axiomapi.MetricsQuerySeries) *data.Frame {
	frame := data.NewFrame("metrics")
	frame.RefID = b.refID
	frame.Meta = &data.FrameMeta{
		PreferredVisualization: data.VisTypeTable,
	}

	if len(series) == 0 {
		return frame
	}

	includeLabelColumn := false
	tagColumns := make([]string, 0)
	tagColumnSeen := map[string]struct{}{}
	metricColumns := make([]string, 0)
	metricColumnSeen := map[string]struct{}{}
	for _, group := range series {
		if _, ok := group.Tags[metricsDisplayLabelTag]; ok {
			includeLabelColumn = true
		}
		for tag := range group.Tags {
			if tag == metricsDisplayLabelTag {
				continue
			}
			if _, ok := tagColumnSeen[tag]; ok {
				continue
			}
			tagColumnSeen[tag] = struct{}{}
			tagColumns = append(tagColumns, tag)
		}

		metric := metricsTableMetricName(group)
		if _, ok := metricColumnSeen[metric]; ok {
			continue
		}
		metricColumnSeen[metric] = struct{}{}
		metricColumns = append(metricColumns, metric)
	}
	sort.Strings(tagColumns)

	rowsByKey := map[string]*metricsTableRow{}
	rowKeys := make([]string, 0)
	for _, group := range series {
		key := metricsTableRowKey(group)
		row, ok := rowsByKey[key]
		if !ok {
			row = &metricsTableRow{
				tags:   group.Tags,
				values: map[string]*float64{},
			}
			rowsByKey[key] = row
			rowKeys = append(rowKeys, key)
		}

		row.values[metricsTableMetricName(group)] = latestMetricsValue(group.Data)
	}

	if includeLabelColumn {
		labelField := data.NewField(metricsDisplayLabelTag, nil, []*string{})
		for _, key := range rowKeys {
			appendMetricsTableStringValue(labelField, rowsByKey[key].tags, metricsDisplayLabelTag)
		}
		frame.Fields = append(frame.Fields, labelField)
	}

	for _, tag := range tagColumns {
		field := data.NewField(tag, nil, []*string{})
		for _, key := range rowKeys {
			appendMetricsTableStringValue(field, rowsByKey[key].tags, tag)
		}
		frame.Fields = append(frame.Fields, field)
	}

	for _, metric := range metricColumns {
		field := data.NewField(metric, nil, []*float64{})
		applyMetricsFieldMetadata(field, b.metadata)
		for _, key := range rowKeys {
			field.Append(rowsByKey[key].values[metric])
		}
		frame.Fields = append(frame.Fields, field)
	}

	return frame
}

func metricsTableMetricName(group axiomapi.MetricsQuerySeries) string {
	if group.Metric == "" {
		return "value"
	}

	return group.Metric
}

func metricsTableRowKey(group axiomapi.MetricsQuerySeries) string {
	if len(group.Tags) == 0 {
		return ""
	}

	keys := make([]string, 0, len(group.Tags))
	for key := range group.Tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"\x00"+group.Tags[key])
	}

	return strings.Join(parts, "\x01")
}

func appendMetricsTableStringValue(field *data.Field, tags map[string]string, tag string) {
	value, ok := tags[tag]
	if !ok {
		field.Append(nil)
		return
	}

	field.Append(&value)
}

func latestMetricsValue(values []*float64) *float64 {
	for i := len(values) - 1; i >= 0; i-- {
		if values[i] != nil {
			return values[i]
		}
	}

	return nil
}

func metricsSeriesFieldName(metric string, tags map[string]string) string {
	if label, ok := tags[metricsDisplayLabelTag]; ok {
		// MPL users can add `extend __label = ...` to choose the series name.
		// applyMetricsDisplayName also uses this value as Grafana's explicit
		// legend text, hiding the normal `{tag=value}` suffix while keeping the
		// labels available on the field.
		return label
	}

	if len(tags) == 0 {
		return metric
	}

	// Without an explicit label, use the tag values as the leading series name
	// so legends are scannable before Grafana appends the normal tag set.
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, tags[key])
	}

	return strings.Join(values, " | ")
}

func applyMetricsDisplayName(field *data.Field, refID string, tags map[string]string) {
	if label, ok := tags[metricsDisplayLabelTag]; ok {
		// A datasource-provided display name suppresses Grafana's automatic
		// label suffix, so `__label` produces a clean legend while preserving
		// the full tag set on the series for filtering and inspection.
		if field.Config == nil {
			field.Config = &data.FieldConfig{}
		}
		field.Config.DisplayNameFromDS = label
		return
	}

	if len(tags) > 0 || refID == "" {
		return
	}

	if field.Config == nil {
		field.Config = &data.FieldConfig{}
	}
	field.Config.DisplayNameFromDS = refID
}

func applyMetricsFieldMetadata(field *data.Field, metadata axiomapi.MetricsQueryMetadata) {
	if metadata.Unit == "" {
		return
	}

	if field.Config == nil {
		field.Config = &data.FieldConfig{}
	}
	field.Config.Unit = metadata.Unit
}

func applyMetricsTimeFieldMetadata(field *data.Field, group axiomapi.MetricsQuerySeries) {
	if group.Resolution <= 0 {
		return
	}

	if field.Config == nil {
		field.Config = &data.FieldConfig{}
	}
	field.Config.Interval = float64(group.Resolution * 1000)
}
