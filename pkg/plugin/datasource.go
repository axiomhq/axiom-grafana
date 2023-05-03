package plugin

import (
	"context"
	"encoding/json"
	"fmt"
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

	var data map[string]string
	err := json.Unmarshal(settings.JSONData, &data)
	if err != nil {
		return nil, err
	}
	host := "https://api.axiom.co"
	if apiHost, exists := data["apiHost"]; exists {
		host = apiHost
	}

	client, err := axiom.NewClient(
		axiom.SetAPITokenConfig(accessToken),
		axiom.SetURL(host),
	)
	if err != nil {
		return nil, err
	}

	return &Datasource{
		client:  client,
		apiHost: host,
	}, nil
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
	APL string `json: "apl"`
}

func (d *Datasource) query(ctx context.Context, host string, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {

	now := time.Now()

	var response backend.DataResponse

	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		log.DefaultLogger.Info("failed to unmarshal query json")
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err.Error()))
	}

	// make request to axiom
	result, err := d.client.Query(ctx, qm.APL, axiQuery.SetStartTime(query.TimeRange.From), axiQuery.SetEndTime(query.TimeRange.To))
	if err != nil {
		log.DefaultLogger.Error("failed to get result from axiom")
		log.DefaultLogger.Error(err.Error())
		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("axiom error: %v", err.Error()))
	}

	frame := data.NewFrame("response")

	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
				log.DefaultLogger.Error(err.Error())
			}
			log.DefaultLogger.Error(err.Error())
		}
	}()

	table := result.Tables[0]

	for index, f := range table.Fields {
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
			frame.Fields = append(frame.Fields, data.NewField(f.Name, nil, values))
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
			frame.Fields = append(frame.Fields, data.NewField(f.Name, nil, values))
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
			frame.Fields = append(frame.Fields, data.NewField(f.Name, nil, values))
		case axiQuery.TypeDateTime:
			// convert []string to []*time.Time
			times := make([]*time.Time, len(table.Columns[index]))
			for i, v := range table.Columns[index] {
				t, err := time.Parse(time.RFC3339, v.(string))
				if err != nil {
					log.DefaultLogger.Error("failed to parse time")
					log.DefaultLogger.Error(err.Error())
					return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("time parse error: %v", err.Error()))
				}
				times[i] = &t
			}
			frame.Fields = append(frame.Fields, data.NewField(f.Name, nil, times))
		case axiQuery.TypeTimespan:
		case axiQuery.TypeArray:
		case axiQuery.TypeDictionary:
		default:
			log.DefaultLogger.Error("unknown field type")
		}
	}

	frame, err = data.LongToWide(frame, nil)
	if err != nil {
		log.DefaultLogger.Error("failed to convert long to wide")
		log.DefaultLogger.Error(err.Error())
		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("long to wide error: %v", err.Error()))
	}

	response.Frames = append(response.Frames, frame)

	log.DefaultLogger.Info(fmt.Sprintf(">>> Query Duration: %v", time.Since(now)))

	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	// TODO: create a valid healthcheck
	var status = backend.HealthStatusOk
	var message = "Data source is working"

	return &backend.CheckHealthResult{
		Status:  status,
		Message: message,
	}, nil
}
