package plugin

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	axiQuery "github.com/axiomhq/axiom-go/axiom/query"
	"github.com/axiomhq/axiom-grafana/pkg/axiomapi"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

const (
	logsVolumeQueryType              = "logs-volume"
	supplementaryQueryTypeLogsVolume = "LogsVolume"
)

func isLogsVolumeQuery(query backend.DataQuery, model *queryModel) bool {
	if query.QueryType == logsVolumeQueryType {
		return true
	}
	return model.SupportingQueryType != nil && *model.SupportingQueryType == supplementaryQueryTypeLogsVolume
}

func (d *Datasource) queryLogsVolume(ctx context.Context, q *queryModel, query backend.DataQuery, datasourceName string) (*backend.DataResponse, error) {
	apl := logsVolumeAPL(*q.Query, logsVolumeInterval(query))
	reqBody := axiomapi.APLQueryRequest{
		APL:       &apl,
		StartTime: query.TimeRange.From,
		EndTime:   query.TimeRange.To,
	}

	result, err := d.api.QueryAPL(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	frame, err := newLogsVolumeFrameBuilder(query, datasourceName, *q.Query).Build(result)
	if err != nil {
		return nil, err
	}

	var response backend.DataResponse
	response.Frames = append(response.Frames, frame)
	return &response, nil
}

func logsVolumeAPL(query string, interval time.Duration) string {
	sourceQuery := strings.TrimSpace(query)
	sourceQuery = strings.TrimSuffix(sourceQuery, ";")

	return fmt.Sprintf(`(%s)
| extend _axiom_logs_volume_time = coalesce(_time, _sysTime)
| where isnotnull(_axiom_logs_volume_time)
| summarize count_ = count() by _time = bin(_axiom_logs_volume_time, %s)
| order by _time asc`, sourceQuery, aplDurationLiteral(interval))
}

func logsVolumeInterval(query backend.DataQuery) time.Duration {
	if query.Interval > 0 {
		return query.Interval
	}

	timeRange := query.TimeRange.To.Sub(query.TimeRange.From)
	if timeRange <= 0 {
		return time.Minute
	}

	maxDataPoints := query.MaxDataPoints
	if maxDataPoints <= 0 {
		maxDataPoints = 240
	}

	interval := time.Duration(math.Ceil(float64(timeRange) / float64(maxDataPoints)))
	if interval < time.Second {
		return time.Second
	}
	return interval
}

func aplDurationLiteral(duration time.Duration) string {
	if duration < time.Second {
		duration = time.Second
	}
	duration = duration.Round(time.Second)

	switch {
	case duration%(24*time.Hour) == 0:
		return fmt.Sprintf("%dd", int(duration/(24*time.Hour)))
	case duration%time.Hour == 0:
		return fmt.Sprintf("%dh", int(duration/time.Hour))
	case duration%time.Minute == 0:
		return fmt.Sprintf("%dm", int(duration/time.Minute))
	default:
		return fmt.Sprintf("%ds", int(duration/time.Second))
	}
}

type logsVolumeFrameBuilder struct {
	query          backend.DataQuery
	datasourceName string
	sourceQuery    string
}

func newLogsVolumeFrameBuilder(query backend.DataQuery, datasourceName string, sourceQuery string) logsVolumeFrameBuilder {
	return logsVolumeFrameBuilder{query: query, datasourceName: datasourceName, sourceQuery: sourceQuery}
}

func (b logsVolumeFrameBuilder) Build(result axiomapi.APLQueryResponse) (*data.Frame, error) {
	if len(result.Tables) == 0 {
		return nil, fmt.Errorf("logs volume query returned no tables")
	}

	table := result.Tables[0]
	columns := logsVolumeColumns(table.Fields)
	rowCount := traceRowCount(table.Columns)

	timeField := data.NewField("Time", nil, []time.Time{})
	countField := data.NewField("count", nil, []*float64{})
	countField.Config = &data.FieldConfig{
		DisplayNameFromDS: "Logs volume",
		Unit:              "logs",
	}

	for row := 0; row < rowCount; row++ {
		timestamp, ok := logTimestamp(logsVolumeColumnValue(table, columns, "time", row))
		if !ok {
			continue
		}

		count := logsVolumeCount(logsVolumeColumnValue(table, columns, "count", row))
		timeField.Append(timestamp)
		countField.Append(&count)
	}

	frame := data.NewFrame("Logs volume", timeField, countField)
	frame.RefID = b.query.RefID
	frame.Meta = &data.FrameMeta{
		Type:                   data.FrameTypeTimeSeriesWide,
		TypeVersion:            data.FrameTypeVersion{0, 1},
		PreferredVisualization: data.VisTypeGraph,
		Custom: map[string]any{
			"absoluteRange": map[string]any{
				"from": b.query.TimeRange.From.UnixMilli(),
				"to":   b.query.TimeRange.To.UnixMilli(),
			},
			"datasourceName": b.datasourceName,
			"logsVolumeType": "FullRange",
			"sourceQuery": map[string]any{
				"kind":  "apl",
				"query": b.sourceQuery,
				"refId": strings.TrimPrefix(b.query.RefID, "log-volume-"),
			},
		},
	}

	return frame, nil
}

func datasourceName(ctx backend.PluginContext) string {
	if ctx.DataSourceInstanceSettings == nil {
		return ""
	}
	return ctx.DataSourceInstanceSettings.Name
}

func logsVolumeColumns(fields []axiQuery.Field) map[string]int {
	columns := make(map[string]int, len(fields))
	for i, field := range fields {
		switch strings.ToLower(field.Name) {
		case "_time", "time", "timestamp":
			if _, exists := columns["time"]; !exists {
				columns["time"] = i
			}
		case "count", "count_":
			if _, exists := columns["count"]; !exists {
				columns["count"] = i
			}
		}
	}

	return columns
}

func logsVolumeColumnValue(table axiQuery.Table, columns map[string]int, name string, row int) any {
	columnIndex, ok := columns[name]
	if !ok || columnIndex >= len(table.Columns) || row >= len(table.Columns[columnIndex]) {
		return nil
	}

	return table.Columns[columnIndex][row]
}

func logsVolumeCount(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}
