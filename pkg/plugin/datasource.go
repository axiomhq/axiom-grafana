package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
)

// NewDatasource creates a new datasource instance.
func NewDatasource(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	accessToken := ""
	if token, exists := settings.DecryptedSecureJSONData["accessToken"]; exists {
		// Use the decrypted API key.
		accessToken = token
	}

	if !IsPersonalToken(accessToken) {
		return nil, fmt.Errorf("personal token required")
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

	orgID, hasOrgID := data["orgID"]
	if !hasOrgID && len(orgID) > 0 {
		return nil, fmt.Errorf("orgID is required")
	}

	client, err := axiom.NewClient(
		axiom.SetToken(accessToken),
		axiom.SetURL(host),
		axiom.SetOrganizationID(orgID),
	)
	if err != nil {
		return nil, err
	}

	return &Datasource{
		client:  client,
		apiHost: host,
	}, nil
}

// IsPersonalToken returns true if the given token is a personal token.
func IsPersonalToken(token string) bool {
	return strings.HasPrefix(token, "xapt-")
}

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct {
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
	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := d.query(ctx, d.apiHost, req.PluginContext, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

type queryModel struct {
	APL    string `json:"apl"`
	Totals bool   `json:"totals"`
}

func (d *Datasource) query(ctx context.Context, host string, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
				log.DefaultLogger.Error(err.Error())
			}
			log.DefaultLogger.Error(err.Error())
		}
	}()

	var response backend.DataResponse

	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err.Error()))
	}

	// if query.QueryType == "schemaLookup" {
	// 	return d.schemaLookup(ctx)
	// }

	if qm.APL == "" {
		return backend.DataResponse{}
	}

	// make request to axiom
	result, err := d.QueryOverride(ctx, qm.APL, axiQuery.SetStartTime(query.TimeRange.From), axiQuery.SetEndTime(query.TimeRange.To))
	if err != nil {
		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("axiom error: %v", err.Error()))
	}

	log.DefaultLogger.Info("result", "result", result.Tables[0].Buckets.Field)

	// var frame *data.Frame
	frame := buildFrameTotals(&result.Result)

	// newFrame, err := data.LongToWide(frame, nil)
	// if err != nil {
	// 	log.DefaultLogger.Warn(err.Error())
	// 	// if conversion fails, return the original frame
	// 	newFrame = frame
	// }

	response.Frames = append(response.Frames, frame)

	return response
}

func (d *Datasource) schemaLookup(ctx context.Context) backend.DataResponse {
	var response backend.DataResponse
	frame := data.NewFrame("response")
	frame.Fields = []*data.Field{
		data.NewField("schema", nil, []string{}),
	}

	dsf, err := d.DatasetFields(ctx)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("axiom schema lookup error: %v", err.Error()))
	}

	dsfJSON, err := json.Marshal(dsf)
	if err != nil {
		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("schema marshal error: %v", err.Error()))
	}

	values := make([]any, 1)
	values[0] = string(dsfJSON)

	frame.AppendRow(values...)

	// newFrame, err := data.LongToWide(frame, nil)
	// if err != nil {
	// 	log.DefaultLogger.Warn("schemaLookup", "error", err.Error())
	// 	// if conversion fails, return the original frame
	// 	newFrame = frame
	// }

	response.Frames = append(response.Frames, frame)

	return response
}

// func buildFrameSeries(result *axiQuery.Result) *data.Frame {
// 	frame := data.NewFrame("response")

// 	// define fields
// 	fields := []*data.Field{
// 		data.NewField("_time", nil, []time.Time{}),
// 	}

// 	for group := range result.Buckets.Totals[0].Group {
// 		fields = append(fields,
// 			data.NewField(group, nil, []string{}),
// 		)
// 	}

// 	for _, agg := range result.Buckets.Totals[0].Aggregations {
// 		fields = append(fields,
// 			data.NewField(agg.Alias, nil, []*float64{}),
// 		)
// 	}

// 	frame.Fields = fields

// 	// FIXME: This is a hack
// 	for _, series := range result.Buckets.Series {
// 		for _, g := range series.Groups {
// 			values := make([]any, 0, 1+len(g.Group)+len(g.Aggregations)) // +1 for time
// 			values = append(values, series.StartTime)
// 			for _, field := range result.GroupBy {
// 				v := g.Group[field]
// 				// convert v to string regardless of type
// 				strV := fmt.Sprintf("%v", v)
// 				values = append(values, strV)
// 			}
// 			for _, agg := range g.Aggregations {
// 				v := agg.Value
// 				switch v := v.(type) {
// 				case float64:
// 					values = append(values, &v)
// 				default:
// 					values = append(values, nil)
// 				}
// 			}
// 			frame.AppendRow(values...)
// 		}
// 	}
// 	return frame
// }

