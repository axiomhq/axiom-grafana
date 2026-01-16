package plugin

import (
	"context"
	"encoding/json"
	"testing"

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

func TestBuildQueryEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		apiHost  string
		region   string
		expected string
	}{
		{
			name:     "no region - uses apiHost with path appended",
			apiHost:  "https://api.axiom.co",
			region:   "",
			expected: "https://api.axiom.co/v1/datasets/_apl",
		},
		{
			name:     "region set - uses edge endpoint",
			apiHost:  "https://api.axiom.co",
			region:   "eu-central-1.aws.edge.axiom.co",
			expected: "https://eu-central-1.aws.edge.axiom.co/v1/datasets/_apl?format=tabular",
		},
		{
			name:     "region with https scheme",
			apiHost:  "https://api.axiom.co",
			region:   "https://eu-central-1.aws.edge.axiom.co",
			expected: "https://eu-central-1.aws.edge.axiom.co/v1/datasets/_apl?format=tabular",
		},
		{
			name:     "region with trailing slash",
			apiHost:  "https://api.axiom.co",
			region:   "us-east-1.aws.edge.axiom.co/",
			expected: "https://us-east-1.aws.edge.axiom.co/v1/datasets/_apl?format=tabular",
		},
		{
			name:     "legacy EU instance - no region",
			apiHost:  "https://api.eu.axiom.co",
			region:   "",
			expected: "https://api.eu.axiom.co/v1/datasets/_apl",
		},
		{
			name:     "staging edge region",
			apiHost:  "https://api.axiom.co",
			region:   "us-east-1.edge.staging.axiomdomain.co",
			expected: "https://us-east-1.edge.staging.axiomdomain.co/v1/datasets/_apl?format=tabular",
		},
		{
			name:     "apiHost with custom path - used as-is, takes precedence over region",
			apiHost:  "http://localhost:3400/v1/datasets/_apl",
			region:   "eu-central-1.aws.edge.axiom.co",
			expected: "http://localhost:3400/v1/datasets/_apl",
		},
		{
			name:     "apiHost with custom path - no region",
			apiHost:  "http://localhost:8080/custom/query/path",
			region:   "",
			expected: "http://localhost:8080/custom/query/path",
		},
		{
			name:     "apiHost with trailing slash - appends query path",
			apiHost:  "https://api.axiom.co/",
			region:   "",
			expected: "https://api.axiom.co/v1/datasets/_apl",
		},
		{
			name:     "no apiHost, no region - uses default cloud endpoint",
			apiHost:  "",
			region:   "",
			expected: "https://api.axiom.co/v1/datasets/_apl",
		},
		{
			name:     "no apiHost, region set - uses region",
			apiHost:  "",
			region:   "eu-central-1.aws.edge.axiom.co",
			expected: "https://eu-central-1.aws.edge.axiom.co/v1/datasets/_apl?format=tabular",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ds := Datasource{
				apiHost: test.apiHost,
				region:  test.region,
			}

			endpoint, err := ds.buildQueryEndpoint()
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

			got := buildFrame(context.Background(), &queryRes.Tables[0])
			t.Logf("%#v", got)
		})
	}
}
