package config

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

func TestResolveBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		apiHost  string
		edge     string
		edgeURL  string
		expected string
		err      string
	}{
		{
			name:    "no edge - returns error",
			apiHost: "https://api.axiom.co",
			err:     "Edge URL is required. Please configure the Edge URL in the Axiom Grafana datasource settings.",
		},
		{
			name:     "edge domain",
			apiHost:  "https://api.axiom.co",
			edge:     "eu-central-1.aws.edge.axiom.co",
			expected: "https://eu-central-1.aws.edge.axiom.co",
		},
		{
			name:     "edge domain with trailing slash",
			apiHost:  "https://api.axiom.co",
			edge:     "us-east-1.aws.edge.axiom.co/",
			expected: "https://us-east-1.aws.edge.axiom.co",
		},
		{
			name:     "edgeURL without path",
			apiHost:  "https://api.axiom.co",
			edgeURL:  "https://eu-central-1.aws.edge.axiom.co",
			expected: "https://eu-central-1.aws.edge.axiom.co",
		},
		{
			name:     "edgeURL with trailing slash",
			apiHost:  "https://api.axiom.co",
			edgeURL:  "https://eu-central-1.aws.edge.axiom.co/",
			expected: "https://eu-central-1.aws.edge.axiom.co",
		},
		{
			name:     "edgeURL with custom path - used as-is",
			edgeURL:  "http://localhost:3400/query",
			expected: "http://localhost:3400/query",
		},
		{
			name:     "edgeURL takes precedence over edge domain",
			edgeURL:  "https://primary.edge.axiom.co",
			edge:     "secondary.edge.axiom.co",
			expected: "https://primary.edge.axiom.co",
		},
		{
			name:    "legacy EU instance - no edge returns error",
			apiHost: "https://api.eu.axiom.co",
			err:     "Edge URL is required. Please configure the Edge URL in the Axiom Grafana datasource settings.",
		},
		{
			name:     "staging edge domain",
			edge:     "us-east-1.edge.staging.axiomdomain.co",
			expected: "https://us-east-1.edge.staging.axiomdomain.co",
		},
		{
			name: "no apiHost, no edge - returns error",
			err:  "Edge URL is required. Please configure the Edge URL in the Axiom Grafana datasource settings.",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			endpoint, err := resolveEdgeUrl(test.edge, test.edgeURL)
			if test.err != "" {
				require.EqualError(t, err, test.err)
				require.Empty(t, endpoint)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.expected, endpoint)
		})
	}
}

func TestParseConfigReadsEdgeURL(t *testing.T) {
	settings := backend.DataSourceInstanceSettings{
		JSONData: json.RawMessage(`{
			"apiHost": "https://api.axiom.co",
			"edge": "legacy.edge.axiom.co",
			"edgeURL": "https://primary.edge.axiom.co"
		}`),
	}

	cfg, err := ParseConfig(context.Background(), settings)

	require.NoError(t, err)
	require.Equal(t, "https://primary.edge.axiom.co", cfg.EdgeURL)
}

func TestParseConfigFallsBackToLegacyEdgeDomain(t *testing.T) {
	settings := backend.DataSourceInstanceSettings{
		JSONData: json.RawMessage(`{
			"apiHost": "https://api.axiom.co",
			"edge": "eu-central-1.aws.edge.axiom.co"
		}`),
	}

	cfg, err := ParseConfig(context.Background(), settings)

	require.NoError(t, err)
	require.Equal(t, "https://eu-central-1.aws.edge.axiom.co", cfg.EdgeURL)
}

func TestParseConfigRequiresEdgeURL(t *testing.T) {
	settings := backend.DataSourceInstanceSettings{
		JSONData: json.RawMessage(`{
			"apiHost": "https://api.axiom.co"
		}`),
	}

	cfg, err := ParseConfig(context.Background(), settings)

	require.Nil(t, cfg)
	require.EqualError(t, err, "Edge URL is required. Please configure the Edge URL in the Axiom Grafana datasource settings.")
}
