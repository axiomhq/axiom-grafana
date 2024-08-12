package plugin

import (
	"context"
	"net/http"
	"net/url"

	"github.com/axiomhq/axiom-go/axiom"
	"github.com/axiomhq/axiom-go/axiom/query"
)

type AplQueryRequest struct {
	query.Options

	// APL is the APL query string.
	APL string `json:"apl"`
}

func (d *Datasource) QueryOverride(ctx context.Context, apl string, options ...query.Option) (*query.Result, error) {
	// Apply supplied options.
	var opts query.Options
	for _, option := range options {
		option(&opts)
	}

	// The only query parameters supported can be hardcoded as they are not
	// configurable as of now.
	queryParams := struct {
		Format string `url:"format"`
	}{
		Format: "tabular",
	}

	path, err := url.JoinPath(d.apiHost, "v1/datasets/_apl")
	if err != nil {
		return nil, err
	} else if path, err = axiom.AddURLOptions(path, queryParams); err != nil {
		return nil, err
	}

	req, err := d.client.NewRequest(ctx, http.MethodPost, path, AplQueryRequest{
		Options: opts,

		APL: apl,
	})
	if err != nil {
		return nil, err
	}

	var res query.Result
	if _, err = d.client.Do(req, &res); err != nil {
		return nil, err
	}

	return &res, nil
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
