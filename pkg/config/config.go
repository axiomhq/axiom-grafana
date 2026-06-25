package config

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/axiomhq/axiom-grafana/pkg/util"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type PluginConfig struct {
	AccessToken string `json:"accessToken"`
	APIHost     string `json:"apiHost"`
	Edge        string `json:"edge"`
	EdgeURL     string `json:"edgeURL"`
}

func ParseConfig(ctx context.Context, settings backend.DataSourceInstanceSettings) (*PluginConfig, error) {
	logger := log.DefaultLogger.FromContext(ctx)
	accessToken := ""
	if token, exists := settings.DecryptedSecureJSONData["accessToken"]; exists {
		// Use the decrypted API key.
		accessToken = token
	}

	var data map[string]any
	err := json.Unmarshal(settings.JSONData, &data)
	if err != nil {
		logger.Error("failed to unmarshal settings", "error", err)
		return nil, err
	}
	host := "https://api.axiom.co"
	if apiHost, exists := data["apiHost"]; exists {
		if s, ok := apiHost.(string); ok {
			host = s
		}
	}

	edge := util.CheckString(data["edge"])
	edgeURL := util.CheckString(data["edgeURL"])

	resolvedEdgeURL, err := resolveEdgeUrl(edge, edgeURL)
	if err != nil {
		logger.Error("failed to resolve correct axiom api/edge url", "error", err)
		return nil, err
	}

	return &PluginConfig{
		AccessToken: accessToken,
		APIHost:     host,
		Edge:        edge,
		EdgeURL:     resolvedEdgeURL,
	}, nil
}

func resolveEdgeUrl(edge string, edgeUrl string) (string, error) {
	// Priority 1: edgeURL takes precedence
	if edgeUrl != "" {
		edgeUrl := strings.TrimSuffix(edgeUrl, "/")

		// edgeURL has a custom path, use as-is
		return edgeUrl, nil
	}

	// Priority 2: edge domain
	if edge != "" {
		edge := strings.TrimSuffix(edge, "/")
		return fmt.Sprintf("https://%s", edge), nil
	}

	return "", fmt.Errorf("Edge URL is required. Please configure the Edge URL in the Axiom Grafana datasource settings.")
}
