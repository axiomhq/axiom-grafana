package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/axiomhq/axiom-go/axiom/query"
	"github.com/axiomhq/axiom-grafana/pkg/axiomapi"
	"github.com/axiomhq/axiom-grafana/pkg/config"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestQueryData(t *testing.T) {
	ds := Datasource{}

	resp, err := ds.QueryData(
		context.Background(),
		&backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{RefID: "A", JSON: json.RawMessage(`{}`)},
				{RefID: "B", JSON: json.RawMessage(`{}`)},
				{RefID: "C", JSON: json.RawMessage(`{}`)},
			},
		},
	)
	require.NoError(t, err)

	for _, res := range resp.Responses {
		require.NoError(t, res.Error)
	}

	require.Len(t, resp.Responses, 3, "QueryData must return a response for each query")
}

func TestQueryDataSkipsEmptyQueries(t *testing.T) {
	requestCount := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer upstream.Close()

	ds := Datasource{
		api: newTestAxiomClient(t, upstream.URL, upstream.URL),
	}

	resp, err := ds.QueryData(
		context.Background(),
		&backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{RefID: "A", JSON: json.RawMessage(`{"kind":"apl","query":""}`)},
				{RefID: "B", JSON: json.RawMessage(`{"kind":"apl","query":"   "}`)},
				{RefID: "C", JSON: json.RawMessage(`{"kind":"mpl","query":"\n\t"}`)},
				{RefID: "D", JSON: json.RawMessage(`{"kind":"apl","apl":"   "}`)},
				{RefID: "E", JSON: json.RawMessage(`{"kind":"apl","query":"// Enter an APL query (run with Ctrl/Cmd+Enter)"}`)},
				{RefID: "F", JSON: json.RawMessage(`{"kind":"mpl","query":"// Enter an APL query (run with Ctrl/Cmd+Enter)"}`)},
				{RefID: "G", JSON: json.RawMessage(`{"kind":"apl","query":"\n  // only a comment\n\t"}`)},
			},
		},
	)
	require.NoError(t, err)
	require.Zero(t, requestCount)
	require.Len(t, resp.Responses, 7)

	for _, res := range resp.Responses {
		require.NoError(t, res.Error)
		require.Empty(t, res.Frames)
	}
}

func TestResourceHandlerFetchesEscapedMetricAutocompleteValues(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.EscapedPath() {
		case "/v1/query/metrics/info/datasets/team%2Fprod/metrics":
			_, err := w.Write([]byte(`["http.requests/total"]`))
			require.NoError(t, err)
		case "/v1/query/metrics/info/datasets/team%2Fprod/tags":
			_, err := w.Write([]byte(`["service.name","host.name"]`))
			require.NoError(t, err)
		case "/v1/query/metrics/info/datasets/team%2Fprod/tags/service.name/values":
			_, err := w.Write([]byte(`["api","worker"]`))
			require.NoError(t, err)
		case "/v1/query/metrics/info/datasets/team%2Fprod/metrics/http.requests%2Ftotal/tags":
			_, err := w.Write([]byte(`["service.name"]`))
			require.NoError(t, err)
		case "/v1/query/metrics/info/datasets/team%2Fprod/metrics/http.requests%2Ftotal/tags/service.name/values":
			_, err := w.Write([]byte(`["api"]`))
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected upstream path: %s", r.URL.EscapedPath())
		}
	}))
	defer upstream.Close()

	ds := Datasource{
		api: newTestAxiomClient(t, upstream.URL, upstream.URL),
	}
	handler := ds.newResourceHandler()

	metricsResp := callResource(t, handler, "/datasets/team%2Fprod/metrics")
	require.Equal(t, http.StatusOK, metricsResp.Status)
	require.JSONEq(t, `["http.requests/total"]`, string(metricsResp.Body))

	datasetTagsResp := callResource(t, handler, "/datasets/team%2Fprod/tags")
	require.Equal(t, http.StatusOK, datasetTagsResp.Status)
	require.JSONEq(t, `["service.name","host.name"]`, string(datasetTagsResp.Body))

	datasetTagValuesResp := callResource(t, handler, "/datasets/team%2Fprod/tags/service.name/values")
	require.Equal(t, http.StatusOK, datasetTagValuesResp.Status)
	require.JSONEq(t, `["api","worker"]`, string(datasetTagValuesResp.Body))

	tagsResp := callResource(t, handler, "/datasets/team%2Fprod/metrics/http.requests%2Ftotal/tags")
	require.Equal(t, http.StatusOK, tagsResp.Status)
	require.JSONEq(t, `["service.name"]`, string(tagsResp.Body))

	metricTagValuesResp := callResource(t, handler, "/datasets/team%2Fprod/metrics/http.requests%2Ftotal/tags/service.name/values")
	require.Equal(t, http.StatusOK, metricTagValuesResp.Status)
	require.JSONEq(t, `["api"]`, string(metricTagValuesResp.Body))
}

func TestLogsVolumeAPLUsesTimeBeforeSysTime(t *testing.T) {
	got := logsVolumeAPL("['logs'] | where level == 'error';", time.Minute)

	require.Contains(t, got, "coalesce(_time, _sysTime)")
	require.Contains(t, got, "bin(_axiom_logs_volume_time, 1m)")
	require.NotContains(t, got, ";)")
}

