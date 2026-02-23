package plugin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/axiomhq/axiom-go/axiom/query"
)

type AplQueryRequest struct {
	query.Options

	// APL is the APL query string.
	APL string `json:"apl"`
}

type DatasetFields struct {
	DatasetName string         `json:"datasetName"`
	Fields      []DatasetField `json:"fields"`
}

type DatasetField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Unit        string `json:"unit"`
	Hidden      bool   `json:"hidden"`
	Description string `json:"description"`
}

func (d *Datasource) DatasetFields(ctx context.Context) ([]*DatasetFields, error) {
	path, err := url.JoinPath(d.apiHost, "v1/datasets/_fields")
	if err != nil {
		return nil, err
	}

	req, err := d.client.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var res []*DatasetFields
	_, err = d.client.Do(req, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// EdgeQueryRequest represents the APL query request for edge endpoints.
type EdgeQueryRequest struct {
	APL       string    `json:"apl"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

// EdgeQueryResponse represents the tabular query response from edge endpoints.
type EdgeQueryResponse struct {
	Format string       `json:"format"`
	Status query.Status `json:"status"`
	Tables []query.Table `json:"tables"`
}

// buildQueryEndpoint returns the query endpoint URL.
// Priority: edgeURL > edge > apiHost (for non-edge queries)
//
// Edge query path: /v1/query/_apl
// Non-edge query path: /v1/datasets/_apl (legacy, via apiHost)
//
// If edgeURL has a custom path, it's used as-is.
// If edgeURL has no path (or only "/"), /v1/query/_apl is appended.
// If edge is set (domain only), builds https://{edge}/v1/query/_apl.
func (d *Datasource) buildQueryEndpoint() (string, error) {
	// Priority 1: edgeURL takes precedence
	if d.edgeURL != "" {
		edgeURL := strings.TrimSuffix(d.edgeURL, "/")

		parsed, err := url.Parse(edgeURL)
		if err != nil {
			return "", fmt.Errorf("failed to parse edgeURL: %w", err)
		}

		path := parsed.Path
		if path == "" || path == "/" {
			// No custom path, append edge query path
			return fmt.Sprintf("%s/v1/query/_apl", edgeURL), nil
		}

		// edgeURL has a custom path, use as-is
		return edgeURL, nil
	}

	// Priority 2: edge domain
	if d.edge != "" {
		edge := strings.TrimSuffix(d.edge, "/")
		return fmt.Sprintf("https://%s/v1/query/_apl", edge), nil
	}

	// Default: use apiHost with legacy query path
	if d.apiHost != "" {
		host := strings.TrimSuffix(d.apiHost, "/")
		return fmt.Sprintf("%s/v1/datasets/_apl", host), nil
	}

	return "https://api.axiom.co/v1/datasets/_apl", nil
}

// QueryEdge executes an APL query against the configured endpoint
// (edge or legacy apiHost, depending on configuration).
func (d *Datasource) QueryEdge(ctx context.Context, apl string, startTime, endTime time.Time) (*query.Result, error) {
	endpoint, err := d.buildQueryEndpoint()
	if err != nil {
		return nil, err
	}

	// Add format=tabular query param
	if strings.Contains(endpoint, "?") {
		endpoint += "&format=tabular"
	} else {
		endpoint += "?format=tabular"
	}

	reqBody := EdgeQueryRequest{
		APL:       apl,
		StartTime: startTime,
		EndTime:   endTime,
	}

	req, err := d.client.NewRequest(ctx, http.MethodPost, endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	var res EdgeQueryResponse
	_, err = d.client.Do(req, &res)
	if err != nil {
		return nil, err
	}

	return &query.Result{
		Status: res.Status,
		Tables: res.Tables,
	}, nil
}
