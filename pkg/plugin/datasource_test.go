package plugin

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/axiomhq/axiom-go/axiom/query"

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

func TestBuildFrame(t *testing.T) {
	tests := []struct {
		name        string
		aplResponse string
		assertions  func(t *testing.T, frame *data.Frame)
	}{
		{
			name:        "regular_query",
			aplResponse: `{"format":"tabular","status":{"elapsedTime":760311,"blocksExamined":2672,"rowsExamined":167932295,"rowsMatched":167932295,"numGroups":0,"isPartial":false,"cacheStatus":1,"minBlockTime":"2021-11-29T18:55:53.248Z","maxBlockTime":"2024-08-15T10:09:06.396Z","messages":[{"priority":"warn","count":1,"code":"apl_implicitendtimeofnowapplied_1","msg":"line: 1, col: 20: implicit end time of 'now' applied"}]},"tables":[{"name":"0","sources":[{"name":"vercel"}],"fields":[{"name":"_time","type":"datetime"},{"name":"request.method","type":"string"},{"name":"count_","type":"integer","agg":{"name":"count"}}],"order":[{"field":"_time","desc":false},{"field":"count_","desc":true}],"groups":[{"name":"_time"},{"name":"request.method"}],"range":{"field":"_time","start":"2013-12-21T00:00:00Z","end":"2024-12-18T00:00:00Z"},"buckets":{"field":"_time","size":31536000000000000},"columns":[["2020-12-19T00:00:00Z","2020-12-19T00:00:00Z","2020-12-19T00:00:00Z","2020-12-19T00:00:00Z","2020-12-19T00:00:00Z","2020-12-19T00:00:00Z","2021-12-19T00:00:00Z","2021-12-19T00:00:00Z","2021-12-19T00:00:00Z","2021-12-19T00:00:00Z","2021-12-19T00:00:00Z","2021-12-19T00:00:00Z","2021-12-19T00:00:00Z","2021-12-19T00:00:00Z","2021-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2022-12-19T00:00:00Z","2023-12-19T00:00:00Z","2023-12-19T00:00:00Z","2023-12-19T00:00:00Z","2023-12-19T00:00:00Z","2023-12-19T00:00:00Z","2023-12-19T00:00:00Z","2023-12-19T00:00:00Z","2023-12-19T00:00:00Z","2023-12-19T00:00:00Z"],["GET",null,"POST","HEAD","PUT","DELETE","GET",null,"HEAD","POST","PUT","DELETE","OPTIONS","PROPFIND","PATCH","GET",null,"POST","HEAD","OPTIONS","PUT","DELETE","CONNECT","PROPFIND","PATCH","GET",null,"POST","HEAD","OPTIONS","PUT","DELETE","PATCH","PROPFIND"],[397262,2753,1608,334,209,70,20591383,1882923,163479,21831,656,280,29,9,1,68114522,7325479,3953165,3296763,8989,1010,419,25,24,24,50657014,8358670,2480118,668371,3377,949,484,26,9]]},{"name":"_totals","sources":[{"name":"vercel"}],"fields":[{"name":"request.method","type":"string"},{"name":"count_","type":"integer","agg":{"name":"count"}}],"order":[{"field":"count_","desc":true}],"groups":[{"name":"request.method"}],"range":{"field":"_time","start":"2013-12-21T00:00:00Z","end":"2024-12-18T00:00:00Z"},"columns":[["GET",null,"POST","HEAD","OPTIONS","PUT","DELETE","PATCH","PROPFIND","CONNECT"],[139760181,17569825,6456722,4128947,12395,2824,1253,51,42,25]]}],"datasetNames":["vercel"],"fieldsMetaMap":{"vercel":[{"name":"report.durationMs","type":"float","unit":"ms","hidden":false,"description":""},{"name":"report.maxMemoryUsedMb","type":"integer","unit":"decmbytes","hidden":false,"description":""},{"name":"webVital.value","type":"integer|float","unit":"ms","hidden":false,"description":""}]}}`,
			assertions: func(t *testing.T, frame *data.Frame) {
				// Should have 3 fields: _time, request.method, count_
				require.Len(t, frame.Fields, 3, "should have 3 fields")

				// Check field names and types
				assert.Equal(t, "_time", frame.Fields[0].Name)
				assert.Equal(t, data.FieldTypeTime, frame.Fields[0].Type())

				assert.Equal(t, "request.method", frame.Fields[1].Name)
				assert.Equal(t, data.FieldTypeNullableString, frame.Fields[1].Type())

				assert.Equal(t, "count_", frame.Fields[2].Name)
				assert.Equal(t, data.FieldTypeNullableFloat64, frame.Fields[2].Type())

				// Check that we have data rows
				assert.Greater(t, frame.Fields[0].Len(), 0, "should have data rows")
			},
		},
		{
			name:        "histogram_query",
			aplResponse: `{"format":"tabular","status":{"elapsedTime":659388,"blocksExamined":32,"blocksCached":0,"blocksMatched":0,"blocksSkipped":0,"rowsExamined":4081672,"rowsMatched":2170280,"numGroups":0,"isPartial":false,"cacheStatus":1,"minBlockTime":"2026-02-19T12:44:44Z","maxBlockTime":"2026-02-19T15:32:46Z"},"tables":[{"name":"0","sources":[{"name":"sample-http-logs"}],"fields":[{"name":"_time","type":"datetime"},{"name":"histogram_req_duration_ms","type":"array","agg":{"name":"histogram","fields":["req_duration_ms"],"args":[5]}}],"order":[{"field":"_time","desc":false}],"groups":[{"name":"_time"}],"range":{"field":"_time","start":"2026-02-19T14:00:00Z","end":"2026-02-19T16:00:00Z"},"buckets":{"field":"_time","size":3600000000000},"columns":[["2026-02-19T14:00:00Z","2026-02-19T15:00:00Z"],[[{"count":978041,"from":0.1000004199205291,"to":1.1217355445622026},{"from":1.1217355445622026,"to":2.143470669203876,"count":336184},{"to":3.1652057938455496,"count":47065,"from":2.143470669203876},{"count":2342,"from":3.1652057938455496,"to":4.186940918487223},{"from":4.186940918487223,"to":5.2086760431288965,"count":44}],[{"from":0.1000004199205291,"to":1.1217355445622026,"count":579366},{"from":1.1217355445622026,"to":2.143470669203876,"count":197615},{"from":2.143470669203876,"to":3.1652057938455496,"count":28173},{"from":3.1652057938455496,"to":4.186940918487223,"count":1426},{"count":24,"from":4.186940918487223,"to":5.2086760431288965}]]]},{"name":"_totals","sources":[{"name":"sample-http-logs"}],"fields":[{"name":"histogram_req_duration_ms","type":"array","agg":{"name":"histogram","fields":["req_duration_ms"],"args":[5]}}],"order":[],"groups":[],"range":{"field":"_time","start":"2026-02-19T14:00:00Z","end":"2026-02-19T16:00:00Z"},"columns":[[[{"from":0.1000004199205291,"to":1.1217355445622026,"count":1557407},{"from":1.1217355445622026,"to":2.143470669203876,"count":533799},{"to":3.1652057938455496,"count":75238,"from":2.143470669203876},{"from":3.1652057938455496,"to":4.186940918487223,"count":3768},{"count":68,"from":4.186940918487223,"to":5.2086760431288965}]]]}],"datasetNames":["sample-http-logs"],"fieldsMetaMap":{"sample-http-logs":[{"name":"req_duration_ms","type":"integer|float","unit":"ms","hidden":false,"description":""},{"name":"resp_body_size_bytes","type":"integer","unit":"decmbytes","hidden":false,"description":""},{"name":"resp_header_size_bytes","type":"integer","unit":"Kbits","hidden":false,"description":""}]}}`,
			assertions: func(t *testing.T, frame *data.Frame) {
				// Should have time field + bucket fields (1 time + 5 buckets = 6 fields)
				require.GreaterOrEqual(t, len(frame.Fields), 6, "should have time field + bucket fields")

				// First field should be time
				timeField := frame.Fields[0]
				assert.Equal(t, "_time", timeField.Name, "first field should be time")
				assert.Equal(t, data.FieldTypeTime, timeField.Type(), "time field should have correct type")

				// Remaining fields should be bucket fields with "le" labels
				bucketCount := 0
				for i, field := range frame.Fields {
					if i == 0 {
						continue // skip time field
					}

					assert.Equal(t, "", field.Name, "bucket fields should have empty name")
					assert.NotNil(t, field.Labels, "bucket fields should have labels")

					le, exists := field.Labels["le"]
					assert.True(t, exists, "bucket field should have 'le' label")
					assert.NotEmpty(t, le, "le label should not be empty")
					bucketCount++
				}

				// Should have 5 bucket boundaries based on the test data
				assert.Equal(t, 5, bucketCount, "should have 5 bucket fields for the test data")

				t.Logf("Created frame with %d fields (%d buckets)", len(frame.Fields), bucketCount)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var queryRes query.Result
			err := json.Unmarshal([]byte(test.aplResponse), &queryRes)
			require.NoError(t, err)

			got := buildFrame(context.Background(), &queryRes.Tables[0])

			// Run test-specific assertions
			test.assertions(t, got)
		})
	}
}

func TestIsHistogramField(t *testing.T) {
	tests := []struct {
		name     string
		field    query.Field
		expected bool
	}{
		{
			name: "histogram_aggregation",
			field: query.Field{
				Name:        "histogram_req_duration_ms",
				Type:        "array",
				Aggregation: &query.Aggregation{Op: query.OpHistogram},
			},
			expected: true,
		},
		{
			name: "array_with_histogram_name",
			field: query.Field{
				Name: "histogram_values",
				Type: "array",
			},
			expected: false, // No longer using name-based guessing
		},
		{
			name: "regular_field",
			field: query.Field{
				Name: "count",
				Type: "integer",
			},
			expected: false,
		},
		{
			name: "array_without_histogram_name",
			field: query.Field{
				Name: "tags",
				Type: "array",
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isHistogramField(test.field)
			assert.Equal(t, test.expected, result)
		})
	}
}
