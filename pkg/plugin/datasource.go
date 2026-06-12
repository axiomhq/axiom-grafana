package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/axiomhq/axiom-grafana/pkg/axiomapi"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
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
	APL                 *string `json:"apl"`
	Kind                *string `json:"kind"`
	Query               *string `json:"query"`
	SupportingQueryType *string `json:"supportingQueryType"`
	Totals              bool    `json:"totals"`
}

// NewDatasource creates a new datasource instance.
func NewDatasource(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	logger := log.DefaultLogger.FromContext(ctx)
	config, err := parseConfig(ctx, settings)
	if err != nil {
		logger.Error("Failed to parse config", "error", err.Error())
		return nil, err
	}
	api := axiomapi.NewClient(axiomapi.Config{
		AccessToken: config.AccessToken,
		APIURL:      config.APIHost,
		EdgeURL:     config.EdgeURL,
		UserAgent:   fmt.Sprintf("axiom-grafana/v%s", Version),
	})

	ds := &Datasource{
		api: api,
	}
	resourceHandler := ds.newResourceHandler()
	ds.CallResourceHandler = resourceHandler

	return ds, nil
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

	if (qm.Query == nil || *qm.Query == "") && qm.APL != nil {
		qm.Query = qm.APL
	}

	if qm.Query == nil || *qm.Query == "" {
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
		queryResponse, err = d.queryMetrics(ctx, &qm, query.DataQuery.RefID, query.DataQuery.TimeRange.From, query.DataQuery.TimeRange.To)
	} else {
		queryResponse, err = d.queryEvents(ctx, &qm, query.DataQuery.TimeRange.From, query.DataQuery.TimeRange.To)
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

// queryEvents executes an APL query against the configured endpoint
// (edge or legacy apiHost, depending on configuration).
func (d *Datasource) queryEvents(ctx context.Context, q *queryModel, startTime, endTime time.Time) (*backend.DataResponse, error) {
	reqBody := axiomapi.APLQueryRequest{
		APL:       q.Query,
		StartTime: startTime,
		EndTime:   endTime,
	}

	result, err := d.api.QueryAPL(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	frameOptions := aplFrameOptions{
		FieldMetaByName: fieldMetaByNameForResponse(result),
		Status:          result.Status,
	}
	if q.Query != nil {
		frameOptions.Query = *q.Query
	}

	frame, err := newAPLResponseFrameBuilder(q.Totals).Build(ctx, result, frameOptions)
	if err != nil {
		return nil, err
	}

	var response backend.DataResponse
	response.Frames = append(response.Frames, frame)
	return &response, nil
}

// queryMetrics executes an MPL query against the configured edge endpoint
func (d *Datasource) queryMetrics(ctx context.Context, q *queryModel, refID string, startTime, endTime time.Time) (*backend.DataResponse, error) {
	reqBody := axiomapi.MPLQueryRequest{
		MPL:       q.Query,
		StartTime: startTime,
		EndTime:   endTime,
	}

	res, err := d.api.QueryMetrics(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	var response backend.DataResponse
	frameBuilder := newMetricsFrameBuilder(res.Metadata, refID)

	for _, group := range res.Series {
		response.Frames = append(response.Frames, frameBuilder.Build(group))
	}

	// extract the data from the response
	return &response, nil
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	// first try to validate the credentials
	// err := d.client.ValidateCredentials(ctx)
	// if err != nil {
	// 	logger.Error("Failed to validate credentials", "error", err)
	// 	return &backend.CheckHealthResult{
	// 		Status: backend.HealthStatusError,
	// 		// simple error message, not the actual error
	// 		Message: "error with datasource",
	// 	}, nil
	// }

	// perform an APL query that we expect to fail (empty)
	// validate that we get HTTP 400, this gives high confidence
	// that we got past network and authentication issues and looked at our request
	// it also should be somewhat inexpensive for the server
	// var msg = "Did not receive expected error"
	err := d.api.CheckHealth(ctx)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: err.Error(),
		}, nil
	}
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "OK",
	}, nil

}
