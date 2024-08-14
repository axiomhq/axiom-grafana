package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/axiomhq/axiom-go/axiom"
	axiQuery "github.com/axiomhq/axiom-go/axiom/query"
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

// NewDatasource creates a new datasource instance.
func NewDatasource(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	logger := log.DefaultLogger.FromContext(ctx)
	accessToken := ""
	if token, exists := settings.DecryptedSecureJSONData["accessToken"]; exists {
		// Use the decrypted API key.
		accessToken = token
	}

	var data map[string]string
	err := json.Unmarshal(settings.JSONData, &data)
	if err != nil {
		logger.Error("failed to unmarshal settings", "error", err)
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
		logger.Error("failed to create axiom client", "error", err)
		return nil, err
	}

	ds := &Datasource{
		client:  client,
		apiHost: host,
	}
	resourceHandler := ds.newResourceHandler()
	ds.CallResourceHandler = resourceHandler

	logger.Info("datasource & client created", "host", host)

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

	return concurrent.QueryData(ctx, req, d.query, 10)
}

type queryModel struct {
	APL    string `json:"apl"`
	Totals bool   `json:"totals"`
}

func (d *Datasource) query(ctx context.Context, query concurrent.Query) backend.DataResponse {
	logger := log.DefaultLogger.FromContext(ctx)
	var response backend.DataResponse

	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	err := json.Unmarshal(query.DataQuery.JSON, &qm)
	if err != nil {
		// Log the actual error since it will be included in the Grafana server log and return a more generic message to the end user.
		logger.Error(err.Error())
		return backend.ErrDataResponse(backend.StatusInternal, "Could not parse query")
	}

	if qm.APL == "" {
		return backend.DataResponse{}
	}

	// make request to axiom
	result, err := d.client.Query(ctx, qm.APL, axiQuery.SetStartTime(query.DataQuery.TimeRange.From), axiQuery.SetEndTime(query.DataQuery.TimeRange.To))
	if err != nil {
		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("axiom error: %v", err.Error()))
	}

	var frame *data.Frame
	var newFrame *data.Frame
	if len(result.Tables) > 1 {
		if qm.Totals {
			frame = buildFrame(ctx, &result.Tables[1])
		} else {
			frame = buildFrame(ctx, &result.Tables[0])
		}
		// Only convert longToWide if There is Aggregations
		newFrame, err = data.LongToWide(frame, nil)
		if err != nil {
			logger.Warn("transformation from long to wide failed", err.Error())
		}
	} else {
		logger.Info("buildFrameSeries for Matches")
		frame = buildFrame(ctx, &result.Tables[0])
	}

	if newFrame != nil {
		response.Frames = append(response.Frames, newFrame)
	} else {
		response.Frames = append(response.Frames, frame)
	}

	return response
}

func buildFrame(ctx context.Context, result *axiQuery.Table) *data.Frame {
	logger := log.DefaultLogger.FromContext(ctx)
	frame := data.NewFrame("response")

	// define fields
	fields := []*data.Field{}

	for _, f := range result.Fields {
		var field *data.Field
		switch f.Type {
		case axiQuery.TypeDateTime:
			field = data.NewField(f.Name, nil, []time.Time{})
		case axiQuery.TypeLong, axiQuery.TypeInt:
			field = data.NewField(f.Name, nil, []*float64{})
		case axiQuery.TypeFloat:
			field = data.NewField(f.Name, nil, []*float64{})
		case axiQuery.TypeBool:
			field = data.NewField(f.Name, nil, []*bool{})
		case axiQuery.TypeTimespan:
			field = data.NewField(f.Name, nil, []*int64{})
		default:
			field = data.NewField(f.Name, nil, []*string{})
		}

		fields = append(fields, field)
	}

	for colIndex, col := range result.Columns {

		for i := 0; i < len(col); i++ {
			if col[i] == nil {
				fields[colIndex].Append(nil)
				continue
			}

			logger.Info(">>checking field type", "field", fields[colIndex].Name, "type", result.Fields[colIndex].Type.String(), "value", col[i])
			switch result.Fields[colIndex].Type {
			case axiQuery.TypeDateTime:
				// parse time
				t, err := time.Parse(time.RFC3339, col[i].(string))
				if err != nil {
					logger.Warn("Failed to parse time", "time", col[i])
					fields[colIndex].Append(time.Time{})
					continue
				}
				fields[colIndex].Append(t)
			case axiQuery.TypeInt:
				num := col[i].(float64)
				fields[colIndex].Append(&num)
			case axiQuery.TypeFloat:
				num := col[i].(float64)
				fields[colIndex].Append(&num)
			case axiQuery.TypeLong:
				num := col[i].(float64)
				fields[colIndex].Append(&num)
			case axiQuery.TypeString:
				txt := col[i].(string)
				fields[colIndex].Append(&txt)
			default:
				txt := fmt.Sprintf("%v", col[i])
				fields[colIndex].Append(&txt)
			}
		}

	}

	frame.Fields = fields

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
	logger := log.DefaultLogger.FromContext(ctx)
	// first try to validate the credentials
	// NOTE: axiom-go doesn't do anything useful today
	err := d.client.ValidateCredentials(ctx)
	if err != nil {
		logger.Error("Failed to validate credentials", "error", err)
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
		logger.Error("Failed to query Axiom", "error", err)
		msg = "Failed to query Axiom"
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusError,
		Message: msg,
	}, nil
}
