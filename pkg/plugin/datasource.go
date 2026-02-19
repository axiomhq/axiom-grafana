package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"slices"
	"strconv"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/concurrent"

	"github.com/axiomhq/axiom-go/axiom"
	axiQuery "github.com/axiomhq/axiom-go/axiom/query"
)

// Make sure Datasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler interfaces. Plugin should not implement all these
// interfaces- only those which are required for a particular task.
var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
	_ backend.CallResourceHandler   = (*Datasource)(nil)
)

// NewDatasource creates a new datasource instance.
func NewDatasource(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
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

	orgID := checkString(data["orgID"])

	client, err := axiom.NewClient(
		axiom.SetToken(accessToken),
		axiom.SetURL(host),
		axiom.SetOrganizationID(orgID),
		axiom.SetUserAgent(fmt.Sprintf("axiom-grafana/v%s", Version)),
	)
	if err != nil {
		logger.Error("failed to create axiom client", "error", err)
		return nil, err
	}

	ds := &Datasource{
		client:  client,
		apiHost: host,
	}
	resourceHandler := ds.newResourceHandler()
	ds.CallResourceHandler = resourceHandler

	logger.Debug("datasource & client created", "host", host)

	return ds, nil
}

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct {
	backend.CallResourceHandler
	apiHost string
	client  *axiom.Client
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *Datasource) Dispose() {
	// Clean up datasource instance resources.
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	logger := log.DefaultLogger.FromContext(ctx)
	// log panic
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
				logger.Error(err.Error())
				return
			}
			logger.Error(err.Error())
			logger.Error(string(debug.Stack()))
		}
	}()

	return concurrent.QueryData(ctx, req, d.query, 10)
}

type queryModel struct {
	APL    string `json:"apl"`
	Totals bool   `json:"totals"`
}

func (d *Datasource) query(ctx context.Context, query concurrent.Query) backend.DataResponse {
	logger := log.DefaultLogger.FromContext(ctx)
	// log panic
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("pkg: %v", r)
				logger.Error(err.Error())
				return
			}
			logger.Error(err.Error())
			logger.Error(string(debug.Stack()))
		}
	}()

	var response backend.DataResponse

	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	err := json.Unmarshal(query.DataQuery.JSON, &qm)
	if err != nil {
		// Log the actual error since it will be included in the Grafana server log and return a more generic message to the end user.
		logger.Error(err.Error())
		return backend.ErrDataResponse(backend.StatusInternal, "Could not parse query")
	}

	if qm.APL == "" {
		return backend.DataResponse{}
	}

	// make request to axiom
	result, err := d.client.Query(ctx, qm.APL, axiQuery.SetStartTime(query.DataQuery.TimeRange.From), axiQuery.SetEndTime(query.DataQuery.TimeRange.To))
	if err != nil {
		logger.Error("failed to query axiom", "error", err)
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("axiom error: %v", err.Error()))
	}

	var frame *data.Frame
	var newFrame *data.Frame
	if len(result.Tables) > 1 {
		var targetTable *axiQuery.Table
		if qm.Totals {
			targetTable = &result.Tables[1]
		} else {
			targetTable = &result.Tables[0]
		}

		frame = buildFrame(ctx, targetTable)
		newFrame, err = data.LongToWide(frame, nil)
		if err != nil {
			logger.Error("transformation from long to wide failed", "error", err.Error())
		}
	} else {
		frame = buildFrame(ctx, &result.Tables[0])
	}

	if newFrame != nil {
		response.Frames = append(response.Frames, newFrame)
	} else {
		response.Frames = append(response.Frames, frame)
	}

	return response
}

func buildFrame(ctx context.Context, result *axiQuery.Table) *data.Frame {
	frame := data.NewFrame("response")

	// Create field groups, each original field can produce 1 or more output fields
	fieldGroups := make([][]*data.Field, len(result.Fields))

	for i, f := range result.Fields {
		fieldGroups[i] = createField(f, result.Columns[i])
	}

	// Process data for all fields
	for colIndex, col := range result.Columns {
		fieldGroups[colIndex] = populateFieldData(ctx, fieldGroups[colIndex], col, result.Fields[colIndex])
	}

	// Flatten field groups into final fields array
	frame.Fields = slices.Concat(fieldGroups...)

	return frame
}

// createField creates appropriate field group for any field type
func createField(f axiQuery.Field, column []any) []*data.Field {
	if isHistogramField(f) {
		// Histogram fields: return empty group, will be populated during data processing
		return []*data.Field{}
	}

	// Regular fields: return single field group
	var field *data.Field
	switch f.Type {
	case "datetime":
		field = data.NewField(f.Name, nil, []time.Time{})
	case "integer":
		field = data.NewField(f.Name, nil, []*float64{})
	case "float":
		field = data.NewField(f.Name, nil, []*float64{})
	case "bool":
		field = data.NewField(f.Name, nil, []*bool{})
	case "timespan":
		field = data.NewField(f.Name, nil, []*string{})
	case "unknown":
		field = data.NewField(f.Name, nil, []*string{})
	case "array":
		// For non-histogram arrays, check the element type from the data
		if len(column) > 0 && column[0] != nil {
			switch column[0].(type) {
			case []float64:
				field = data.NewField(f.Name, nil, [][]*float64{})
			default:
				field = data.NewField(f.Name, nil, [][]*string{})
			}
		} else {
			field = data.NewField(f.Name, nil, [][]*string{})
		}
	default:
		field = data.NewField(f.Name, nil, []*string{})
	}
	return []*data.Field{field}
}

