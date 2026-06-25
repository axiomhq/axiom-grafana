package axiomapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/axiomhq/axiom-go/axiom"
	"github.com/axiomhq/axiom-go/axiom/query"
	"github.com/axiomhq/axiom-grafana/pkg/config"
	"github.com/axiomhq/axiom-grafana/pkg/version"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type Client struct {
	apiURL  string
	edgeURL string
	client  *http.Client
}

type Dataset struct {
	Name string `json:"name"`
	Kind string
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

// APLQueryRequest represents the APL query request for edge endpoints.
type APLQueryRequest struct {
	APL       *string   `json:"apl"`
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

// MPLQueryRequest represents the MPL query request for edge endpoints.
type MPLQueryRequest struct {
	MPL        *string   `json:"mpl"`
	StartTime  time.Time `json:"startTime"`
	EndTime    time.Time `json:"endTime"`
	ChartWidth int64     `json:"-"`
}

// APLQueryResponse represents the tabular query response from edge endpoints.
type APLQueryResponse struct {
	Format        string                       `json:"format"`
	Status        *APLQueryStatus              `json:"status"`
	Tables        []query.Table                `json:"tables"`
	DatasetNames  []string                     `json:"datasetNames"`
	FieldsMetaMap map[string][]APLFieldMetaMap `json:"fieldsMetaMap"`
	TraceID       string                       `json:"-"`
}

type APLQueryStatus struct {
	ElapsedTime    int64           `json:"elapsedTime"`
	BlocksExamined int64           `json:"blocksExamined"`
	BlocksCached   int64           `json:"blocksCached"`
	BlocksMatched  int64           `json:"blocksMatched"`
	BlocksSkipped  int64           `json:"blocksSkipped"`
	RowsExamined   int64           `json:"rowsExamined"`
	RowsMatched    int64           `json:"rowsMatched"`
	NumGroups      int64           `json:"numGroups"`
	IsPartial      bool            `json:"isPartial"`
	CacheStatus    int64           `json:"cacheStatus"`
	MinBlockTime   *time.Time      `json:"minBlockTime"`
	MaxBlockTime   *time.Time      `json:"maxBlockTime"`
	Messages       []query.Message `json:"messages"`
}

type APLFieldMetaMap struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Unit        string `json:"unit"`
	Hidden      bool   `json:"hidden"`
	Description string `json:"description"`
}

type MetricsQueryResponse struct {
	Metadata MetricsQueryMetadata `json:"metadata"`
	Series   []MetricsQuerySeries `json:"series"`
	TraceID  string               `json:"-"`
}

type MetricsQueryMetadata struct {
	Unit     string   `json:"unit"`
	Warnings []string `json:"warnings"`
}

type MetricsQuerySeries struct {
	Resolution int
	Start      int64
	Data       []*float64
	Tags       map[string]string
	Metric     string
}

func NewClient(opts httpclient.Options, c *config.PluginConfig) (*Client, error) {
	if opts.Header == nil {
		opts.Header = http.Header{}
	}
	if c.AccessToken != "" {
		opts.Header.Set("Authorization", "Bearer "+c.AccessToken)
	}
	// set the SDK identifier
	opts.Header.Set("User-Agent", fmt.Sprintf("axiom-grafana/v%s", version.Version))
	// increase timeout for metrics queries
	if opts.Timeouts == nil {
		timeouts := httpclient.DefaultTimeoutOptions
		opts.Timeouts = &timeouts
	}
	opts.Timeouts.Timeout = 5 * time.Minute

	client, err := httpclient.New(opts)
	if err != nil {
		return nil, err
	}

	return &Client{
		apiURL:  c.APIHost,
		edgeURL: c.EdgeURL,
		client:  client,
	}, nil
}

func (api *Client) DatasetFields(ctx context.Context) ([]*DatasetFields, error) {
	endpoint := "/v1/datasets/_fields"
	path, err := url.JoinPath(api.apiURL, endpoint)
	if err != nil {
		return nil, err
	}

	req, err := api.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var res []*DatasetFields
	_, err = api.Do(req, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (api *Client) Datasets(ctx context.Context) ([]Dataset, error) {
	endpoint := "/v2/datasets"
	path, err := url.JoinPath(api.apiURL, endpoint)
	if err != nil {
		return nil, err
	}

	req, err := api.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var res []Dataset
	_, err = api.Do(req, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (api *Client) FetchMetricsDataset(ctx context.Context) ([]string, error) {
	logger := log.DefaultLogger.FromContext(ctx)
	res, err := api.Datasets(ctx)
	if err != nil {
		logger.Error(err.Error())
		return []string{}, err
	}

	datasets := []string{}

	for _, ds := range res {
		if ds.Kind == "otel:metrics:v1" {
			datasets = append(datasets, ds.Name)
		}
	}

	return datasets, nil
}

func (api *Client) GetMetricsForDataset(ctx context.Context, dataset string, startTime, endTime string) ([]string, error) {
	endpoint := fmt.Sprintf("/v1/query/metrics/info/datasets/%s/metrics", url.PathEscape(dataset))
	path, err := url.JoinPath(api.edgeURL, endpoint)
	if err != nil {
		return nil, err
	}

	path = fmt.Sprintf("%s?start=%s&end=%s", path, startTime, endTime)

	req, err := api.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var res []string
	_, err = api.Do(req, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (api *Client) GetMetricTags(ctx context.Context, dataset string, metric string, startTime, endTime string) ([]string, error) {
	endpoint := fmt.Sprintf("/v1/query/metrics/info/datasets/%s/tags", url.PathEscape(dataset))
	if metric != "" {
		endpoint = fmt.Sprintf("/v1/query/metrics/info/datasets/%s/metrics/%s/tags", url.PathEscape(dataset), url.PathEscape(metric))
	}
	path, err := url.JoinPath(api.edgeURL, endpoint)
	if err != nil {
		return nil, err
	}
	path = fmt.Sprintf("%s?start=%s&end=%s", path, startTime, endTime)

	req, err := api.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var res []string
	_, err = api.Do(req, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (api *Client) GetMetricTagValues(ctx context.Context, dataset string, metric string, tag string, startTime, endTime string) ([]string, error) {
	endpoint := fmt.Sprintf("/v1/query/metrics/info/datasets/%s/tags/%s/values", url.PathEscape(dataset), url.PathEscape(tag))
	if metric != "" {
		endpoint = fmt.Sprintf("/v1/query/metrics/info/datasets/%s/metrics/%s/tags/%s/values", url.PathEscape(dataset), url.PathEscape(metric), url.PathEscape(tag))
	}
	path, err := url.JoinPath(api.edgeURL, endpoint)
	if err != nil {
		return nil, err
	}
	path = fmt.Sprintf("%s?start=%s&end=%s", path, startTime, endTime)

	req, err := api.NewRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var res []string
	_, err = api.Do(req, &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (api *Client) QueryAPL(ctx context.Context, reqBody APLQueryRequest) (APLQueryResponse, error) {
	endpoint := "/v1/query/_apl"
	path, err := url.JoinPath(api.edgeURL, endpoint)
	if err != nil {
		return APLQueryResponse{}, err
	}

	path = path + "?format=tabular"

	req, err := api.NewRequest(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return APLQueryResponse{}, err
	}

	var result APLQueryResponse
	resp, err := api.Do(req, &result)
	if err != nil {
		return APLQueryResponse{}, err
	}
	result.TraceID = traceIDFromResponse(resp)

	return result, nil
}

func (api *Client) QueryMetrics(ctx context.Context, reqBody MPLQueryRequest) (MetricsQueryResponse, error) {
	endpoint := "/v1/query/_mpl"
	path, err := url.JoinPath(api.edgeURL, endpoint)
	if err != nil {
		return MetricsQueryResponse{}, err
	}

	req, err := api.NewRequest(ctx, http.MethodPost, path, reqBody)
	if err != nil {
		return MetricsQueryResponse{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.metrics.v3+json")
	if reqBody.ChartWidth > 0 {
		req.Header.Set("x-axiom-chart-width", strconv.FormatInt(reqBody.ChartWidth, 10))
	}

	var res MetricsQueryResponse
	resp, err := api.Do(req, &res)
	if err != nil {
		return MetricsQueryResponse{}, err
	}
	res.TraceID = traceIDFromResponse(resp)

	return res, nil
}

func traceIDFromResponse(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	return resp.Header.Get("X-Axiom-Trace-Id")
}

func (api *Client) NewRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	var reader io.Reader
	if body != nil {
		if r, ok := body.(io.Reader); ok {
			reader = r
		} else {
			b, err := json.Marshal(body)
			if err != nil {
				return nil, err
			}
			reader = bytes.NewReader(b)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
	if err != nil {
		return nil, err
	}
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}

	return req, nil
}

func (api *Client) Do(req *http.Request, out any) (*http.Response, error) {
	resp, err := api.client.Do(req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return resp, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	if out == nil {
		return resp, nil
	}

	err = json.NewDecoder(resp.Body).Decode(out)
	if err != nil && err != io.EOF {
		return resp, err
	}

	return resp, nil
}

// ValidateCredentials validates the credentials by performing an APL query that we expect to fail (empty)
// validate that we get HTTP 422, this gives high confidence
// that we got past network and authentication issues and looked at our request
// it also should be somewhat inexpensive for the server
func (api *Client) ValidateCredentials(ctx context.Context) error {
	logger := log.DefaultLogger.FromContext(ctx)

	var axiErr axiom.HTTPError
	path, err := url.JoinPath(api.edgeURL, "/v1/query/_apl")
	if err != nil {
		return err
	}
	r, err := api.NewRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return err
	}
	res, err := api.client.Do(r)
	if err != nil && errors.As(err, &axiErr) {
		if axiErr.Status == 422 {
			// expected 422 for empty query, HEALTHY
			return nil
		}
	}
	if err != nil {
		logger.Error("failed to query Axiom", "error", err.Error())
		return fmt.Errorf("invalid edge url or API token")
	}
	if res == nil {
		return fmt.Errorf("no response received from %s", api.edgeURL)
	}
	defer res.Body.Close()

	if res.StatusCode != 422 {
		return fmt.Errorf("unexpected status %d", res.StatusCode)
	}

	return nil
}
