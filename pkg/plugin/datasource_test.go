package plugin

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/axiomhq/axiom-go/axiom/query"
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

func TestResolveBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		apiHost  string
		edge     string
		edgeURL  string
		expected string
	}{
		{
			name:     "no edge - uses apiHost",
			apiHost:  "https://api.axiom.co",
			expected: "https://api.axiom.co",
		},
		{
			name:     "edge domain",
			apiHost:  "https://api.axiom.co",
			edge:     "eu-central-1.aws.edge.axiom.co",
			expected: "https://eu-central-1.aws.edge.axiom.co",
		},
		{
			name:     "edge domain with trailing slash",
			apiHost:  "https://api.axiom.co",
			edge:     "us-east-1.aws.edge.axiom.co/",
			expected: "https://us-east-1.aws.edge.axiom.co",
		},
		{
			name:     "edgeURL without path",
			apiHost:  "https://api.axiom.co",
			edgeURL:  "https://eu-central-1.aws.edge.axiom.co",
			expected: "https://eu-central-1.aws.edge.axiom.co",
		},
		{
			name:     "edgeURL with trailing slash",
			apiHost:  "https://api.axiom.co",
			edgeURL:  "https://eu-central-1.aws.edge.axiom.co/",
			expected: "https://eu-central-1.aws.edge.axiom.co",
		},
		{
			name:     "edgeURL with custom path - used as-is",
			edgeURL:  "http://localhost:3400/query",
			expected: "http://localhost:3400/query",
		},
		{
			name:     "edgeURL takes precedence over edge domain",
			edgeURL:  "https://primary.edge.axiom.co",
			edge:     "secondary.edge.axiom.co",
			expected: "https://primary.edge.axiom.co",
		},
		{
			name:     "legacy EU instance - no edge",
			apiHost:  "https://api.eu.axiom.co",
			expected: "https://api.eu.axiom.co",
		},
		{
			name:     "staging edge domain",
			edge:     "us-east-1.edge.staging.axiomdomain.co",
			expected: "https://us-east-1.edge.staging.axiomdomain.co",
		},
		{
			name:     "no apiHost, no edge - uses default cloud endpoint",
			expected: "https://api.axiom.co",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			endpoint, err := resolveBaseUrl(urlInput{
				APIHost: test.apiHost,
				Edge:    test.edge,
				EdgeURL: test.edgeURL,
			})
			require.NoError(t, err)
			require.Equal(t, test.expected, endpoint)
		})
	}
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

			got, err := buildFrame(context.Background(), &queryRes.Tables[0])
			require.NoError(t, err)
			t.Logf("%#v", got)
		})
	}
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

	got, err := buildFrame(context.Background(), &table)
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

	got, err := buildFrame(context.Background(), &table)
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

	got, err := buildFrame(context.Background(), &table)
	require.NoError(t, err)
	require.Len(t, got.Fields, 2)
	require.Equal(t, data.FieldTypeTime, got.Fields[0].Type())
}

func TestBuildFrameTreatsTimeFieldAsDatetimeRegardlessOfDeclaredType(t *testing.T) {
	table := query.Table{
		Fields: []query.Field{
			{Name: "_time", Type: "string"},
			{Name: "message", Type: "string"},
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

	got, err := buildFrame(context.Background(), &table)
	require.NoError(t, err)
	require.Len(t, got.Fields, 2)
	require.Equal(t, data.FieldTypeTime, got.Fields[0].Type())
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
	fieldMetaByName := fieldMetaByNameForResponse(APLQueryResponse{
		DatasetNames: []string{"aws-lambda-dev"},
		FieldsMetaMap: map[string][]APLFieldMetaMap{
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

	got, err := buildFrame(context.Background(), &table, aplFrameOptions{FieldMetaByName: fieldMetaByName})
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
	status := &APLQueryStatus{
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

	got, err := buildFrame(context.Background(), &table, aplFrameOptions{
		Status: status,
		Query:  "['aws-lambda-dev'] | count",
	})
	require.NoError(t, err)
	require.NotNil(t, got.Meta)
	require.Equal(t, "['aws-lambda-dev'] | count", got.Meta.ExecutedQueryString)
	require.Len(t, got.Meta.Stats, 9)
	require.Equal(t, "Elapsed time", got.Meta.Stats[0].DisplayName)
	require.Equal(t, "µs", got.Meta.Stats[0].Unit)
	require.Equal(t, float64(467358), got.Meta.Stats[0].Value)
	require.Len(t, got.Meta.Notices, 2)
	require.Equal(t, data.NoticeSeverityWarning, got.Meta.Notices[0].Severity)
	require.Equal(t, "Axiom returned a partial response", got.Meta.Notices[0].Text)
	require.Equal(t, data.NoticeSeverityWarning, got.Meta.Notices[1].Severity)
	require.Equal(t, "apl_warning: something happened", got.Meta.Notices[1].Text)

	custom, ok := got.Meta.Custom.(map[string]any)
	require.True(t, ok)
	require.Same(t, status, custom["axiomStatus"])
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

	got, err := buildFrame(context.Background(), &table)
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

func TestLongToWideFrameFillsMissingWithPreviousValue(t *testing.T) {
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

	wideFrame, err := longToWideFrame(longFrame)
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

func TestBuildMetricsFrameSetsDisplayNameFromLabels(t *testing.T) {
	v1 := float64(0.1)
	v2 := float64(0.2)

	frame := buildMetricsFrame(
		MetricsQuerySeries{
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
		MetricsQueryMetadata{Unit: "{cpu}"},
		"A",
	)

	require.Equal(t, "A", frame.RefID)
	require.Len(t, frame.Fields, 2)
	require.NotNil(t, frame.Meta)
	require.Equal(t, data.FrameTypeTimeSeriesMulti, frame.Meta.Type)
	require.Equal(t, data.FrameTypeVersion{0, 1}, frame.Meta.TypeVersion)
	require.NotNil(t, frame.Fields[0].Config)
	require.Equal(t, float64(60000), frame.Fields[0].Config.Interval)
	require.NotNil(t, frame.Fields[1].Config)
	require.Equal(t, "", frame.Fields[1].Config.DisplayNameFromDS)
	require.Equal(t, "api", frame.Fields[1].Labels["container.name"])
	require.Equal(t, "{cpu}", frame.Fields[1].Config.Unit)
}

func TestBuildMetricsFrameSetsDisplayNameFromRefIDWhenTagsAreEmpty(t *testing.T) {
	v1 := float64(0.1)

	frame := buildMetricsFrame(
		MetricsQuerySeries{
			Resolution: 60,
			Start:      1781186400,
			Metric:     "k8s.pod.cpu.usage",
			Data:       []*float64{&v1},
		},
		MetricsQueryMetadata{},
		"A",
	)

	require.Len(t, frame.Fields, 2)
	require.NotNil(t, frame.Fields[1].Config)
	require.Empty(t, frame.Fields[1].Labels)
	require.Equal(t, "A", frame.Fields[1].Config.DisplayNameFromDS)
}
