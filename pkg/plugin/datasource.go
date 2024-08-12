package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/axiomhq/axiom-go/axiom"
	axiQuery "github.com/axiomhq/axiom-go/axiom/query"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
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

// NewDatasource creates a new datasource instance.
func NewDatasource(_ context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	accessToken := ""
	if token, exists := settings.DecryptedSecureJSONData["accessToken"]; exists {
		// Use the decrypted API key.
		accessToken = token
	}

	var data map[string]string
	err := json.Unmarshal(settings.JSONData, &data)
	if err != nil {
		return nil, err
	}
	host := "https://api.axiom.co"
	if apiHost, exists := data["apiHost"]; exists {
		host = apiHost
	}

	orgID := data["orgID"]

	client, err := axiom.NewClient(
		axiom.SetToken(accessToken),
		axiom.SetURL(host),
		axiom.SetOrganizationID(orgID),
		axiom.SetUserAgent(fmt.Sprintf("axiom-grafana/v%s", Version)),
	)
	if err != nil {
		return nil, err
	}

	ds := &Datasource{
		client:  client,
		apiHost: host,
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
	client  *axiom.Client
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
	// log panic
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
				log.DefaultLogger.Error(err.Error())
			}
			log.DefaultLogger.Error(err.Error())
			log.DefaultLogger.Error(string(debug.Stack()))
		}
	}()

	// create response struct
	response := backend.NewQueryDataResponse()

	type Job struct {
		query backend.DataQuery
	}

	type JobResponse struct {
		refID    string
		response backend.DataResponse
	}

	concurrencyLimit := 10 // limit to 10 concurrent requests

	jobCh := make(chan Job)
	responseCh := make(chan JobResponse)
	var wg sync.WaitGroup

	processJobsWorker := func() {
		for job := range jobCh {
			res := d.query(ctx, d.apiHost, req.PluginContext, job.query)
			responseCh <- JobResponse{
				refID:    job.query.RefID,
				response: res,
			}
		}
	}

	for i := 0; i < concurrencyLimit; i++ {
		go processJobsWorker()
	}

	go func() {
		for res := range responseCh {
			// save the response in a hashmap
			// based on with RefID as identifier
			response.Responses[res.refID] = res.response
			wg.Done()
		}
	}()

	for _, q := range req.Queries {
		wg.Add(1)
		jobCh <- Job{query: q}
	}

	// Wait for all queries to complete and then close the
	// channels so that all goroutines exit
	wg.Wait()
	close(responseCh)
	close(jobCh)

	return response, nil
}

type queryModel struct {
	APL    string `json:"apl"`
	Totals bool   `json:"totals"`
}

func (d *Datasource) query(ctx context.Context, host string, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {

	var response backend.DataResponse

	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		// Log the actual error since it will be included in the Grafana server log and return a more generic message to the end user.
		log.DefaultLogger.Error(err.Error())
		return backend.ErrDataResponse(backend.StatusInternal, "Could not parse query")
	}

	if qm.APL == "" {
		return backend.DataResponse{}
	}

	// make request to axiom
	result, err := d.QueryOverride(ctx, qm.APL, axiQuery.SetStartTime(query.TimeRange.From), axiQuery.SetEndTime(query.TimeRange.To))
	if err != nil {
		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("axiom error: %v", err.Error()))
	}

	var frame *data.Frame
	var newframe *data.Frame
	log.DefaultLogger.Info("totals", qm.Totals)
	log.DefaultLogger.Info("buckets", result.Tables[0].Buckets.Size)
	if len(result.Tables) > 0 {
		if qm.Totals {
			frame = buildFrameSeries(&result.Tables[1])
		} else {
			frame = buildFrameSeries(&result.Tables[0])
		}
		// Only convert longToWide if There is Aggregations
		newframe, err = data.LongToWide(frame, nil)
		if err != nil {
			log.DefaultLogger.Error("transformation from long to wide failed", err.Error())
		}
	} else {
		log.DefaultLogger.Info("buildFrameSeries for Matches")
		frame = buildFrameSeries(&result.Tables[0])
	}

	if newframe != nil {
		response.Frames = append(response.Frames, newframe)
	} else {
		response.Frames = append(response.Frames, frame)
	}

	return response
}

func buildFrameSeries(result *axiQuery.Table) *data.Frame {
	frame := data.NewFrame("response")

	// define fields
	fields := []*data.Field{}

	for _, f := range result.Fields {
		var field *data.Field
		switch f.Type {
		case axiQuery.TypeDateTime:
			field = data.NewField(f.Name, nil, []time.Time{})
			break
		case axiQuery.TypeLong, axiQuery.TypeInt:
			field = data.NewField(f.Name, nil, []float64{})
			break
		case axiQuery.TypeBool:
			field = data.NewField(f.Name, nil, []bool{})
			break
		case axiQuery.TypeTimespan:
			field = data.NewField(f.Name, nil, []int64{})
			break
		default:
			field = data.NewField(f.Name, nil, []string{})
		}

		fields = append(fields, field)
	}

	frame.Fields = fields

	for i := 0; i < len(result.Columns[0]); i++ {
		values := make([]any, 0, len(result.Fields))
		for colIndex, col := range result.Columns {
			switch result.Fields[colIndex].Type {
			case axiQuery.TypeDateTime:
				// parse time
				t, err := time.Parse(time.RFC3339, col[i].(string))
				if err != nil {
					log.DefaultLogger.Warn("Failed to parse time", "time", col[i])
					values = append(values, "")
					continue
				}
				values = append(values, t)
			case axiQuery.TypeInt:
				values = append(values, col[i])
			case axiQuery.TypeLong:
				values = append(values, col[i])
			default:
				values = append(values, fmt.Sprintf("%v", col[i]))
			}
		}
		frame.AppendRow(values...)
	}

	return frame
}

func walkMatch(m any, path []string, valFunc func(string, any)) {
	switch m := m.(type) {
	case map[string]any:
		for k, v := range m {
			if k == "" {
				// results returned by Axiom sometimes exist with an empty key at the end
				walkMatch(v, path, valFunc)
			} else {
				walkMatch(v, append(path, strings.ReplaceAll(k, `.`, `\.`)), valFunc)
			}
		}
	default:
		valFunc(strings.Join(path, "."), m)
	}
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	// first try to validate the credentials
	// NOTE: axiom-go doesn't do anything useful today
	err := d.client.ValidateCredentials(ctx)
	if err != nil {
		log.DefaultLogger.Error("Failed to validate credentials", "error", err)
		return &backend.CheckHealthResult{
			Status: backend.HealthStatusError,
			// simple error message, not the actual error
			Message: "error with datasource",
		}, nil
	}

	// perform an APL query that we expect to fail (empty)
	// validate that we get HTTP 400, this gives high confidence
	// that we got past network and authentication issues and looked at our request
	// it also should be somewhat inexpensive for the server
	var axiErr axiom.HTTPError
	var msg = "Did not receive expected error"
	_, err = d.client.Query(ctx, "")
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
		log.DefaultLogger.Error("Failed to query Axiom", "error", err)
		msg = "Failed to query Axiom"
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusError,
		Message: msg,
	}, nil
}
