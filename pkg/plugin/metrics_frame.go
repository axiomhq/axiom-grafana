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
		Type:        data.FrameTypeTimeSeriesMulti,
		TypeVersion: data.FrameTypeVersion{0, 1},
		// TODO: fix notices
		// Notices: []data.Notice{res.Metadata.Warnings},
	}
	timeField := data.NewField("Time", nil, []time.Time{})
	applyMetricsTimeFieldMetadata(timeField, group)
	frame.Fields = append(frame.Fields, timeField)
	valueField := data.NewField(fieldName, labels, []*float64{})
	applyMetricsDisplayName(valueField, b.refID, len(labels) > 0)
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

func metricsSeriesFieldName(metric string, tags map[string]string) string {
	if label, ok := tags[metricsDisplayLabelTag]; ok {
		// MPL users can add `extend __label = ...` to choose the leading
		// series name while keeping all labels available for Grafana's normal
		// `{tag=value}` suffix and filtering behavior.
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

func applyMetricsDisplayName(field *data.Field, refID string, hasLabels bool) {
	if hasLabels || refID == "" {
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
