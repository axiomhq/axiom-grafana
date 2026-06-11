package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/axiomhq/axiom-go/axiom"
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

type queryModel struct {
	APL    *string `json:"apl"`
	Kind   *string `json:"kind"`
	Query  *string `json:"query"`
	Totals bool    `json:"totals"`
}

// NewDatasource creates a new datasource instance.
func NewDatasource(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	logger := log.DefaultLogger.FromContext(ctx)
	accessToken := ""
	if token, exists := settings.DecryptedSecureJSONData["accessToken"]; exists {
		// Use the decrypted API key.
		accessToken = token
	}

	var data map[string]any
	err := json.Unmarshal(settings.JSONData, &data)
	if err != nil {
		logger.Error("failed to unmarshal settings", "error", err)
		return nil, err
	}
	host := "https://api.axiom.co"
	if apiHost, exists := data["apiHost"]; exists {
		host = apiHost.(string)
	}

	edge := checkString(data["edge"])
	edgeURL := checkString(data["edgeURL"])

	baseUrl, err := resolveBaseUrl(urlInput{
		EdgeURL: edgeURL,
		Edge:    edge,
		APIHost: host,
	})
	if err != nil {
		logger.Error("failed to resolve correct axiom api/edge url", "error", err)
		return nil, err
	}
	client := newClient(baseUrl, accessToken)

	ds := &Datasource{
		api: &AxiomAPI{
			apiHost: baseUrl,
			client:  client,
		},
		apiHost: host,
		edge:    edge,
		edgeURL: edgeURL,
	}
	resourceHandler := ds.newResourceHandler()
	ds.CallResourceHandler = resourceHandler

	return ds, nil
}

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct {
	backend.CallResourceHandler
	apiHost string
	edge    string // Optional regional edge domain (e.g., "eu-central-1.aws.edge.axiom.co")
	edgeURL string // Optional explicit edge URL (takes precedence over edge)
	api     *AxiomAPI
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
	if kind == "mpl" {
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

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	logger := log.DefaultLogger.FromContext(ctx)
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
	var axiErr axiom.HTTPError
	var msg = "Did not receive expected error"
	r, err := http.NewRequest(http.MethodPost, "/v1/query/_apl", nil)
	_, err = d.api.client.http.Do(r)
	// _, err = d.client.Query(ctx, "")
	if err != nil && errors.As(err, &axiErr) {
		if axiErr.Status == 400 {
			// expected 400 for empty query, HEALTHY
			return &backend.CheckHealthResult{
				Status:  backend.HealthStatusOk,
				Message: "Data source is working",
			}, nil
		}
	}

	if err != nil {
		logger.Error("Failed to query Axiom", "error", err)
		msg = "Failed to query Axiom"
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusError,
		Message: msg,
	}, nil
}
