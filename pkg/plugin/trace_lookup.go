package plugin

import (
	"fmt"
	"strings"
)

// APL quoted literals escape backslashes and quotes, including bracket field access.
func aplLiteral(value string) string {
	return "'" + strings.NewReplacer("\\", "\\\\", "'", "\\'", "\n", "\\n", "\r", "\\r").Replace(value) + "'"
}

func traceLookupAPL(dataset, field, traceID string) (string, error) {
	if strings.TrimSpace(dataset) == "" || strings.TrimSpace(traceID) == "" {
		return "", fmt.Errorf("trace lookup requires a dataset and trace ID")
	}
	if field == "" {
		field = "trace_id"
	}
	return "[" + aplLiteral(dataset) + "] | where [" + aplLiteral(field) + "] == " + aplLiteral(traceID) + " | extend traceID = [" + aplLiteral(field) + "]", nil
}
