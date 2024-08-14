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

func TestBuildFrame(t *testing.T) {
	tests := []struct {
		name        string
		aplResponse string
		expect      *data.Frame
	}{
		{
			name:        "example",
			aplResponse: `{"format":"legacy","status":{"minCursor":"0csdv2htzccgx-065f722a1e0025c0-0000","maxCursor":"0csdz409sdq6l-065f7eb634003f40-d91b","elapsedTime":291650,"blocksExamined":11,"rowsExamined":557382,"rowsMatched":217079,"numGroups":10,"isPartial":false,"cacheStatus":1,"minBlockTime":"2023-05-04T22:53:31.819299105Z","maxBlockTime":"2023-05-05T02:03:34.760941005Z"},"matches":[],"buckets":{"series":[{"startTime":"1970-01-01T00:00:00Z","endTime":"2262-04-11T23:47:16.854775807Z","groups":[{"id":7895528429356786403,"group":{"SDK":null,"Version":null},"aggregations":[{"op":"Requests","value":14524640}]},{"id":10670061725563702516,"group":{"SDK":"axiom-cloudflare","Version":"v0.2.0"},"aggregations":[{"op":"Requests","value":212220}]},{"id":15459439666255162747,"group":{"SDK":"axiom-go","Version":"v0.13.4"},"aggregations":[{"op":"Requests","value":171572}]},{"id":4088840906701912936,"group":{"SDK":"axiom-go","Version":"v0.15.0"},"aggregations":[{"op":"Requests","value":114046}]},{"id":14852570339305235431,"group":{"SDK":"axiom-node","Version":"v0.11.0"},"aggregations":[{"op":"Requests","value":89842}]},{"id":14422770809727427189,"group":{"SDK":"axiom-py","Version":"v0.1.0-beta.2"},"aggregations":[{"op":"Requests","value":38818}]},{"id":3678376667374218836,"group":{"SDK":"axiom-rs","Version":"v0.8.0"},"aggregations":[{"op":"Requests","value":30838}]},{"id":15864127934818674950,"group":{"SDK":"axiom-node","Version":"v0.8.0"},"aggregations":[{"op":"Requests","value":19726}]},{"id":5875348913754133788,"group":{"SDK":"axiom-node","Version":"v0.10.0"},"aggregations":[{"op":"Requests","value":16546}]},{"id":16652330192304560158,"group":{"SDK":"next-axiom","Version":"v0.17.0"},"aggregations":[{"op":"Requests","value":13058}]}]}],"totals":[{"id":7895528429356786403,"group":{"SDK":null,"Version":null},"aggregations":[{"op":"Requests","value":14524640}]},{"id":10670061725563702516,"group":{"SDK":"axiom-cloudflare","Version":"v0.2.0"},"aggregations":[{"op":"Requests","value":212220}]},{"id":15459439666255162747,"group":{"SDK":"axiom-go","Version":"v0.13.4"},"aggregations":[{"op":"Requests","value":171572}]},{"id":4088840906701912936,"group":{"SDK":"axiom-go","Version":"v0.15.0"},"aggregations":[{"op":"Requests","value":114046}]},{"id":14852570339305235431,"group":{"SDK":"axiom-node","Version":"v0.11.0"},"aggregations":[{"op":"Requests","value":89842}]},{"id":14422770809727427189,"group":{"SDK":"axiom-py","Version":"v0.1.0-beta.2"},"aggregations":[{"op":"Requests","value":38818}]},{"id":3678376667374218836,"group":{"SDK":"axiom-rs","Version":"v0.8.0"},"aggregations":[{"op":"Requests","value":30838}]},{"id":15864127934818674950,"group":{"SDK":"axiom-node","Version":"v0.8.0"},"aggregations":[{"op":"Requests","value":19726}]},{"id":5875348913754133788,"group":{"SDK":"axiom-node","Version":"v0.10.0"},"aggregations":[{"op":"Requests","value":16546}]},{"id":16652330192304560158,"group":{"SDK":"next-axiom","Version":"v0.17.0"},"aggregations":[{"op":"Requests","value":13058}]}]},"request":{"startTime":"2023-05-05T01:24:18Z","endTime":"2023-05-05T01:54:18Z","resolution":"","aggregations":[{"op":"sum","field":"count","alias":"Requests"}],"groupBy":["SDK","Version"],"order":[{"field":"Requests","desc":true}],"limit":10,"virtualFields":null,"project":[{"field":"Requests","alias":"Requests"},{"field":"SDK","alias":"SDK"},{"field":"Version","alias":"Version"}],"cursor":"","includeCursor":false},"datasetNames":["axiom-dx-analytics"],"fieldsMetaMap":{"heroku":[],"segement-io-screen":[],"hctest":[],"openai":[],"segement-io-track":[],"test-honeycomb":[],"axiomdb-dataset-metrics":[{"name":"compressedBytes","type":"integer","unit":"decbytes","hidden":false,"description":""},{"name":"bytesIngested","type":"integer","unit":"decbytes","hidden":false,"description":""}],"cloudwatch-v0.3.0b1":[{"name":"lambda.durationMS","type":"float","unit":"ms","hidden":false,"description":""},{"name":"lambda.maxMemoryMB","type":"integer","unit":"decmbytes","hidden":false,"description":""},{"name":"lambda.memorySizeMB","type":"integer","unit":"decmbytes","hidden":false,"description":""},{"name":"lambda","type":"string","unit":"","hidden":false,"description":""}],"tv-on-axiom":[],"caps-test":[],"deno":[],"test-empty":[],"axiom_cloudfront_static":[],"axiomdb-metrics":[{"name":"node.bytesIngested","type":"integer","unit":"bytes","hidden":false,"description":""}],"fly-log-shipper":[],"render":[],"covid-johns-hopkins":[],"testupload":[],"_traces":[],"axiom-cloudfront-lambda":[],"duckbilldemo":[],"logs":[],"metricbeat":[],"oracle-axiom":[],"pingdom":[],"seif-playground":[],"axiom-lambda-dev":[],"cookout-render":[],"github":[],"herokuapp":[],"render-tola":[],"axiom-lambda-extension-go":[],"blabla":[],"cloudflare_http":[{"name":"ColoCode","type":"string","unit":"","hidden":true,"description":""}],"segement-io-page":[],"site-that-use-vercel":[],"axiom-history":[],"axiom-query-metrics":[],"emails-csv":[],"logstash":[],"two-hundo":[],"axiom_segment_webhook":[],"doc-emoji-feedback":[],"test-loki":[],"cloudflare-logs":[],"nginx-logs":[],"segement-io-alias":[],"test-syslog":[],"axiom_cloudfront_backfill":[],"axiomcore-query-metrics":[],"metrics":[],"prisma_test":[],"twilio":[],"empty-dataset":[],"ifttt-weather":[{"name":"humidity","type":"integer","unit":"percent100","hidden":false,"description":""}],"cloudwatch":[{"name":"eks","type":"string","unit":"","hidden":true,"description":""}],"fluentbit":[],"packages.redis.io-cf-logs":[],"segement-io-identify":[],"test-splunk":[],"axiom-audit":[],"axiom-dx-analytics":[],"segement-io-group":[],"axiom-app-webhook":[],"journalbeat":[]}}`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			var queryRes query.Result
			err := json.Unmarshal([]byte(test.aplResponse), &queryRes)
			require.NoError(t, err)
			var f any
			err = json.Unmarshal([]byte(test.aplResponse), &f)
			require.NoError(t, err)

			grpByArr := f.(map[string]any)["request"].(map[string]any)["groupBy"].([]any)
			queryResGrpBy := make([]string, 0, len(grpByArr))
			for _, grpByPart := range grpByArr {
				queryResGrpBy = append(queryResGrpBy, grpByPart.(string))
			}
			queryRes.GroupBy = queryResGrpBy

			got := buildFrame(&queryRes)
			t.Logf("%#v", got)
		})
	}
}
