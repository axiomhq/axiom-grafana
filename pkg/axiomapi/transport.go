package axiomapi

import "net/http"

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
