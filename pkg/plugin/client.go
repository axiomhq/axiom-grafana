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
// Priority: apiHost (with path detection) > region > default cloud endpoint
//
// If apiHost has a custom path, it's used as-is.
// If apiHost has no path (or only "/"), the query path is appended for backwards compatibility.
// If region is set (and apiHost has no custom path), use the edge endpoint.
func (d *Datasource) buildQueryEndpoint() (string, error) {
	// If apiHost is set, check if it has a custom path
	if d.apiHost != "" {
		host := strings.TrimSuffix(d.apiHost, "/")

		// Parse URL to check if path is provided
		parsed, err := url.Parse(host)
		if err != nil {
			return "", fmt.Errorf("failed to parse apiHost: %w", err)
		}

		path := parsed.Path
		// If path is empty or just "/", we need to append the query path
		if path == "" || path == "/" {
			// Check if region is set - if so, use region for queries
			if d.region != "" {
				region := strings.TrimSuffix(d.region, "/")
				// Ensure we have a proper URL with scheme
				if !strings.HasPrefix(region, "http://") && !strings.HasPrefix(region, "https://") {
					region = "https://" + region
				}
				return fmt.Sprintf("%s/v1/datasets/_apl?format=tabular", region), nil
			}
			// No region, append query path to apiHost for backwards compatibility
			return fmt.Sprintf("%s/v1/datasets/_apl", host), nil
		}

		// apiHost has a custom path, use as-is (apiHost takes precedence)
		return host, nil
	}

	// No apiHost set, check for region
	if d.region != "" {
		region := strings.TrimSuffix(d.region, "/")
		if !strings.HasPrefix(region, "http://") && !strings.HasPrefix(region, "https://") {
			region = "https://" + region
		}
		return fmt.Sprintf("%s/v1/datasets/_apl?format=tabular", region), nil
	}

	// Default: use cloud endpoint
	return "https://api.axiom.co/v1/datasets/_apl", nil
}

// QueryEdge executes an APL query against the edge endpoint.
func (d *Datasource) QueryEdge(ctx context.Context, apl string, startTime, endTime time.Time) (*query.Result, error) {
	endpoint, err := d.buildQueryEndpoint()
	if err != nil {
		return nil, err
	}

	// Add format=tabular query param if not already present (for apiHost fallback)
	if !strings.Contains(endpoint, "format=") {
		if strings.Contains(endpoint, "?") {
			endpoint += "&format=tabular"
		} else {
			endpoint += "?format=tabular"
		}
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
