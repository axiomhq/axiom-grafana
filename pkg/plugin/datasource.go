package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"time"

	"github.com/axiomhq/axiom-go/axiom"
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

	// opts, err := settings.HTTPClientOptions()
	// if err != nil {
	// 	return nil, fmt.Errorf("http client options: %w", err)
	// }

	accessToken := ""
	if token, exists := settings.DecryptedSecureJSONData["accessToken"]; exists {
		// Use the decrypted API key.
		// opts.Headers["Authorization"] = "Bearer " + accessToken
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

	// Important to reuse the same client for each query, to avoid using all available connections on a host
	// cl, err := httpclient.New(opts)
	// if err != nil {
	// 	return nil, fmt.Errorf("httpclient new: %w", err)
	// }
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
	var response backend.DataResponse

	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	err := json.Unmarshal(query.JSON, &qm)
	if err != nil {
		log.DefaultLogger.Info("failed to unmarshal query json")
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err.Error()))
	}

	// make request to axiom
	result, err := d.client.Query(ctx, qm.APL)
	if err != nil {
		log.DefaultLogger.Error("failed to get result from axiom")
		log.DefaultLogger.Error(err.Error())
		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("axiom error: %v", err.Error()))
	}

	// var labels data.Labels
	// var values []int64
	// var fields map[string][]int64

	// for _, s := range result.Buckets.Series {
	// 	for _, g := range s.Groups {
	// 		for k, v := range g.Group {
	// 			log.DefaultLogger.Info(fmt.Sprintf("label: %s, value: %v", k, v))
	// 			labels[k] = k
	// 			values = append(values, v.(int64))
	// 		}
	// 	}
	// }

	// for _, s := range result.Matches {
	// 	log.DefaultLogger.Info(fmt.Sprintf("data: %v", s.Data))
	// }

	// for _, t := range result.Buckets.Totals {
	// 	// for label, name := range t.Group {
	// 	// labels[label] = name.(string)
	// 	// }
	// 	for _, agg := range t.Aggregations {
	// 		// fields[agg.Alias] = append(fields[agg.Alias], agg.Value.(int64))
	// 		log.DefaultLogger.Info(fmt.Sprintf("agg: %v", agg.Value))
	// 		values = append(values, agg.Value.(int64))
	// 	}
	// }

	for _, series := range result.Buckets.Series {
		// create data frame response.
		// For an overview on data frames and how grafana handles them:
		// https://grafana.com/docs/grafana/latest/developers/plugins/data-frames/
		frame := data.NewFrame("response", data.NewField("time", nil, []time.Time{series.StartTime, series.EndTime}))

		for _, group := range series.Groups {
			for _, agg := range group.Aggregations {
				log.DefaultLogger.Info("value", agg.Value.(int64))
				frame.Fields = append(frame.Fields,
					data.NewField(agg.Alias, nil, []int64{agg.Value.(int64)}),
				)
			}
		}

		// add fields.
		// frame.Fields = append(frame.Fields,
		// 	data.NewField("time", nil, []time.Time{series.StartTime, series.EndTime}),
		// 	data.NewField("values", nil, []int64{10, 20}),
		// )

		// add the frames to the response.
		response.Frames = append(response.Frames, frame)
	}

	return response
}

// func (d *Datasource) query(ctx context.Context, host string, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
// 	var response backend.DataResponse

// 	// Unmarshal the JSON into our queryModel.
// 	var qm queryModel

// 	log.DefaultLogger.Info(string(query.JSON))

// 	err := json.Unmarshal(query.JSON, &qm)
// 	if err != nil {
// 		log.DefaultLogger.Info("failed to unmarshal query json")
// 		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("json unmarshal: %v", err.Error()))
// 	}

// 	// body, err := json.Marshal(query)
// 	// if err != nil {

// 	// 	return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("failed to decode query: %v", err.Error()))
// 	// }

// 	// make request to axiom
// 	req, err := http.NewRequestWithContext(ctx, http.MethodPost, host+"/datasets/_apl", bytes.NewReader(query.JSON))
// 	if err != nil {
// 		panic(err)
// 		// return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("%v", err.Error()))
// 	}

// 	axiomResp, err := d.httpClient.Do(req)
// 	if err != nil {
// 		log.DefaultLogger.Error(err.Error())
// 		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("%v", err.Error()))
// 	}

// 	log.DefaultLogger.Info(">>>", query.JSON)

// 	if axiomResp.StatusCode >= 400 {
// 		return backend.ErrDataResponse(backend.StatusInternal, fmt.Sprintf("http %d error: %v", axiomResp.StatusCode, err.Error()))
// 	}

// 	b, err := io.ReadAll(axiomResp.Body)
// 	log.DefaultLogger.Info(">>> %v\n%v\n", err, string(b))

// 	var result map[string]any
// 	err = json.NewDecoder(axiomResp.Body).Decode(&result)
// 	if err != nil {
// 		log.DefaultLogger.Error(err.Error())
// 		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("%v", err.Error()))
// 	}
// 	defer axiomResp.Body.Close()

// 	log.DefaultLogger.Info(fmt.Sprintf("result: %v", result))

// 	// create data frame response.
// 	// For an overview on data frames and how grafana handles them:
// 	// https://grafana.com/docs/grafana/latest/developers/plugins/data-frames/
// 	frame := data.NewFrame("response")

// 	// add fields.
// 	frame.Fields = append(frame.Fields,
// 		data.NewField("time", nil, []time.Time{query.TimeRange.From, query.TimeRange.To}),
// 		data.NewField("values", nil, []int64{10, 20}),
// 	)

// 	// add the frames to the response.
// 	response.Frames = append(response.Frames, frame)

// 	return response
// }

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