func buildFrameTotals(result *axiQuery.Result) *data.Frame {
	frame := data.NewFrame("response")

	// define fields
	// var fields []*data.Field
	for index, f := range result.Tables[0].Fields {
		log.DefaultLogger.Info("table 0", "index", index, "field", f.Name)
	}

	table := result.Tables[0]
	fields := make([]*data.Field, 0, len(table.Fields))

	for index, f := range table.Fields {
		log.DefaultLogger.Info("table 1", "index", index, "field", f.Name)

		switch f.Type {
		case axiQuery.TypeString:
			values := make([]string, 0, len(table.Columns[index]))
			iter := table.Columns[index].Values()
			iter.Range(context.Background(), func(c context.Context, v any) error {
				str, ok := v.(string)
				if !ok {
					log.DefaultLogger.Error("failed to convert value to string", "field", f.Name)
					// return error
					return fmt.Errorf("failed to convert value to string")
				}
				values = append(values, str)
				return nil
			})
			//frame.Fields = append(frame.Fields, data.NewField(f.Name, nil, values))
			fields = append(fields, data.NewField(f.Name, nil, values))
		case axiQuery.TypeInt:
			values := make([]float64, 0, len(table.Columns[index]))
			iter := table.Columns[index].Values()
			iter.Range(context.Background(), func(c context.Context, v any) error {
				num, ok := v.(float64)
				if !ok {
					log.DefaultLogger.Error("failed to convert value to int")
					// return error
					return fmt.Errorf("failed to convert value to int")
				}
				values = append(values, num)
				return nil
			})
			fields = append(fields, data.NewField(f.Name, nil, values))
			//frame.Fields = append(frame.Fields, data.NewField(f.Name, nil, values))
		case axiQuery.TypeReal, axiQuery.TypeLong:
			values := make([]float64, 0, len(table.Columns[index]))
			iter := table.Columns[index].Values()
			iter.Range(context.Background(), func(c context.Context, v any) error {
				num, ok := v.(float64)
				if !ok {
					log.DefaultLogger.Error("failed to convert value to real")
					// return error
					return fmt.Errorf("failed to convert value to real")
				}
				values = append(values, num)
				return nil
			})
			fields = append(fields, data.NewField(f.Name, nil, values))
			//frame.Fields = append(frame.Fields, data.NewField(f.Name, nil, values))
		case axiQuery.TypeDateTime:
			// convert []string to []*time.Time
			times := make([]*time.Time, len(table.Columns[index]))
			for i, v := range table.Columns[index] {
				t, err := time.Parse(time.RFC3339, v.(string))
				if err != nil {
					log.DefaultLogger.Error("failed to parse time")
					log.DefaultLogger.Error(err.Error())
					// return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("time parse error: %v", err.Error()))
					continue
				}
				times[i] = &t
			}
			fields = append(fields, data.NewField(f.Name, nil, values))
			//frame.Fields = append(frame.Fields, data.NewField(f.Name, nil, times))
		case axiQuery.TypeTimespan:
		case axiQuery.TypeArray:
		case axiQuery.TypeDictionary:
		default:
			log.DefaultLogger.Error("unknown field type", "type", f.Type, "name", f.Name)
		}
	}

	// for index, f := range result.Tables[0].Fields {
	// 	log.DefaultLogger.Info("found field", "name", f.Name)
	// 	fields = append(fields,
	// 		data.NewField(f.Name, nil, result.Tables[0].Columns[index].Values()),
	// 	)
	// }

	// for _, agg := range result.Tables[1]. {
	// 	fields = append(fields,
	// 		data.NewField(agg.Alias, nil, []*float64{}),
	// 	)
	// }

	// frame.Fields = fields

	// for _, g := range result.Tables[0].Groups {
	// 	values := make([]any, 0, len(g.Group)+len(g.Aggregations))
	// 	for _, field := range result.GroupBy {
	// 		v := g.Group[field]
	// 		// convert v to string regardless of type
	// 		strV := fmt.Sprintf("%v", v)
	// 		values = append(values, strV)
	// 	}
	// 	for _, agg := range g.Aggregations {
	// 		v := agg.Value
	// 		switch v := v.(type) {
	// 		case float64:
	// 			values = append(values, &v)
	// 		default:
	// 			values = append(values, nil)
	// 		}
	// 	}
	// 	frame.AppendRow(values...)
	// }

	return frame
}

// func buildFrameMatches(result *AplQueryResponse) *data.Frame {
// 	frame := data.NewFrame("response").SetMeta(&data.FrameMeta{
// 		PreferredVisualization: data.VisTypeTable,
// 	})

// 	// define fields
// 	for _, proj := range result.Tables {
// 		for _, f := range proj.Fields {
// 			switch f.Name {
// 			case "_time", "_sysTime":
// 				frame.Fields = append(frame.Fields, data.NewField(f.Name, nil, []time.Time{}))
// 			default:
// 				frame.Fields = append(frame.Fields, data.NewField(f.Name, nil, []string{}))
// 			}
// 		}
// 	}

// 	for _, match := range result.Tables[1].Buckets {
// 		// convert structure to map of field values
// 		vals := make(map[string]any)
// 		walkMatch(match.Data, nil, func(k string, v any) {
// 			vals[k] = fmt.Sprintf("%v", v)
// 		})

// 		// build values
// 		values := make([]any, 0, len(frame.Fields))
// 		for _, field := range frame.Fields {
// 			switch field.Name {
// 			case "_time":
// 				values = append(values, match.Time)
// 			case "_sysTime":
// 				values = append(values, match.SysTime)
// 			default:
// 				values = append(values, vals[field.Name])
// 			}
// 		}

// 		frame.AppendRow(values...)
// 	}

// 	return frame
// }

func walkMatch(m any, path []string, valFunc func(string, any)) {
	switch m := m.(type) {
	case map[string]any:
		for k, v := range m {
			if k == "" {
				// results returned by Axiom sometimes exist with an empty key at the end
				walkMatch(v, path, valFunc)
			} else {
				walkMatch(v, append(path, k), valFunc)
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
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("error validating credentials: %v", err),
		}, nil
	}

	// perform an APL query that we expect to fail (empty)
	// validate that we get HTTP 400, this gives high confidence
	// that we got past network and authentication issues and looked at our request
	// it also should be somewhat inexpensive for the server
	var axiErr *axiom.Error
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
		msg = err.Error()
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusError,
		Message: msg,
	}, nil
}