func TestLogsVolumeFrameBuilderUsesTimeBeforeSysTime(t *testing.T) {
	frame, err := newLogsVolumeFrameBuilder(
		backend.DataQuery{
			RefID:     "log-volume-A",
			TimeRange: backend.TimeRange{From: time.Date(2026, 6, 11, 2, 0, 0, 0, time.UTC), To: time.Date(2026, 6, 11, 3, 0, 0, 0, time.UTC)},
		},
		"Axiom",
		"['logs']",
	).Build(axiomapi.APLQueryResponse{
		Tables: []query.Table{
			{
				Fields: []query.Field{
					{Name: "_sysTime", Type: "datetime"},
					{Name: "_time", Type: "datetime"},
					{Name: "count_", Type: "integer"},
				},
				Columns: []query.Column{
					{"2026-06-11T02:19:39Z"},
					{"2026-06-11T02:20:39Z"},
					{3},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, frame.Fields, 2)
	timestamp, ok := frame.Fields[0].At(0).(time.Time)
	require.True(t, ok)
	require.Equal(t, time.Date(2026, 6, 11, 2, 20, 39, 0, time.UTC), timestamp)
}

func TestQueryLogsVolumeReturnsFullRangeHistogramFrame(t *testing.T) {
	start := time.Date(2026, 6, 11, 2, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 11, 3, 0, 0, 0, time.UTC)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/query/_apl", r.URL.Path)
		require.Equal(t, "tabular", r.URL.Query().Get("format"))

		var body axiomapi.APLQueryRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.NotNil(t, body.APL)
		require.True(t, strings.Contains(*body.APL, "coalesce(_time, _sysTime)"))
		require.True(t, strings.Contains(*body.APL, "bin(_axiom_logs_volume_time, 5m)"))
		require.Equal(t, start, body.StartTime)
		require.Equal(t, end, body.EndTime)

		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{
			"format":"tabular",
			"tables":[{
				"fields":[{"name":"_time","type":"datetime"},{"name":"count_","type":"integer"}],
				"columns":[["2026-06-11T02:00:00Z","2026-06-11T02:05:00Z"],[3,7]]
			}]
		}`))
		require.NoError(t, err)
	}))
	defer upstream.Close()

	queryText := "['logs'] | where level == 'error'"
	ds := Datasource{
		api: newTestAxiomClient(t, upstream.URL, upstream.URL),
	}

	resp, err := ds.queryLogsVolume(
		context.Background(),
		&queryModel{Query: &queryText, SupportingQueryType: stringPtr(supplementaryQueryTypeLogsVolume)},
		backend.DataQuery{
			RefID:         "log-volume-A",
			QueryType:     logsVolumeQueryType,
			TimeRange:     backend.TimeRange{From: start, To: end},
			Interval:      5 * time.Minute,
			MaxDataPoints: 100,
		},
		"Axiom",
	)
	require.NoError(t, err)
	require.Len(t, resp.Frames, 1)

	frame := resp.Frames[0]
	require.Equal(t, "Logs volume", frame.Name)
	require.Equal(t, "log-volume-A", frame.RefID)
	require.NotNil(t, frame.Meta)
	require.Equal(t, data.FrameTypeTimeSeriesWide, frame.Meta.Type)
	require.Equal(t, data.FrameTypeVersion{0, 1}, frame.Meta.TypeVersion)
	require.EqualValues(t, data.VisTypeGraph, frame.Meta.PreferredVisualization)
	require.Len(t, frame.Fields, 2)
	require.Equal(t, data.FieldTypeTime, frame.Fields[0].Type())
	require.Equal(t, data.FieldTypeNullableFloat64, frame.Fields[1].Type())

	custom, ok := frame.Meta.Custom.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "FullRange", custom["logsVolumeType"])
	require.Equal(t, "Axiom", custom["datasourceName"])
}

func TestMPLQuerySendsChartWidthHeader(t *testing.T) {
	start := time.Date(2026, 6, 11, 2, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 11, 3, 0, 0, 0, time.UTC)
	queryText := "fetch cpu"
	requestCount := 0

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		require.Equal(t, "/v1/query/_mpl", r.URL.Path)
		require.Equal(t, "734", r.Header.Get("x-axiom-chart-width"))

		var body axiomapi.MPLQueryRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.NotNil(t, body.MPL)
		require.Equal(t, queryText, *body.MPL)
		require.Equal(t, start, body.StartTime)
		require.Equal(t, end, body.EndTime)
		require.Zero(t, body.ChartWidth)

		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"metadata":{"unit":"","warnings":[]},"series":[]}`))
		require.NoError(t, err)
	}))
	defer upstream.Close()

	ds := Datasource{
		api: newTestAxiomClient(t, upstream.URL, upstream.URL),
	}

	resp, err := ds.QueryData(
		context.Background(),
		&backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{
					RefID:         "A",
					TimeRange:     backend.TimeRange{From: start, To: end},
					MaxDataPoints: 734,
					JSON:          json.RawMessage(`{"kind":"mpl","query":"fetch cpu"}`),
				},
			},
		},
	)
	require.NoError(t, err)
	require.Equal(t, 1, requestCount)
	require.NoError(t, resp.Responses["A"].Error)
}

