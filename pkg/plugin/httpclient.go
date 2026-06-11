package plugin

import (
	"net/http"
)

// type APIClient struct {
// 	http *http.Client
// }

type authTransport struct {
	base  http.RoundTripper
	token string
}

func (t authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.Header.Set("Authorization", "Bearer "+t.token)
	cloned.Header.Set("X-My-Header", "value")

	return t.base.RoundTrip(cloned)
}

// func newClient(baseURL string, accessToken string) *APIClient {
// 	u, err := url.Parse(baseURL)
// 	if err != nil {
// 		u = &url.URL{}
// 	}

// 	client := http.Client{
// 		Transport: authTransport{
// 			base:  http.DefaultTransport,
// 			token: accessToken,
// 		},
// 		Timeout: 5 * time.Minute,
// 	}

// 	return &APIClient{
// 		http:    &client,
// 		baseURL: u,
// 	}
// }

// func (c *APIClient) NewRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
// 	u, err := url.Parse(path)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if !u.IsAbs() {
// 		u = c.baseURL.ResolveReference(u)
// 	}

// 	var reader io.Reader
// 	if body != nil {
// 		if r, ok := body.(io.Reader); ok {
// 			reader = r
// 		} else {
// 			b, err := json.Marshal(body)
// 			if err != nil {
// 				return nil, err
// 			}
// 			reader = bytes.NewReader(b)
// 		}
// 	}

// 	req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if body != nil && req.Header.Get("Content-Type") == "" {
// 		req.Header.Set("Content-Type", "application/json")
// 	}
// 	req.Header.Set("User-Agent", fmt.Sprintf("axiom-grafana/v%s", Version))

// 	return req, nil
// }

// func (c *APIClient) Do(req *http.Request, out any) (*http.Response, error) {
// 	resp, err := c.http.Do(req)
// 	if err != nil {
// 		return resp, err
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
// 		body, _ := io.ReadAll(resp.Body)
// 		return resp, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
// 	}

// 	if out == nil {
// 		return resp, nil
// 	}

// 	err = json.NewDecoder(resp.Body).Decode(out)
// 	if err != nil && err != io.EOF {
// 		return resp, err
// 	}

// 	return resp, nil
// }
