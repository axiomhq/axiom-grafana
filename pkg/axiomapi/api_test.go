package axiomapi

import (
	"context"
	"errors"
	"io"
	"net/http"
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