func TestMPLQueryReturnsExploreTableFrameWhenRequested(t *testing.T) {
	start := time.Date(2026, 6, 11, 2, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 11, 3, 0, 0, 0, time.UTC)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/query/_mpl", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{
			"metadata":{"unit":"short","warnings":[]},
			"series":[
				{"resolution":60,"start":1781186400,"metric":"cpu","tags":{"__label":"api","pod.name":"api-7d9"},"data":[0.1,0.2]},
				{"resolution":60,"start":1781186400,"metric":"memory","tags":{"__label":"api","pod.name":"api-7d9"},"data":[512]},
				{"resolution":60,"start":1781186400,"metric":"cpu","tags":{"__label":"worker","pod.name":"worker-5f8"},"data":[0.7]}
			]
		}`))
		require.NoError(t, err)
	}))
	defer upstream.Close()

	ds := Datasource{
		api: newTestAxiomClient(t, upstream.URL, upstream.URL),
	}

	resp, err := ds.QueryData(
		context.Background(),
		&backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{
					RefID:     "A",
					TimeRange: backend.TimeRange{From: start, To: end},
					JSON:      json.RawMessage(`{"kind":"mpl","query":"fetch cpu","includeTotalsTableFrame":true}`),
				},
			},
		},
	)
	require.NoError(t, err)
	queryResp := resp.Responses["A"]
	require.NoError(t, queryResp.Error)
	require.Len(t, queryResp.Frames, 4)

	tableFrame := queryResp.Frames[3]
	require.NotNil(t, tableFrame.Meta)
	require.EqualValues(t, data.VisTypeTable, tableFrame.Meta.PreferredVisualization)
	require.Len(t, tableFrame.Fields, 4)
	require.Equal(t, "__label", tableFrame.Fields[0].Name)
	require.Equal(t, "pod.name", tableFrame.Fields[1].Name)
	require.Equal(t, "cpu", tableFrame.Fields[2].Name)
	require.Equal(t, "memory", tableFrame.Fields[3].Name)
	require.Equal(t, "api", *tableFrame.Fields[0].At(0).(*string))
	require.Equal(t, "worker", *tableFrame.Fields[0].At(1).(*string))
	require.Equal(t, "api-7d9", *tableFrame.Fields[1].At(0).(*string))
	require.Equal(t, "worker-5f8", *tableFrame.Fields[1].At(1).(*string))
	require.Equal(t, 0.2, *tableFrame.Fields[2].At(0).(*float64))
	require.Equal(t, 0.7, *tableFrame.Fields[2].At(1).(*float64))
	require.Equal(t, float64(512), *tableFrame.Fields[3].At(0).(*float64))
	require.Nil(t, tableFrame.Fields[3].At(1))
}

func TestQueryEventsPrependsLogsVolumeFrameForPanelLogQueries(t *testing.T) {
	start := time.Date(2026, 6, 11, 2, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 11, 3, 0, 0, 0, time.UTC)
	requestCount := 0

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/query/_apl", r.URL.Path)
		require.Equal(t, "tabular", r.URL.Query().Get("format"))

		var body axiomapi.APLQueryRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.NotNil(t, body.APL)
		require.Equal(t, start, body.StartTime)
		require.Equal(t, end, body.EndTime)

		w.Header().Set("Content-Type", "application/json")
		requestCount++
		switch requestCount {
		case 1:
			w.Header().Set("X-Axiom-Trace-Id", "main-trace")
			require.Equal(t, "['logs']", *body.APL)
			_, err := w.Write([]byte(`{
				"format":"tabular",
				"tables":[{
					"fields":[{"name":"_time","type":"datetime"},{"name":"message","type":"string"},{"name":"level","type":"string"}],
					"columns":[["2026-06-11T02:00:00Z"],["hello"],["info"]]
				}]
			}`))
			require.NoError(t, err)
		case 2:
			w.Header().Set("X-Axiom-Trace-Id", "logs-volume-trace")
			require.Contains(t, *body.APL, "summarize count_ = count()")
			require.Contains(t, *body.APL, "bin(_axiom_logs_volume_time, 5m)")
			_, err := w.Write([]byte(`{
				"format":"tabular",
				"tables":[{
					"fields":[{"name":"_time","type":"datetime"},{"name":"count_","type":"integer"}],
					"columns":[["2026-06-11T02:00:00Z"],[1]]
				}]
			}`))
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected APL request %d", requestCount)
		}
	}))
	defer upstream.Close()

	queryText := "['logs']"
	ds := Datasource{
		api: newTestAxiomClient(t, upstream.URL, upstream.URL),
	}

	resp, err := ds.queryEvents(
		context.Background(),
		&queryModel{Query: &queryText, IncludeLogsVolumeFrame: true},
		backend.DataQuery{
			RefID:         "A",
			TimeRange:     backend.TimeRange{From: start, To: end},
			Interval:      5 * time.Minute,
			MaxDataPoints: 100,
		},
		"Axiom",
	)
	require.NoError(t, err)
	require.Equal(t, 2, requestCount)
	require.Len(t, resp.Frames, 2)

	require.Equal(t, "Logs volume", resp.Frames[0].Name)
	require.Equal(t, data.FrameTypeTimeSeriesWide, resp.Frames[0].Meta.Type)
	require.EqualValues(t, data.VisTypeGraph, resp.Frames[0].Meta.PreferredVisualization)
	logsVolumeCustom, ok := resp.Frames[0].Meta.Custom.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "logs-volume-trace", logsVolumeCustom["axiomTraceId"])

	require.Equal(t, "Logs", resp.Frames[1].Name)
	require.Equal(t, data.FrameTypeLogLines, resp.Frames[1].Meta.Type)
	require.EqualValues(t, data.VisTypeLogs, resp.Frames[1].Meta.PreferredVisualization)
	logsCustom, ok := resp.Frames[1].Meta.Custom.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "main-trace", logsCustom["axiomTraceId"])
}

func callResource(t *testing.T, handler backend.CallResourceHandler, path string) *backend.CallResourceResponse {
	t.Helper()

	var resp *backend.CallResourceResponse
	err := handler.CallResource(
		context.Background(),
		&backend.CallResourceRequest{
			Method: http.MethodGet,
			Path:   path,
			URL:    path,
		},
		backend.CallResourceResponseSenderFunc(func(r *backend.CallResourceResponse) error {
			resp = r
			return nil
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, resp)

	return resp
}

func newTestAxiomClient(t *testing.T, apiHost, edgeURL string) *axiomapi.Client {
	t.Helper()

	timeouts := httpclient.DefaultTimeoutOptions
	client, err := axiomapi.NewClient(
		httpclient.Options{Timeouts: &timeouts},
		&config.PluginConfig{
			APIHost: apiHost,
			EdgeURL: edgeURL,
		},
	)
	require.NoError(t, err)

	return client
}

func TestBuildFrame(t *testing.T) {
	tests := []struct {
		name        string
		aplResponse string
		expect      *data.Frame
	}{
		{
			name:        "example",
			aplResponse: `{"format":"tabular","status":{"elapsedTime":760311,"blocksExamined":2672,"rowsExamined":167932295,"rowsMatched":167932295,"numGroups":0,"isPartial":false,"cacheStatus":1,"minBlockTime":"2021-11-29T18:55:53.248Z","maxBlockTime":"2024-08-15T10:09:06.396Z","messages":[{"priority":"warn","count":1,"code":"apl_implicitendtimeofnowapplied_1","msg":"line: 1, col: 20: implicit end time of 'now' applied"}]},"tables":[{"name":"0","sources":[{"name":"vercel"}],"fields":[{"name":"_time","type":"datetime"},{"name":"request.method","type":"string"},{"name":"count_","type":"integer","agg":{"name":"count"}}],"order":[{"field":"_time","desc":false},{"field":"count_","desc":true}],"groups":[{"name":"_time"},{"name":"request.method"}],"range":{"field":"_time","start":"2013-12-21T00:00:00Z","end":"2024-12-18T00:00:00Z"},"buckets":{"field":"_time","size":31536000000000000},"columns":[["2020-12-19T00:00:00Z","2020-12-19T00:00:00Z","2020-12-19T00:00:00Z","2020-12-19T00:00:00Z","2020-12-19T00:00:00Z","2020-12-19T00:00:00Z","2021-12-19T00:00:00Z","2021-12-19T00:00:00Z","2021-12-19T00:00:00Z","2021-12-19T00:00:00Z","2021-12-19T00:00:00Z","2021-12-19T00:00:00Z","2021-12-19T00:00:00Z","2021-12-19T00:00:00Z","2021-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2023-12-19T00:00:00Z","2023-12-19T00:00:00Z","2023-12-19T00:00:00Z","2023-12-19T00:00:00Z","2023-12-19T00:00:00Z","2023-12-19T00:00:00Z","2023-12-19T00:00:00Z","2023-12-19T00:00:00Z","2023-12-19T00:00:00Z"],["GET",null,"POST","HEAD","PUT","DELETE","GET",null,"HEAD","POST","PUT","DELETE","OPTIONS","PROPFIND","PATCH","GET",null,"POST","HEAD","OPTIONS","PUT","DELETE","CONNECT","PROPFIND","PATCH","GET",null,"POST","HEAD","OPTIONS","PUT","DELETE","PATCH","PROPFIND"],[397262,2753,1608,334,209,70,20591383,1882923,163479,21831,656,280,29,9,1,68114522,7325479,3953165,3296763,8989,1010,419,25,24,24,50657014,8358670,2480118,668371,3377,949,484,26,9]]},{"name":"_totals","sources":[{"name":"vercel"}],"fields":[{"name":"request.method","type":"string"},{"name":"count_","type":"integer","agg":{"name":"count"}}],"order":[{"field":"count_","desc":true}],"groups":[{"name":"request.method"}],"range":{"field":"_time","start":"2013-12-21T00:00:00Z","end":"2024-12-18T00:00:00Z"},"columns":[["GET",null,"POST","HEAD","OPTIONS","PUT","DELETE","PATCH","PROPFIND","CONNECT"],[139760181,17569825,6456722,4128947,12395,2824,1253,51,42,25]]}],"datasetNames":["vercel"],"fieldsMetaMap":{"vercel":[{"name":"report.durationMs","type":"float","unit":"ms","hidden":false,"description":""},{"name":"report.maxMemoryUsedMb","type":"integer","unit":"decmbytes","hidden":false,"description":""},{"name":"webVital.value","type":"integer|float","unit":"ms","hidden":false,"description":""}]}}`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			var queryRes query.Result
			err := json.Unmarshal([]byte(test.aplResponse), &queryRes)
			require.NoError(t, err)

			got, err := buildAPLFrame(context.Background(), &queryRes.Tables[0])
			require.NoError(t, err)
			t.Logf("%#v", got)
		})
	}
}

func TestAPLResponseFrameBuilderBuildsTimeSeriesFrame(t *testing.T) {
	v1 := float64(100)
	v2 := float64(200)
	v3 := float64(110)
	result := axiomapi.APLQueryResponse{
		Tables: []query.Table{
			{
				Fields: []query.Field{
					{Name: "_time", Type: "datetime"},
					{Name: "Lambda Name", Type: "string"},
					{Name: "Duration", Type: "float"},
				},
				Columns: []query.Column{
					{"2026-06-11T13:45:00Z", "2026-06-11T13:50:00Z", "2026-06-11T13:55:00Z"},
					{"a", "b", "a"},
					{v1, v2, v3},
				},
			},
			{
				Fields: []query.Field{
					{Name: "Lambda Name", Type: "string"},
					{Name: "Duration", Type: "float"},
				},
				Columns: []query.Column{
					{"a", "b"},
					{v3, v2},
				},
			},
		},
	}

	got, err := newAPLResponseFrameBuilder(false).Build(context.Background(), result, aplFrameOptions{})
	require.NoError(t, err)
	require.NotNil(t, got.Meta)
	require.EqualValues(t, data.VisTypeGraph, got.Meta.PreferredVisualization)
	require.Len(t, got.Fields, 3)
}

func TestAPLResponseFrameBuilderBuildFramesReturnsGraphOnlyForTimeSeriesByDefault(t *testing.T) {
	v1 := float64(100)
	v2 := float64(200)
	result := axiomapi.APLQueryResponse{
		Tables: []query.Table{
			{
				Fields: []query.Field{
					{Name: "_time", Type: "datetime"},
					{Name: "method", Type: "string"},
					{Name: "count_", Type: "integer"},
				},
				Columns: []query.Column{
					{"2026-06-11T13:45:00Z", "2026-06-11T13:45:00Z"},
					{"GET", "POST"},
					{v1, v2},
				},
			},
			{
				Fields: []query.Field{
					{Name: "method", Type: "string"},
					{Name: "count_", Type: "integer"},
				},
				Columns: []query.Column{
					{"GET", "POST"},
					{v1, v2},
				},
			},
		},
	}

	frames, err := newAPLResponseFrameBuilder(false).BuildFrames(context.Background(), result, aplFrameOptions{})
	require.NoError(t, err)
	require.Len(t, frames, 1)
	require.NotNil(t, frames[0].Meta)
	require.Equal(t, data.FrameTypeTimeSeriesWide, frames[0].Meta.Type)
	require.EqualValues(t, data.VisTypeGraph, frames[0].Meta.PreferredVisualization)
}

func TestAPLResponseFrameBuilderBuildFramesReturnsTotalsTableWhenRequested(t *testing.T) {
	v1 := float64(100)
	v2 := float64(200)
	result := axiomapi.APLQueryResponse{
		Tables: []query.Table{
			{
				Fields: []query.Field{
					{Name: "_time", Type: "datetime"},
					{Name: "method", Type: "string"},
					{Name: "count_", Type: "integer"},
				},
				Columns: []query.Column{
					{"2026-06-11T13:45:00Z", "2026-06-11T13:45:00Z"},
					{"GET", "POST"},
					{v1, v2},
				},
			},
			{
				Fields: []query.Field{
					{Name: "method", Type: "string"},
					{Name: "count_", Type: "integer"},
				},
				Columns: []query.Column{
					{"GET", "POST"},
					{v1, v2},
				},
			},
		},
	}

	frames, err := newAPLResponseFrameBuilder(false, true).BuildFrames(context.Background(), result, aplFrameOptions{})
	require.NoError(t, err)
	require.Len(t, frames, 2)

	require.NotNil(t, frames[0].Meta)
	require.Equal(t, data.FrameTypeTimeSeriesWide, frames[0].Meta.Type)
	require.EqualValues(t, data.VisTypeGraph, frames[0].Meta.PreferredVisualization)

	require.NotNil(t, frames[1].Meta)
	require.EqualValues(t, data.VisTypeTable, frames[1].Meta.PreferredVisualization)
	require.Empty(t, frames[1].Meta.Type)
	require.Len(t, frames[1].Fields, 2)
	require.Equal(t, "method", frames[1].Fields[0].Name)
	require.Equal(t, "GET", *frames[1].Fields[0].At(0).(*string))
	require.Equal(t, "POST", *frames[1].Fields[0].At(1).(*string))
}

func TestAPLResponseFrameBuilderBuildFramesReturnsGraphAndTableForWideTimeSeries(t *testing.T) {
	v1 := float64(100)
	v2 := float64(200)
	result := axiomapi.APLQueryResponse{
		Tables: []query.Table{
			{
				Fields: []query.Field{
					{Name: "_time", Type: "datetime"},
					{Name: "count_", Type: "integer"},
				},
				Columns: []query.Column{
					{"2026-06-11T13:45:00Z", "2026-06-11T13:50:00Z"},
					{v1, v2},
				},
			},
			{
				Fields: []query.Field{
					{Name: "count_", Type: "integer"},
				},
				Columns: []query.Column{
					{v1 + v2},
				},
			},
		},
	}

	frames, err := newAPLResponseFrameBuilder(false, true).BuildFrames(context.Background(), result, aplFrameOptions{})
	require.NoError(t, err)
	require.Len(t, frames, 2)

	require.NotNil(t, frames[0].Meta)
	require.Equal(t, data.FrameTypeTimeSeriesWide, frames[0].Meta.Type)
	require.Equal(t, data.FrameTypeVersion{0, 1}, frames[0].Meta.TypeVersion)
	require.EqualValues(t, data.VisTypeGraph, frames[0].Meta.PreferredVisualization)
	require.Len(t, frames[0].Fields, 2)

	require.NotNil(t, frames[1].Meta)
	require.EqualValues(t, data.VisTypeTable, frames[1].Meta.PreferredVisualization)
	require.Empty(t, frames[1].Meta.Type)
	require.Len(t, frames[1].Fields, 1)
	require.Equal(t, "count_", frames[1].Fields[0].Name)
}

func TestAPLResponseFrameBuilderUsesTimeBeforeSysTimeForTimeSeries(t *testing.T) {
	v1 := float64(100)
	result := axiomapi.APLQueryResponse{
		Tables: []query.Table{
			{
				Fields: []query.Field{
					{Name: "_sysTime", Type: "datetime"},
					{Name: "_time", Type: "datetime"},
					{Name: "Lambda Name", Type: "string"},
					{Name: "Duration", Type: "float"},
				},
				Columns: []query.Column{
					{"2026-06-11T13:40:00Z"},
					{"2026-06-11T13:45:00Z"},
					{"a"},
					{v1},
				},
			},
			{
				Fields: []query.Field{
					{Name: "Lambda Name", Type: "string"},
					{Name: "Duration", Type: "float"},
				},
				Columns: []query.Column{
					{"a"},
					{v1},
				},
			},
		},
	}

	got, err := newAPLResponseFrameBuilder(false).Build(context.Background(), result, aplFrameOptions{})
	require.NoError(t, err)
	require.NotNil(t, got.Meta)
	require.EqualValues(t, data.VisTypeGraph, got.Meta.PreferredVisualization)
	require.Equal(t, "_time", got.Fields[0].Name)
	timestamp, ok := got.Fields[0].At(0).(time.Time)
	require.True(t, ok)
	require.Equal(t, time.Date(2026, 6, 11, 13, 45, 0, 0, time.UTC), timestamp)
	require.Len(t, got.Fields, 2)
}

func TestAPLResponseFrameBuilderBuildsTraceFrame(t *testing.T) {
	result := axiomapi.APLQueryResponse{
		Tables: []query.Table{
			{
				Fields: []query.Field{
					{Name: "trace_id", Type: "string"},
					{Name: "span_id", Type: "string"},
					{Name: "name", Type: "string"},
					{Name: "service.name", Type: "string"},
					{Name: "_time", Type: "datetime"},
					{Name: "duration", Type: "timespan"},
				},
				Columns: []query.Column{
					{"trace-1"},
					{"span-1"},
					{"GET /"},
					{"api"},
					{"2026-06-11T02:19:39Z"},
					{"666.387µs"},
				},
			},
		},
	}

	got, err := newAPLResponseFrameBuilder(false).Build(context.Background(), result, aplFrameOptions{})
	require.NoError(t, err)
	require.Equal(t, "Trace", got.Name)
	require.NotNil(t, got.Meta)
	require.EqualValues(t, data.VisTypeTrace, got.Meta.PreferredVisualization)
}

func TestAPLResponseFrameBuilderUsesTimeBeforeTimestampForTraces(t *testing.T) {
	result := axiomapi.APLQueryResponse{
		Tables: []query.Table{
			{
				Fields: []query.Field{
					{Name: "trace_id", Type: "string"},
					{Name: "span_id", Type: "string"},
					{Name: "name", Type: "string"},
					{Name: "service.name", Type: "string"},
					{Name: "timestamp", Type: "datetime"},
					{Name: "_sysTime", Type: "datetime"},
					{Name: "_time", Type: "datetime"},
					{Name: "duration", Type: "timespan"},
				},
				Columns: []query.Column{
					{"trace-1"},
					{"span-1"},
					{"GET /"},
					{"api"},
					{"2026-06-11T02:18:39Z"},
					{"2026-06-11T02:19:39Z"},
					{"2026-06-11T02:20:39Z"},
					{"666.387µs"},
				},
			},
		},
	}

	got, err := newAPLResponseFrameBuilder(false).Build(context.Background(), result, aplFrameOptions{})
	require.NoError(t, err)
	require.Equal(t, "Trace", got.Name)
	startTime, ok := got.Fields[6].At(0).(*float64)
	require.True(t, ok)
	require.InDelta(t, float64(1781144439000), *startTime, 0.001)
}

func TestAPLResponseFrameBuilderBuildsLogsFrame(t *testing.T) {
	result := axiomapi.APLQueryResponse{
		Tables: []query.Table{
			{
				Fields: []query.Field{
					{Name: "_time", Type: "datetime"},
					{Name: "message", Type: "string"},
					{Name: "level", Type: "string"},
					{Name: "_id", Type: "string"},
					{Name: "service.name", Type: "string"},
				},
				Columns: []query.Column{
					{"2026-06-11T02:19:39Z"},
					{"hello"},
					{"info"},
					{"abc-123"},
					{"api"},
				},
			},
		},
	}

	got, err := newAPLResponseFrameBuilder(false).Build(context.Background(), result, aplFrameOptions{})
	require.NoError(t, err)
	require.NotNil(t, got.Meta)
	require.Equal(t, data.FrameTypeLogLines, got.Meta.Type)
	require.Equal(t, data.FrameTypeVersion{0, 0}, got.Meta.TypeVersion)
	require.EqualValues(t, data.VisTypeLogs, got.Meta.PreferredVisualization)
	require.Len(t, got.Fields, 5)
	require.Equal(t, "timestamp", got.Fields[0].Name)
	require.Equal(t, "body", got.Fields[1].Name)
	require.Equal(t, "severity", got.Fields[2].Name)
	require.Equal(t, "id", got.Fields[3].Name)
	require.Equal(t, "labels", got.Fields[4].Name)
	require.Equal(t, data.FieldTypeTime, got.Fields[0].Type())
	require.Equal(t, data.FieldTypeString, got.Fields[1].Type())
	require.Equal(t, data.FieldTypeString, got.Fields[2].Type())
	require.Equal(t, data.FieldTypeString, got.Fields[3].Type())

	require.Equal(t, "hello", got.Fields[1].At(0))
	require.Equal(t, "info", got.Fields[2].At(0))
	require.Equal(t, "abc-123", got.Fields[3].At(0))

	labels, ok := got.Fields[4].At(0).(json.RawMessage)
	require.True(t, ok)
	require.JSONEq(t, `{"service.name":"api"}`, string(labels))
}

func TestAPLResponseFrameBuilderUsesTimeBeforeSysTimeForLogs(t *testing.T) {
	result := axiomapi.APLQueryResponse{
		Tables: []query.Table{
			{
				Fields: []query.Field{
					{Name: "_sysTime", Type: "datetime"},
					{Name: "_time", Type: "datetime"},
					{Name: "message", Type: "string"},
				},
				Columns: []query.Column{
					{"2026-06-11T02:19:39Z"},
					{"2026-06-11T02:20:39Z"},
					{"hello"},
				},
			},
		},
	}

	got, err := newAPLResponseFrameBuilder(false).Build(context.Background(), result, aplFrameOptions{})
	require.NoError(t, err)
	require.NotNil(t, got.Meta)
	require.Equal(t, data.FrameTypeLogLines, got.Meta.Type)

	timestamp, ok := got.Fields[0].At(0).(time.Time)
	require.True(t, ok)
	require.Equal(t, time.Date(2026, 6, 11, 2, 20, 39, 0, time.UTC), timestamp)
}

func TestAPLResponseFrameBuilderUsesTimeBeforeTimestampForLogs(t *testing.T) {
	result := axiomapi.APLQueryResponse{
		Tables: []query.Table{
			{
				Fields: []query.Field{
					{Name: "timestamp", Type: "datetime"},
					{Name: "_time", Type: "datetime"},
					{Name: "message", Type: "string"},
				},
				Columns: []query.Column{
					{"2026-06-11T02:19:39Z"},
					{"2026-06-11T02:20:39Z"},
					{"hello"},
				},
			},
		},
	}

	got, err := newAPLResponseFrameBuilder(false).Build(context.Background(), result, aplFrameOptions{})
	require.NoError(t, err)
	require.NotNil(t, got.Meta)
	require.Equal(t, data.FrameTypeLogLines, got.Meta.Type)

	timestamp, ok := got.Fields[0].At(0).(time.Time)
	require.True(t, ok)
	require.Equal(t, time.Date(2026, 6, 11, 2, 20, 39, 0, time.UTC), timestamp)
}

func TestAPLResponseFrameBuilderUsesSysTimeFallbackForLogs(t *testing.T) {
	result := axiomapi.APLQueryResponse{
		Tables: []query.Table{
			{
				Fields: []query.Field{
					{Name: "_sysTime", Type: "datetime"},
					{Name: "message", Type: "string"},
				},
				Columns: []query.Column{
					{"2026-06-11T02:19:39Z"},
					{"hello"},
				},
			},
		},
	}

	got, err := newAPLResponseFrameBuilder(false).Build(context.Background(), result, aplFrameOptions{})
	require.NoError(t, err)
	require.NotNil(t, got.Meta)
	require.Equal(t, data.FrameTypeLogLines, got.Meta.Type)

	timestamp, ok := got.Fields[0].At(0).(time.Time)
	require.True(t, ok)
	require.Equal(t, time.Date(2026, 6, 11, 2, 19, 39, 0, time.UTC), timestamp)
}

func TestAPLResponseFrameBuilderFallsBackToSysTimeWhenTimeIsNullForLogs(t *testing.T) {
	result := axiomapi.APLQueryResponse{
		Tables: []query.Table{
			{
				Fields: []query.Field{
					{Name: "_time", Type: "datetime"},
					{Name: "_sysTime", Type: "datetime"},
					{Name: "_raw", Type: "string"},
				},
				Columns: []query.Column{
					{nil},
					{"2026-06-11T02:19:39Z"},
					{"raw log line"},
				},
			},
		},
	}

	got, err := newAPLResponseFrameBuilder(false).Build(context.Background(), result, aplFrameOptions{})
	require.NoError(t, err)
	require.NotNil(t, got.Meta)
	require.Equal(t, data.FrameTypeLogLines, got.Meta.Type)

	timestamp, ok := got.Fields[0].At(0).(time.Time)
	require.True(t, ok)
	require.Equal(t, time.Date(2026, 6, 11, 2, 19, 39, 0, time.UTC), timestamp)
	require.Equal(t, "raw log line", got.Fields[1].At(0))
}

func TestAPLResponseFrameBuilderBuildsGenericEventsFrame(t *testing.T) {
	result := axiomapi.APLQueryResponse{
		Tables: []query.Table{
			{
				Fields: []query.Field{
					{Name: "_time", Type: "datetime"},
					{Name: "count_", Type: "integer"},
				},
				Columns: []query.Column{
					{"2026-06-11T02:19:39Z"},
					{float64(10)},
				},
			},
		},
	}

	got, err := newAPLResponseFrameBuilder(false).Build(context.Background(), result, aplFrameOptions{})
	require.NoError(t, err)
	require.NotNil(t, got.Meta)
	require.EqualValues(t, data.VisTypeTable, got.Meta.PreferredVisualization)
	require.Empty(t, got.Meta.Type)
	require.Len(t, got.Fields, 2)
}

func TestBuildFrameStringifiesUnknownArrayFields(t *testing.T) {
	table := query.Table{
		Fields: []query.Field{
			{Name: "events", Type: "unknown"},
		},
		Columns: []query.Column{
			{
				[]any{
					map[string]any{
						"attributes": map[string]any{"duration": "666.387µs"},
						"name":       "sentry middleware done",
						"timestamp":  float64(1781109117999701800),
					},
					map[string]any{
						"attributes": map[string]any{"duration": "726.738µs"},
						"name":       "tracing middleware done",
						"timestamp":  float64(1781109117999705000),
					},
				},
			},
		},
	}

	got, err := buildAPLFrame(context.Background(), &table)
	require.NoError(t, err)
	require.Len(t, got.Fields, 1)

	value, ok := got.Fields[0].At(0).(*string)
	require.True(t, ok)
	require.JSONEq(t, `[{"attributes":{"duration":"666.387µs"},"name":"sentry middleware done","timestamp":1781109117999701800},{"attributes":{"duration":"726.738µs"},"name":"tracing middleware done","timestamp":1781109117999705000}]`, *value)
}

func TestBuildFrameStringifiesNonStringValuesInStringFields(t *testing.T) {
	table := query.Table{
		Fields: []query.Field{
			{Name: "attributes.custom", Type: "unknown"},
		},
		Columns: []query.Column{
			{
				"plain",
				nil,
				map[string]any{
					"normalizedDatasetName": "axiomersft83axisynthendpointslokiurpnppqp",
				},
			},
		},
	}

	got, err := buildAPLFrame(context.Background(), &table)
	require.NoError(t, err)
	require.Len(t, got.Fields, 1)

	firstValue, ok := got.Fields[0].At(0).(*string)
	require.True(t, ok)
	require.Equal(t, "plain", *firstValue)
	require.True(t, got.Fields[0].NilAt(1))

	thirdValue, ok := got.Fields[0].At(2).(*string)
	require.True(t, ok)
	require.JSONEq(t, `{"normalizedDatasetName":"axiomersft83axisynthendpointslokiurpnppqp"}`, *thirdValue)
}

func TestBuildFrameInfersUnknownTimeField(t *testing.T) {
	table := query.Table{
		Fields: []query.Field{
			{Name: "_time", Type: "unknown"},
			{Name: "count_", Type: "integer"},
		},
		Columns: []query.Column{
			{
				"2026-06-10T22:38:00Z",
				"2026-06-10T22:39:00.123456789Z",
			},
			{
				float64(1),
				float64(2),
			},
		},
	}

	got, err := buildAPLFrame(context.Background(), &table)
	require.NoError(t, err)
	require.Len(t, got.Fields, 2)
	require.True(t, got.Fields[0].Type().Time())
}

func TestBuildFrameTreatsTimeFieldAsDatetimeRegardlessOfDeclaredType(t *testing.T) {
	table := query.Table{
		Fields: []query.Field{
			{Name: "_time", Type: "string"},
			{Name: "path", Type: "string"},
		},
		Columns: []query.Column{
			{
				"2026-06-10T22:38:00Z",
			},
			{
				"hello",
			},
		},
	}

	got, err := buildAPLFrame(context.Background(), &table)
	require.NoError(t, err)
	require.Len(t, got.Fields, 2)
	require.True(t, got.Fields[0].Type().Time())
}

func TestBuildFrameAllowsNullDatetimeValues(t *testing.T) {
	table := query.Table{
		Fields: []query.Field{
			{Name: "_time", Type: "datetime"},
			{Name: "path", Type: "string"},
		},
		Columns: []query.Column{
			{
				nil,
				"2026-06-10T22:38:00Z",
			},
			{
				"first",
				"second",
			},
		},
	}

	got, err := buildAPLFrame(context.Background(), &table)
	require.NoError(t, err)
	require.Len(t, got.Fields, 2)
	require.Equal(t, data.FieldTypeNullableTime, got.Fields[0].Type())
	require.True(t, got.Fields[0].NilAt(0))
	require.False(t, got.Fields[0].NilAt(1))
}

func TestBuildFrameAppliesFieldMetaMapUnitToAggregatedField(t *testing.T) {
	table := query.Table{
		Fields: []query.Field{
			{Name: "_time", Type: "datetime"},
			{Name: "Lambda Name", Type: "string"},
			{
				Name: "Duration",
				Type: "float",
				Aggregation: &query.Aggregation{
					Op:     query.OpMax,
					Fields: []string{"record.metrics.durationMs"},
				},
			},
		},
		Columns: []query.Column{
			{"2026-06-10T22:38:00Z"},
			{"lambda-a"},
			{float64(5234)},
		},
	}
	fieldMetaByName := fieldMetaByNameForResponse(axiomapi.APLQueryResponse{
		DatasetNames: []string{"aws-lambda-dev"},
		FieldsMetaMap: map[string][]axiomapi.APLFieldMetaMap{
			"aws-lambda-dev": {
				{
					Name:        "record.metrics.durationMs",
					Type:        "float",
					Unit:        "ms",
					Description: "Lambda duration",
				},
			},
		},
	})

	got, err := buildAPLFrame(context.Background(), &table, aplFrameOptions{FieldMetaByName: fieldMetaByName})
	require.NoError(t, err)
	require.Len(t, got.Fields, 3)
	require.NotNil(t, got.Fields[2].Config)
	require.Equal(t, "ms", got.Fields[2].Config.Unit)
	require.Equal(t, "Lambda duration", got.Fields[2].Config.Description)
}

func TestBuildFrameAddsAxiomStatusToFrameMetadata(t *testing.T) {
	table := query.Table{
		Fields: []query.Field{
			{Name: "_time", Type: "datetime"},
			{Name: "count_", Type: "integer"},
		},
		Columns: []query.Column{
			{"2026-06-10T22:38:00Z"},
			{float64(10)},
		},
	}
	status := &axiomapi.APLQueryStatus{
		ElapsedTime:    467358,
		BlocksExamined: 10,
		RowsExamined:   155986,
		RowsMatched:    3871,
		IsPartial:      true,
		CacheStatus:    1,
		Messages: []query.Message{
			{Priority: "warn", Count: 1, Code: "apl_warning", Msg: "something happened"},
		},
	}

	got, err := buildAPLFrame(context.Background(), &table, aplFrameOptions{
		Status:  status,
		Query:   "['aws-lambda-dev'] | count",
		TraceID: "trace-123",
	})
	require.NoError(t, err)
	require.NotNil(t, got.Meta)
	require.Equal(t, "['aws-lambda-dev'] | count", got.Meta.ExecutedQueryString)
	require.Len(t, got.Meta.Stats, 9)
	require.Equal(t, "Elapsed time", got.Meta.Stats[0].DisplayName)
	require.Equal(t, "µs", got.Meta.Stats[0].Unit)
	require.Equal(t, float64(467358), got.Meta.Stats[0].Value)
	require.Len(t, got.Meta.Notices, 3)
	require.Equal(t, data.NoticeSeverityWarning, got.Meta.Notices[0].Severity)
	require.Equal(t, "Axiom returned a partial response", got.Meta.Notices[0].Text)
	require.Equal(t, data.NoticeSeverityWarning, got.Meta.Notices[1].Severity)
	require.Equal(t, "apl_warning: something happened", got.Meta.Notices[1].Text)
	require.Equal(t, data.NoticeSeverityInfo, got.Meta.Notices[2].Severity)
	require.Equal(t, "Axiom trace ID: trace-123", got.Meta.Notices[2].Text)
	require.Equal(t, data.InspectTypeStats, got.Meta.Notices[2].Inspect)

	custom, ok := got.Meta.Custom.(map[string]any)
	require.True(t, ok)
	require.Same(t, status, custom["axiomStatus"])
	require.Equal(t, "trace-123", custom["axiomTraceId"])
}

func TestFieldsMatchTrace(t *testing.T) {
	tests := []struct {
		name   string
		fields []query.Field
		want   bool
	}{
		{
			name: "grafana trace fields",
			fields: []query.Field{
				{Name: "traceID"},
				{Name: "spanID"},
				{Name: "operationName"},
				{Name: "serviceName"},
				{Name: "startTime"},
				{Name: "duration"},
			},
			want: true,
		},
		{
			name: "apl otel aliases",
			fields: []query.Field{
				{Name: "trace_id"},
				{Name: "span_id"},
				{Name: "name"},
				{Name: "service.name"},
				{Name: "_time"},
				{Name: "duration"},
			},
			want: true,
		},
		{
			name: "missing duration",
			fields: []query.Field{
				{Name: "trace_id"},
				{Name: "span_id"},
				{Name: "name"},
				{Name: "service.name"},
				{Name: "_time"},
			},
			want: false,
		},
		{
			name: "parent and service tags are not required",
			fields: []query.Field{
				{Name: "trace_id"},
				{Name: "span_id"},
				{Name: "name"},
				{Name: "service.name"},
				{Name: "_time"},
				{Name: "duration"},
				{Name: "parent_span_id"},
			},
			want: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, fieldsMatchTrace(context.Background(), test.fields))
		})
	}
}

func TestBuildFrameSetsTraceMetadata(t *testing.T) {
	table := query.Table{
		Fields: []query.Field{
			{Name: "trace_id", Type: "string"},
			{Name: "span_id", Type: "string"},
			{Name: "name", Type: "string"},
			{Name: "service.name", Type: "string"},
			{Name: "_time", Type: "datetime"},
			{Name: "duration", Type: "timespan"},
			{Name: "attributes.custom", Type: "unknown"},
			{Name: "http.method", Type: "string"},
		},
		Columns: []query.Column{
			{"trace-1"},
			{"span-1"},
			{"GET /"},
			{"api"},
			{"2026-06-11T02:19:39Z"},
			{"666.387µs"},
			{map[string]any{"normalizedDatasetName": "axiom-dataset"}},
			{"POST"},
		},
	}

	got, err := buildAPLFrame(context.Background(), &table)
	require.NoError(t, err)
	require.Equal(t, "Trace", got.Name)
	require.NotNil(t, got.Meta)
	require.EqualValues(t, data.VisTypeTrace, got.Meta.PreferredVisualization)
	require.Len(t, got.Fields, 10)
	require.Equal(t, "startTime", got.Fields[6].Name)
	require.Equal(t, data.FieldTypeNullableFloat64, got.Fields[6].Type())
	startTime, ok := got.Fields[6].At(0).(*float64)
	require.True(t, ok)
	require.InDelta(t, float64(1781144379000), *startTime, 0.001)

	require.Equal(t, "duration", got.Fields[7].Name)
	require.Equal(t, data.FieldTypeNullableFloat64, got.Fields[7].Type())
	duration, ok := got.Fields[7].At(0).(*float64)
	require.True(t, ok)
	require.InDelta(t, 0.666387, *duration, 0.000001)

	serviceTags, ok := got.Fields[5].At(0).(*json.RawMessage)
	require.True(t, ok)
	require.JSONEq(t, `[{"key":"service.name","value":"api"}]`, string(*serviceTags))

	tags, ok := got.Fields[9].At(0).(*json.RawMessage)
	require.True(t, ok)
	require.JSONEq(t, `[{"key":"attributes.custom.normalizedDatasetName","value":"axiom-dataset"},{"key":"http.method","value":"POST"}]`, string(*tags))
}

func TestBuildFrameNormalizesTraceLogs(t *testing.T) {
	table := query.Table{
		Fields: []query.Field{
			{Name: "trace_id", Type: "string"},
			{Name: "span_id", Type: "string"},
			{Name: "name", Type: "string"},
			{Name: "service.name", Type: "string"},
			{Name: "_time", Type: "datetime"},
			{Name: "duration", Type: "timespan"},
			{Name: "events", Type: "unknown"},
		},
		Columns: []query.Column{
			{"trace-1"},
			{"span-1"},
			{"GET /"},
			{"api"},
			{"2026-06-11T02:19:39Z"},
			{"666.387µs"},
			{
				[]any{
					map[string]any{
						"timestamp": "2026-06-11T02:19:40Z",
						"_time":     "2026-06-11T02:20:40Z",
						"name":      "exception",
						"attributes": map[string]any{
							"exception.message": "boom",
						},
					},
				},
			},
		},
	}

	got, err := buildAPLFrame(context.Background(), &table)
	require.NoError(t, err)

	logs, ok := got.Fields[8].At(0).(*json.RawMessage)
	require.True(t, ok)
	require.JSONEq(t, `[{"fields":[{"key":"exception.message","value":"boom"}],"name":"exception","timestamp":1781144440000}]`, string(*logs))
}

func TestAPLWideFrameBuilderFillsMissingWithPreviousValue(t *testing.T) {
	t1 := time.Date(2026, 6, 11, 13, 45, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 11, 13, 50, 0, 0, time.UTC)
	t3 := time.Date(2026, 6, 11, 13, 55, 0, 0, time.UTC)
	v1 := float64(100)
	v2 := float64(200)
	v3 := float64(110)

	longFrame := data.NewFrame(
		"response",
		data.NewField("_time", nil, []time.Time{t1, t2, t3}),
		data.NewField("Lambda Name", nil, []*string{stringPtr("a"), stringPtr("b"), stringPtr("a")}),
		data.NewField("Duration", nil, []*float64{&v1, &v2, &v3}),
	)
	longFrame.Fields[2].Config = &data.FieldConfig{Unit: "ms"}

	wideFrame, err := aplWideFrameBuilder{}.Build(longFrame)
	require.NoError(t, err)
	require.Len(t, wideFrame.Fields, 3)

	var seriesA *data.Field
	for _, field := range wideFrame.Fields {
		if field.Labels["Lambda Name"] == "a" {
			seriesA = field
			break
		}
	}
	require.NotNil(t, seriesA)
	require.NotNil(t, seriesA.Config)
	require.Equal(t, "a", seriesA.Config.DisplayNameFromDS)
	require.Equal(t, "ms", seriesA.Config.Unit)

	filledValue, ok := seriesA.At(1).(*float64)
	require.True(t, ok)
	require.Equal(t, float64(100), *filledValue)
}

func TestMetricsFrameBuilderUsesTagValuesForSeriesName(t *testing.T) {
	v1 := float64(0.1)
	v2 := float64(0.2)

	frame := newMetricsFrameBuilder(axiomapi.MetricsQueryMetadata{Unit: "{cpu}"}, "A").Build(
		axiomapi.MetricsQuerySeries{
			Resolution: 60,
			Start:      1781186400,
			Metric:     "k8s.pod.cpu.usage",
			Tags: map[string]string{
				"container.image.name": "602401143452.dkr.ecr.eu-west-1.amazonaws.com/eks/kube-proxy",
				"container.name":       "api",
				"pod.name":             "api-7d9",
			},
			Data: []*float64{&v1, &v2},
		},
	)

	require.Equal(t, "A", frame.RefID)
	require.Len(t, frame.Fields, 2)
	require.NotNil(t, frame.Meta)
	require.Equal(t, data.FrameTypeTimeSeriesMulti, frame.Meta.Type)
	require.Equal(t, data.FrameTypeVersion{0, 1}, frame.Meta.TypeVersion)
	require.Equal(t, "602401143452.dkr.ecr.eu-west-1.amazonaws.com/eks/kube-proxy | api | api-7d9", frame.Fields[1].Name)
	require.NotNil(t, frame.Fields[0].Config)
	require.Equal(t, float64(60000), frame.Fields[0].Config.Interval)
	require.NotNil(t, frame.Fields[1].Config)
	require.Equal(t, "", frame.Fields[1].Config.DisplayNameFromDS)
	require.Equal(t, "api", frame.Fields[1].Labels["container.name"])
	require.Equal(t, "{cpu}", frame.Fields[1].Config.Unit)
}

func TestMetricsFrameBuilderUsesExplicitLabelTagForSeriesName(t *testing.T) {
	v1 := float64(0.1)

	frame := newMetricsFrameBuilder(axiomapi.MetricsQueryMetadata{Unit: "short"}, "A").Build(
		axiomapi.MetricsQuerySeries{
			Resolution: 60,
			Start:      1781186400,
			Metric:     "k8s.pod.cpu.usage",
			Tags: map[string]string{
				"__label":  "api-7d9",
				"pod.name": "api-7d9",
			},
			Data: []*float64{&v1},
		},
	)

	require.Len(t, frame.Fields, 2)
	require.Equal(t, "api-7d9", frame.Fields[1].Name)
	require.NotNil(t, frame.Fields[1].Config)
	require.Equal(t, "api-7d9", frame.Fields[1].Config.DisplayNameFromDS)
	require.Equal(t, "api-7d9", frame.Fields[1].Labels["__label"])
	require.Equal(t, "api-7d9", frame.Fields[1].Labels["pod.name"])
}

func TestMetricsFrameBuilderBuildsLabeledTableFrame(t *testing.T) {
	cpuAPI := float64(0.2)
	cpuAPIReplica := float64(0.4)
	memAPI := float64(512)
	cpuWorker := float64(0.8)

	frame := newMetricsFrameBuilder(axiomapi.MetricsQueryMetadata{Unit: "short"}, "A").BuildTable(
		[]axiomapi.MetricsQuerySeries{
			{
				Metric: "cpu",
				Tags:   map[string]string{"__label": "api", "container.name": "app", "pod.name": "api-7d9"},
				Data:   []*float64{nil, &cpuAPI},
			},
			{
				Metric: "memory",
				Tags:   map[string]string{"__label": "api", "container.name": "app", "pod.name": "api-7d9"},
				Data:   []*float64{&memAPI},
			},
			{
				Metric: "cpu",
				Tags:   map[string]string{"__label": "api", "container.name": "sidecar", "pod.name": "api-8bc"},
				Data:   []*float64{&cpuAPIReplica},
			},
			{
				Metric: "cpu",
				Tags:   map[string]string{"__label": "worker", "container.name": "app", "pod.name": "worker-5f8"},
				Data:   []*float64{&cpuWorker},
			},
			{
				Metric: "memory",
				Tags:   map[string]string{"__label": "worker", "container.name": "app", "pod.name": "worker-5f8"},
				Data:   []*float64{nil},
			},
		},
	)

	require.Equal(t, "A", frame.RefID)
	require.NotNil(t, frame.Meta)
	require.EqualValues(t, data.VisTypeTable, frame.Meta.PreferredVisualization)
	require.Len(t, frame.Fields, 5)
	require.Equal(t, "__label", frame.Fields[0].Name)
	require.Equal(t, "container.name", frame.Fields[1].Name)
	require.Equal(t, "pod.name", frame.Fields[2].Name)
	require.Equal(t, "cpu", frame.Fields[3].Name)
	require.Equal(t, "memory", frame.Fields[4].Name)
	require.Equal(t, "api", *frame.Fields[0].At(0).(*string))
	require.Equal(t, "api", *frame.Fields[0].At(1).(*string))
	require.Equal(t, "worker", *frame.Fields[0].At(2).(*string))
	require.Equal(t, "app", *frame.Fields[1].At(0).(*string))
	require.Equal(t, "sidecar", *frame.Fields[1].At(1).(*string))
	require.Equal(t, "app", *frame.Fields[1].At(2).(*string))
	require.Equal(t, "api-7d9", *frame.Fields[2].At(0).(*string))
	require.Equal(t, "api-8bc", *frame.Fields[2].At(1).(*string))
	require.Equal(t, "worker-5f8", *frame.Fields[2].At(2).(*string))
	require.Equal(t, cpuAPI, *frame.Fields[3].At(0).(*float64))
	require.Equal(t, cpuAPIReplica, *frame.Fields[3].At(1).(*float64))
	require.Equal(t, cpuWorker, *frame.Fields[3].At(2).(*float64))
	require.Equal(t, memAPI, *frame.Fields[4].At(0).(*float64))
	require.Nil(t, frame.Fields[4].At(1))
	require.Nil(t, frame.Fields[4].At(2))
	require.NotNil(t, frame.Fields[3].Config)
	require.Equal(t, "short", frame.Fields[3].Config.Unit)
}

func TestMetricsFrameBuilderSetsDisplayNameFromRefIDWhenTagsAreEmpty(t *testing.T) {
	v1 := float64(0.1)

	frame := newMetricsFrameBuilder(axiomapi.MetricsQueryMetadata{}, "A").Build(
		axiomapi.MetricsQuerySeries{
			Resolution: 60,
			Start:      1781186400,
			Metric:     "k8s.pod.cpu.usage",
			Data:       []*float64{&v1},
		},
	)

	require.Len(t, frame.Fields, 2)
	require.Equal(t, "k8s.pod.cpu.usage", frame.Fields[1].Name)
	require.NotNil(t, frame.Fields[1].Config)
	require.Empty(t, frame.Fields[1].Labels)
	require.Equal(t, "A", frame.Fields[1].Config.DisplayNameFromDS)
}
