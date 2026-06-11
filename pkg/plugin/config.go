package plugin

import (
	"context"
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type PluginConfig struct {
	AccessToken string `json:"accessToken"`
	APIHost     string `json:"apiHost"`
	EdgeURL     string `json:"edgeUrl"`
}

func parseConfig(ctx context.Context, settings backend.DataSourceInstanceSettings) (*PluginConfig, error) {
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
		host = apiHost.(string)
	}

	edge := checkString(data["edge"])
	edgeURL := checkString(data["edgeURL"])

	resolvedEdgeURL, err := resolveBaseUrl(urlInput{
		EdgeURL: edgeURL,
		Edge:    edge,
		APIHost: host,
	})
	if err != nil {
		logger.Error("failed to resolve correct axiom api/edge url", "error", err)
		return nil, err
	}

	return &PluginConfig{
		AccessToken: accessToken,
		APIHost:     host,
		EdgeURL:     resolvedEdgeURL,
	}, nil
}
