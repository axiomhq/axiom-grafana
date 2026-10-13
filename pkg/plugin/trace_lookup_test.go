package plugin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	axiQuery "github.com/axiomhq/axiom-go/axiom/query"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

func TestTraceLookupAPL(t *testing.T) {
	apl, err := traceLookupAPL("spans", "", "abc")
	require.NoError(t, err)
	require.Equal(t, "['spans'] | where ['trace_id'] == 'abc' | extend traceID = ['trace_id']", apl)
	apl, err = traceLookupAPL("a'b", "trace.id", "x' | take 1")
	require.NoError(t, err)
	require.Contains(t, apl, "['a\\'b']")
	require.Contains(t, apl, "== 'x\\' | take 1'")
	_, err = traceLookupAPL("", "", "abc")
	require.Error(t, err)
	_, err = traceLookupAPL("spans", "", "")
	require.Error(t, err)
}

func TestTraceLookupQueryData(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/query/_apl", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "['spans'] | where ['trace_id'] == 'abc' | extend traceID = ['trace_id']", body["apl"])
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"tables":[{"fields":[{"name":"traceID","type":"string"},{"name":"span_id","type":"string"},{"name":"parent_span_id","type":"string"},{"name":"name","type":"string"},{"name":"service.name","type":"string"},{"name":"_time","type":"datetime"},{"name":"duration","type":"timespan"}],"columns":[["abc","abc"],["root","child"],[null,"root"],["request","db"],["api","db"],["2026-10-13T00:00:00Z","2026-10-13T00:00:00Z"],["2ms","1ms"]]}]}`))
		require.NoError(t, err)
	}))
	defer upstream.Close()
	ds := Datasource{api: newTestAxiomClient(t, upstream.URL, upstream.URL)}
	resp, err := ds.QueryData(context.Background(), &backend.QueryDataRequest{Queries: []backend.DataQuery{{RefID: "A", JSON: json.RawMessage(`{"kind":"trace","dataset":"spans","traceId":"abc","totals":true}`)}}})
	require.NoError(t, err)
	require.NoError(t, resp.Responses["A"].Error)
	require.Len(t, resp.Responses["A"].Frames, 1)
	frame := resp.Responses["A"].Frames[0]
	require.EqualValues(t, data.VisTypeTrace, frame.Meta.PreferredVisualization)
	require.Equal(t, "root", *frame.Fields[2].At(1).(*string))
}

func TestTraceMetadataAndInvalidSpans(t *testing.T) {
	table := &axiQuery.Table{Fields: []axiQuery.Field{{Name: "trace_id"}, {Name: "span_id"}, {Name: "name"}, {Name: "service.name"}, {Name: "_time"}, {Name: "duration"}, {Name: "status.code"}, {Name: "links"}}, Columns: []axiQuery.Column{{"a", "a"}, {"root", ""}, {"request", "bad"}, {"api", "api"}, {"2026-10-13T00:00:00Z", "bad"}, {"1ms", "bad"}, {float64(2), float64(0)}, {[]any{map[string]any{"trace_id": "other", "span_id": "linked"}}, nil}}}
	frame, err := buildTraceFrame(context.Background(), table)
	require.NoError(t, err)
	require.Equal(t, 1, frame.Fields[0].Len())
	require.Len(t, frame.Meta.Notices, 1)
	tags := frame.Fields[9].At(0).(*json.RawMessage)
	require.JSONEq(t, `[{"key":"error","value":true}]`, string(*tags))
	for _, field := range frame.Fields {
		if field.Name == "references" {
			require.JSONEq(t, `[{"traceID":"other","spanID":"linked"}]`, string(*field.At(0).(*json.RawMessage)))
		}
	}
}
