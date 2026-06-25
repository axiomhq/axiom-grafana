package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/axiomhq/axiom-grafana/pkg/axiomapi"
	"github.com/axiomhq/axiom-grafana/pkg/config"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/concurrent"
)

// Make sure Datasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler interfaces. Plugin should not implement all these
// interfaces- only those which are required for a particular task.
var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
	_ backend.CallResourceHandler   = (*Datasource)(nil)
)

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct {
	backend.CallResourceHandler
	api *axiomapi.Client
}

type queryModel struct {
	Version                 *string `json:"version"`
	APL                     *string `json:"apl"`
	Kind                    *string `json:"kind"`
	Query                   *string `json:"query"`
	SupportingQueryType     *string `json:"supportingQueryType"`
	Totals                  bool    `json:"totals"`
	IncludeTotalsTableFrame bool    `json:"includeTotalsTableFrame"`
	IncludeLogsVolumeFrame  bool    `json:"includeLogsVolumeFrame"`
}

// NewDatasource creates a new datasource instance.
func NewDatasource(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	logger := log.DefaultLogger.FromContext(ctx)
	config, err := config.ParseConfig(ctx, settings)
	if err != nil {
		logger.Error("Failed to parse config", "error", err.Error())
		return nil, err
	}

	opts, err := settings.HTTPClientOptions(ctx)
	if err != nil {
		return nil, err
	}
	api, err := axiomapi.NewClient(opts, config)
	if err != nil {
		return nil, err
	}

	ds := Datasource{
		api: api,
	}
	resourceHandler := ds.newResourceHandler()
	ds.CallResourceHandler = resourceHandler

	return &ds, nil
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *Datasource) Dispose() {
	// Clean up datasource instance resources.
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	logger := log.DefaultLogger.FromContext(ctx)
	// log panic
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
				logger.Error(err.Error())
				return
			}
			logger.Error(err.Error())
			logger.Error(string(debug.Stack()))
		}
	}()

	return concurrent.QueryData(ctx, req, d.execQuery, 10)
}

func (d *Datasource) execQuery(ctx context.Context, query concurrent.Query) (response backend.DataResponse) {
	logger := log.DefaultLogger.FromContext(ctx)
	// log panic
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
			}
			logger.Error(err.Error())
			logger.Error(string(debug.Stack()))
			response = backend.ErrDataResponse(backend.StatusInternal, "Unexpected error while running query")
		}
	}()

	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	err := json.Unmarshal(query.DataQuery.JSON, &qm)
	if err != nil {
		// Log the actual error since it will be included in the Grafana server log and return a more generic message to the end user.
		logger.Error(err.Error())
		return backend.ErrDataResponse(backend.StatusInternal, "Could not parse query")
	}

	if isEmptyQuery(qm.Query) && qm.APL != nil {
		qm.Query = qm.APL
	}

	if isEmptyQuery(qm.Query) {
		return backend.DataResponse{}
	}

	kind := "apl"
	if qm.Kind != nil && *qm.Kind != "" {
		kind = *qm.Kind
	}

	var queryResponse *backend.DataResponse

	// make request to axiom
	if isLogsVolumeQuery(query.DataQuery, &qm) {
		queryResponse, err = d.queryLogsVolume(ctx, &qm, query.DataQuery, datasourceName(query.PluginContext))
	} else if kind == "mpl" {
		queryResponse, err = d.queryMetrics(ctx, &qm, query.DataQuery.RefID, query.DataQuery.TimeRange.From, query.DataQuery.TimeRange.To, query.DataQuery.MaxDataPoints)
	} else {
		queryResponse, err = d.queryEvents(ctx, &qm, query.DataQuery, datasourceName(query.PluginContext))
	}
	if err != nil {
		logger.Error("failed to query axiom", "error", err)
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("axiom error: %v", err.Error()))
	}
	if queryResponse == nil {
		logger.Error("query returned nil response")
		return backend.ErrDataResponse(backend.StatusInternal, "Query returned no response")
	}

	return *queryResponse
}

func isEmptyQuery(query *string) bool {
	if query == nil {
		return true
	}

	for _, line := range strings.Split(*query, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "//") {
			return false
		}
	}

	return true
}

// queryEvents executes an APL query against the configured endpoint.
// Dashboard panels ask for includeLogsVolumeFrame so raw log queries still
// produce a numeric frame for Grafana's default Time series visualization.
func (d *Datasource) queryEvents(ctx context.Context, q *queryModel, query backend.DataQuery, datasourceName string) (*backend.DataResponse, error) {
	reqBody := axiomapi.APLQueryRequest{
		APL:       q.Query,
		StartTime: query.TimeRange.From,
		EndTime:   query.TimeRange.To,
	}

	result, err := d.api.QueryAPL(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	frameOptions := aplFrameOptions{
		FieldMetaByName: fieldMetaByNameForResponse(result),
		Status:          result.Status,
		TraceID:         result.TraceID,
	}
	if q.Query != nil {
		frameOptions.Query = *q.Query
	}

	frames, err := newAPLResponseFrameBuilder(q.Totals, q.IncludeTotalsTableFrame).BuildFrames(ctx, result, frameOptions)
	if err != nil {
		return nil, err
	}

	var response backend.DataResponse
	if shouldPrependLogsVolumeFrame(q, frames) {
		volumeResponse, err := d.queryLogsVolume(ctx, q, query, datasourceName)
		if err != nil {
			return nil, err
		}
		response.Frames = append(response.Frames, volumeResponse.Frames...)
	}
	response.Frames = append(response.Frames, frames...)
	return &response, nil
}

func shouldPrependLogsVolumeFrame(q *queryModel, frames []*data.Frame) bool {
	if !q.IncludeLogsVolumeFrame || len(frames) != 1 {
		return false
	}

	// Branch on the built frame, not the raw query text: APL that starts from a
	// log dataset can still aggregate into a normal time series or table.
	return frames[0].Meta != nil && frames[0].Meta.Type == data.FrameTypeLogLines
}

// queryMetrics executes an MPL query against the configured edge endpoint
func (d *Datasource) queryMetrics(ctx context.Context, q *queryModel, refID string, startTime, endTime time.Time, chartWidth int64) (*backend.DataResponse, error) {
	reqBody := axiomapi.MPLQueryRequest{
		MPL:        q.Query,
		StartTime:  startTime,
		EndTime:    endTime,
		ChartWidth: chartWidth,
	}

	res, err := d.api.QueryMetrics(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	var response backend.DataResponse
	frameBuilder := newMetricsFrameBuilder(res.Metadata, refID)

	for _, group := range res.Series {
		frame := frameBuilder.Build(group)
		applyAxiomTraceID(frame, res.TraceID)
		response.Frames = append(response.Frames, frame)
	}
	if q.IncludeTotalsTableFrame {
		tableFrame := frameBuilder.BuildTable(res.Series)
		applyAxiomTraceID(tableFrame, res.TraceID)
		response.Frames = append(response.Frames, tableFrame)
	}

	// extract the data from the response
	return &response, nil
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	err := d.api.ValidateCredentials(ctx)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:      backend.HealthStatusError,
			Message:     fmt.Sprintf("Failed to validate configuration: %s", err.Error()),
			JSONDetails: []byte(`{"error": "` + err.Error() + `"}`),
		}, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Configuration is valid and ready to use",
	}, nil
}
