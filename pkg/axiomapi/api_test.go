package axiomapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/axiomhq/axiom-grafana/pkg/config"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type closeTrackingBody struct {
	io.Reader
	closed bool
}

func (b *closeTrackingBody) Close() error {
	b.closed = true
	return nil
}

func TestNewClientAcceptsZeroValueHTTPClientOptions(t *testing.T) {
	client, err := NewClient(httpclient.Options{}, &config.PluginConfig{
		APIHost: "https://api.axiom.co",
		EdgeURL: "https://api.axiom.co",
	})
	if err != nil {
		t.Fatalf("expected zero-value options to build client, got error: %v", err)
	}
	if client == nil {
		t.Fatal("expected client")
	}
}

func TestNewRequestSetsDefaultJSONHeaders(t *testing.T) {
	client := &Client{}

	getReq, err := client.NewRequest(context.Background(), http.MethodGet, "https://api.axiom.co/v2/datasets", nil)
	if err != nil {
		t.Fatalf("expected request, got error: %v", err)
	}
	if got := getReq.Header.Get("Accept"); got != "application/json" {
		t.Fatalf("expected default Accept header, got %q", got)
	}
	if got := getReq.Header.Get("Content-Type"); got != "" {
		t.Fatalf("expected no Content-Type for request without body, got %q", got)
	}

	postReq, err := client.NewRequest(context.Background(), http.MethodPost, "https://api.axiom.co/v1/query/_apl", map[string]string{"apl": ""})
	if err != nil {
		t.Fatalf("expected request, got error: %v", err)
	}
	if got := postReq.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type header for request body, got %q", got)
	}
}

func TestValidateCredentialsClosesResponseBody(t *testing.T) {
	body := &closeTrackingBody{Reader: strings.NewReader("expected validation error")}
	client := &Client{
		edgeURL: "https://api.axiom.co",
		client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost {
					t.Fatalf("expected POST request, got %s", req.Method)
				}
				if req.URL.Path != "/v1/query/_apl" {
					t.Fatalf("expected validation path, got %s", req.URL.Path)
				}

				return &http.Response{
					StatusCode: http.StatusUnprocessableEntity,
					Header:     make(http.Header),
					Body:       body,
					Request:    req,
				}, nil
			}),
		},
	}

	if err := client.ValidateCredentials(context.Background()); err != nil {
		t.Fatalf("expected credentials to validate, got error: %v", err)
	}
	if !body.closed {
		t.Fatal("expected response body to be closed")
	}
}

func TestValidateCredentialsReturnsTransportError(t *testing.T) {
	doErr := errors.New("dial failed")
	client := &Client{
		edgeURL: "https://api.axiom.co",
		client: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, doErr
			}),
		},
	}

	err := client.ValidateCredentials(context.Background())
	if err == nil {
		t.Fatal("expected validation error for transport failure")
	}
	if err.Error() != "invalid edge url or API token" {
		t.Fatalf("expected validation error for transport failure, got %v", err)
	}
}

func TestMetricsQueryResponseDecodesNonStringTagValues(t *testing.T) {
	body := `{
		"metadata":{"unit":"ms"},
		"series":[
			{"metric":"http.requests","tags":{"status_code":200,"le":0.5,"sampled":true,"zone":null,"route":"/user\u002fprofile"},"start":1781186400,"resolution":60,"data":[1.5,null]},
			{"metric":"http.requests","tags":{},"start":1781186400,"resolution":60,"data":[]},
			{"metric":"http.requests","start":1781186400,"resolution":60,"data":[2]}
		]
	}`

	var res MetricsQueryResponse
	if err := json.Unmarshal([]byte(body), &res); err != nil {
		t.Fatalf("expected non-string tag values to decode, got error: %v", err)
	}
	if len(res.Series) != 3 {
		t.Fatalf("expected 3 series, got %d", len(res.Series))
	}

	want := map[string]string{"status_code": "200", "le": "0.5", "sampled": "true", "zone": "", "route": "/user/profile"}
	if got := map[string]string(res.Series[0].Tags); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected tags %v, got %v", want, got)
	}
	if got := res.Series[1].Tags; got == nil || len(got) != 0 {
		t.Fatalf("expected empty tags, got %v", got)
	}
	if got := res.Series[2].Tags; got != nil {
		t.Fatalf("expected nil tags when the field is missing, got %v", got)
	}

	first := res.Series[0]
	if first.Metric != "http.requests" || first.Start != 1781186400 || first.Resolution != 60 {
		t.Fatalf("unexpected series metadata: %+v", first)
	}
	if len(first.Data) != 2 || first.Data[0] == nil || *first.Data[0] != 1.5 || first.Data[1] != nil {
		t.Fatalf("unexpected series data: %v", first.Data)
	}
}
