package plugin

import (
	"context"
	"fmt"
	"net/http"
)

type AxiomAPI struct {
	apiHost string
	client  *APIClient
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

func (d *AxiomAPI) DatasetFields(ctx context.Context) ([]*DatasetFields, error) {
	endpoint := "v1/datasets/_fields"

	req, err := d.client.NewRequest(ctx, http.MethodGet, endpoint, nil)
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

func (d *AxiomAPI) Datasets(ctx context.Context) ([]string, error) {
	endpoint := "/v2/datasets"

	req, err := d.client.NewRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var res []string
	_, err = d.client.Do(req, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (d *AxiomAPI) GetMetricsForDataset(ctx context.Context, dataset string) ([]string, error) {
	endpoint := fmt.Sprintf("v1/query/metrics/info/datasets/%s/metrics", dataset)

	req, err := d.client.NewRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var res []string
	_, err = d.client.Do(req, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (d *AxiomAPI) GetMetricTags(ctx context.Context, dataset string, metric string) ([]string, error) {
	endpoint := fmt.Sprintf("/v1/query/metrics/info/datasets/%s/tags", dataset)
	if metric != "" {
		endpoint = fmt.Sprintf("/v1/query/metrics/info/datasets/%s/metrics/%s/tags", dataset, metric)
	}

	req, err := d.client.NewRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var res []string
	_, err = d.client.Do(req, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}