// populateFieldData populates field data for any field type
func populateFieldData(ctx context.Context, fieldGroup []*data.Field, col []any, fieldDef axiQuery.Field) []*data.Field {
	if isHistogramField(fieldDef) {
		// Process histogram field: create and populate bucket fields
		return processHistogramColumn(col)
	}

	logger := log.DefaultLogger.FromContext(ctx)

	// Process regular field: populate existing field (fieldGroup has exactly one field)
	field := fieldGroup[0]
	for _, val := range col {
		if val == nil {
			field.Append(nil)
			continue
		}

		switch fieldDef.Type {
		case "datetime":
			t, err := time.Parse(time.RFC3339, val.(string))
			if err != nil {
				logger.Warn("Failed to parse time", "time", val)
				field.Append(time.Time{})
				continue
			}
			field.Append(t)
		case "integer":
			num := val.(float64)
			field.Append(&num)
		case "float":
			num := val.(float64)
			field.Append(&num)
		case "string", "unknown":
			txt := val.(string)
			field.Append(&txt)
		case "bool":
			b := val.(bool)
			field.Append(&b)
		case "timespan":
			num := val.(string)
			field.Append(&num)
		default:
			txt := fmt.Sprintf("%v", val)
			field.Append(&txt)
		}
	}
	return fieldGroup
}

// processHistogramColumn processes a single histogram column and returns bucket fields
func processHistogramColumn(column []any) []*data.Field {
	// Collect all unique boundaries
	boundarySet := make(map[float64]bool)

	for _, cellValue := range column {
		if cellValue != nil {
			if histArray, ok := cellValue.([]any); ok {
				for _, bucket := range histArray {
					if bucketMap, ok := bucket.(map[string]any); ok {
						if toVal, ok := bucketMap["to"].(float64); ok {
							boundarySet[toVal] = true
						}
					}
				}
			}
		}
	}

	// Sort boundaries
	boundaries := make([]float64, 0, len(boundarySet))
	for boundary := range boundarySet {
		boundaries = append(boundaries, boundary)
	}
	slices.Sort(boundaries)

	// Create bucket fields
	bucketFields := make([]*data.Field, len(boundaries))
	for i, boundary := range boundaries {
		labels := map[string]string{"le": strconv.FormatFloat(boundary, 'g', -1, 64)}
		bucketFields[i] = data.NewField("", labels, []*float64{})
	}

	// Populate bucket fields
	for _, cellValue := range column {
		bucketCounts := make(map[float64]float64)

		if cellValue != nil {
			if histArray, ok := cellValue.([]any); ok {
				for _, bucket := range histArray {
					if bucketMap, ok := bucket.(map[string]any); ok {
						if toVal, ok := bucketMap["to"].(float64); ok {
							if countVal, ok := bucketMap["count"].(float64); ok {
								bucketCounts[toVal] = countVal
							}
						}
					}
				}
			}
		}

		for i, boundary := range boundaries {
			if count, exists := bucketCounts[boundary]; exists {
				bucketFields[i].Append(&count)
			} else {
				bucketFields[i].Append(nil)
			}
		}
	}

	return bucketFields
}

// isHistogramField checks if a field is a histogram aggregation field
func isHistogramField(field axiQuery.Field) bool {
	// Check if field has Aggregation metadata indicating histogram
	return field.Aggregation != nil && field.Aggregation.Op == axiQuery.OpHistogram
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	logger := log.DefaultLogger.FromContext(ctx)
	// first try to validate the credentials
	err := d.client.ValidateCredentials(ctx)
	if err != nil {
		logger.Error("Failed to validate credentials", "error", err)
		return &backend.CheckHealthResult{
			Status: backend.HealthStatusError,
			// simple error message, not the actual error
			Message: "error with datasource",
		}, nil
	}

	// perform an APL query that we expect to fail (empty)
	// validate that we get HTTP 400, this gives high confidence
	// that we got past network and authentication issues and looked at our request
	// it also should be somewhat inexpensive for the server
	var axiErr axiom.HTTPError
	var msg = "Did not receive expected error"
	_, err = d.client.Query(ctx, "")
	if err != nil && errors.As(err, &axiErr) {
		if axiErr.Status == 400 {
			// expected 400 for empty query, HEALTHY
			return &backend.CheckHealthResult{
				Status:  backend.HealthStatusOk,
				Message: "Data source is working",
			}, nil
		}
	}

	if err != nil {
		logger.Error("Failed to query Axiom", "error", err)
		msg = "Failed to query Axiom"
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusError,
		Message: msg,
	}, nil
}

func checkString(i any) string {
	if str, ok := i.(string); ok {
		return str
	}
	return ""
}
