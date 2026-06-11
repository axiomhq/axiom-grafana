package plugin

import (
	"context"
	"net/http"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type MetricsQueryResponse struct {
	Metadata MetricsQueryMetadata `json:"metadata"`
	Series   []MetricsQuerySeries `json:"series"`
}

type MetricsQueryMetadata struct {
	Unit     string   `json:"unit"`
	Warnings []string `json:"warnings"`
}

type MetricsQuerySeries struct {
	Resolution int
	Start      int64
	Data       []*float64
	Tags       map[string]string
	Metric     string
}

// queryMetrics executes an MPL query against the configured endpoint
// (edge or legacy apiHost, depending on configuration).
func (d *Datasource) queryMetrics(ctx context.Context, q *queryModel, refID string, startTime, endTime time.Time) (*backend.DataResponse, error) {
	endpoint := "/v1/query/_mpl"

	reqBody := APLQueryRequest{
		MPL:       q.Query,
		StartTime: startTime,
		EndTime:   endTime,
	}

	req, err := d.api.client.NewRequest(ctx, http.MethodPost, endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.metrics.v3+json")

	var res MetricsQueryResponse
	_, err = d.api.client.Do(req, &res)
	if err != nil {
		return nil, err
	}

	var response backend.DataResponse

	for _, group := range res.Series {
		response.Frames = append(response.Frames, buildMetricsFrame(group, res.Metadata, refID))
	}

	// extract the data from the response
	return &response, nil
}

func buildMetricsFrame(group MetricsQuerySeries, metadata MetricsQueryMetadata, refID string) *data.Frame {
	frameName := group.Metric
	if frameName == "" {
		frameName = "value"
	}

	labels := data.Labels{}
	for key, value := range group.Tags {
		labels[key] = value
	}

	frame := data.NewFrame(frameName)
	frame.RefID = refID
	frame.Meta = &data.FrameMeta{
		Type:        data.FrameTypeTimeSeriesMulti,
		TypeVersion: data.FrameTypeVersion{0, 1},
		// TODO: fix notices
		// Notices: []data.Notice{res.Metadata.Warnings},
	}
	timeField := data.NewField("Time", nil, []time.Time{})
	applyMetricsTimeFieldMetadata(timeField, group)
	frame.Fields = append(frame.Fields, timeField)
	valueField := data.NewField(frameName, labels, []*float64{})
	applyMetricsDisplayName(valueField, refID, len(labels) > 0)
	applyMetricsFieldMetadata(valueField, metadata)
	frame.Fields = append(frame.Fields, valueField)

	// add values
	for i, value := range group.Data {
		_time := time.Unix(group.Start+int64(i*group.Resolution), 0)
		frame.Fields[0].Append(_time)
		frame.Fields[1].Append(value)
	}

	return frame
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

func applyMetricsFieldMetadata(field *data.Field, metadata MetricsQueryMetadata) {
	if metadata.Unit == "" {
		return
	}

	if field.Config == nil {
		field.Config = &data.FieldConfig{}
	}
	field.Config.Unit = metadata.Unit
}

func applyMetricsTimeFieldMetadata(field *data.Field, group MetricsQuerySeries) {
	if group.Resolution <= 0 {
		return
	}

	if field.Config == nil {
		field.Config = &data.FieldConfig{}
	}
	field.Config.Interval = float64(group.Resolution * 1000)
}
