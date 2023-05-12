package plugin

import (
	"context"
	"fmt"
	"github.com/axiomhq/axiom-go/axiom"
	"github.com/axiomhq/axiom-go/axiom/query"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"net/http"
	"net/url"
)

// All the code in this file is working around the fact that axiom-go hides
// the Request structure returned in the APL response.  We need this to know
// whether the query projected _time or _sysTime

type Projection struct {
	Field string `json:"field"`
	Alias string `json:"alias"`
}

type AplQueryResponse struct {
	query.Result

	// HINT(lukasmalkmus): Ignore these fields as they are not relevant for the
	// user and/or will change with the new query result format.
	LegacyRequest struct {
		StartTime         any          `json:"startTime"`
		EndTime           any          `json:"endTime"`
		Resolution        any          `json:"resolution"`
		Aggregations      any          `json:"aggregations"`
		Filter            any          `json:"filter"`
		Order             any          `json:"order"`
		Limit             any          `json:"limit"`
		VirtualFields     any          `json:"virtualFields"`
		Projections       []Projection `json:"project"`
		Cursor            any          `json:"cursor"`
		IncludeCursor     any          `json:"includeCursor"`
		ContinuationToken any          `json:"continuationToken"`

		// HINT(lukasmalkmus): Preserve the legacy request's "groupBy"
		// field for now. This is needed to properly render some results.
		GroupBy []string `json:"groupBy"`
	} `json:"request"`
	FieldsMeta any `json:"fieldsMetaMap"`
}

type AplQueryRequest struct {
	query.Options

	// APL is the APL query string.
	APL string `json:"apl"`
}

func (d *Datasource) QueryOverride(ctx context.Context, apl string, options ...query.Option) (*AplQueryResponse, error) {
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
		Format: "legacy", // Hardcode legacy APL format for now.
	}

	path, err := url.JoinPath(d.apiHost, "v1/datasets/_apl")
	if err != nil {
		return nil, err
	} else if path, err = axiom.AddURLOptions(path, queryParams); err != nil {
		return nil, err
	}

	log.DefaultLogger.Info(fmt.Sprintf("query path is: %s", path))

	req, err := d.client.NewRequest(ctx, http.MethodPost, path, AplQueryRequest{
		Options: opts,

		APL: apl,
	})
	if err != nil {
		return nil, err
	}

	var res AplQueryResponse
	if _, err = d.client.Do(req, &res); err != nil {
		return nil, err
	}

	return &res, nil
}
