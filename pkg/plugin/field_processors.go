package plugin

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"

	axiQuery "github.com/axiomhq/axiom-go/axiom/query"
)

// FieldProcessor processes Axiom columns into Grafana data fields
type FieldProcessor interface {
	ProcessColumn(col []any) ([]*data.Field, error)
}

// Field type detection functions
func isHistogramField(field axiQuery.Field) bool {
	// Check if field has Aggregation metadata indicating histogram
	return field.Aggregation != nil && field.Aggregation.Op == axiQuery.OpHistogram
}

func isTopkField(field axiQuery.Field) bool {
	return field.Aggregation != nil && field.Aggregation.Op == axiQuery.OpTopk
}

// NewFieldProcessor creates the appropriate processor for the given field
func NewFieldProcessor(logger log.Logger, fieldDef axiQuery.Field) FieldProcessor {
	switch {
	case isHistogramField(fieldDef):
		return &HistogramFieldProcessor{fieldDef: fieldDef}
	case isTopkField(fieldDef):
		return &TopkFieldProcessor{fieldDef: fieldDef}
	default:
	return &RegularFieldProcessor{logger: logger, fieldDef: fieldDef}
	}
}

// RegularFieldProcessor handles standard field types (string, int, float, etc.)
type RegularFieldProcessor struct {
	logger   log.Logger
	fieldDef axiQuery.Field
}

// ProcessColumn processes a regular field column
func (p *RegularFieldProcessor) ProcessColumn(col []any) ([]*data.Field, error) {
	// Create field based on type
	var field *data.Field
	switch p.fieldDef.Type {
	case "datetime":
		field = data.NewField(p.fieldDef.Name, nil, []time.Time{})
	case "integer":
		field = data.NewField(p.fieldDef.Name, nil, []*float64{})
	case "float":
		field = data.NewField(p.fieldDef.Name, nil, []*float64{})
	case "bool":
		field = data.NewField(p.fieldDef.Name, nil, []*bool{})
	case "timespan":
		field = data.NewField(p.fieldDef.Name, nil, []*string{})
	case "unknown":
		field = data.NewField(p.fieldDef.Name, nil, []*string{})
	case "array":
		if len(col) > 0 && col[0] != nil {
			switch col[0].(type) {
			case []float64:
				field = data.NewField(p.fieldDef.Name, nil, [][]float64{})
			default:
				field = data.NewField(p.fieldDef.Name, nil, [][]*string{})
			}
		} else {
			field = data.NewField(p.fieldDef.Name, nil, [][]*string{})
		}
	default:
		field = data.NewField(p.fieldDef.Name, nil, []*string{})
	}

	// Populate field data
	for _, val := range col {
		if val == nil {
			field.Append(nil)
			continue
		}

		switch p.fieldDef.Type {
		case "datetime":
			t, err := time.Parse(time.RFC3339, val.(string))
			if err != nil {
				p.logger.Error("failed to parse datetime", "error", err)
				field.Append(nil)
			} else {
				field.Append(t)
			}
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

	return []*data.Field{field}, nil
}

// HistogramFieldProcessor handles histogram aggregation fields
type HistogramFieldProcessor struct {
	fieldDef axiQuery.Field
}

// ProcessColumn processes a histogram field column
func (p *HistogramFieldProcessor) ProcessColumn(col []any) ([]*data.Field, error) {
	// Collect all unique boundaries
	boundarySet := make(map[float64]bool)

	for _, cellValue := range col {
		if cellValue != nil {
			if histArray, ok := cellValue.([]any); ok {
				for _, bucketValue := range histArray {
					if bucketMap, ok := bucketValue.(map[string]any); ok {
						if to, exists := bucketMap["to"]; exists {
							boundarySet[to.(float64)] = true
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
	for _, cellValue := range col {
		bucketCounts := make(map[float64]float64)

		if cellValue != nil {
			if histArray, ok := cellValue.([]any); ok {
				for _, bucketValue := range histArray {
					if bucketMap, ok := bucketValue.(map[string]any); ok {
						if to, exists := bucketMap["to"]; exists {
							if count, exists := bucketMap["count"]; exists {
								bucketCounts[to.(float64)] = count.(float64)
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
				bucketFields[i].Append((*float64)(nil))
			}
		}
	}

	return bucketFields, nil
}

// TopkFieldProcessor handles topk aggregation fields
type TopkFieldProcessor struct {
	fieldDef axiQuery.Field
}

// ProcessColumn processes a topk field column
func (p *TopkFieldProcessor) ProcessColumn(col []any) ([]*data.Field, error) {
	// Create separate fields for key, count, and error
	keyField := data.NewField("key", nil, []*string{})
	countField := data.NewField("count", nil, []*float64{})
	errorField := data.NewField("error", nil, []*float64{})

	for _, cellValue := range col {
		if cellValue == nil {
			// Append nil values to maintain row alignment with other fields
			keyField.Append((*string)(nil))
			countField.Append((*float64)(nil))
			errorField.Append((*float64)(nil))
			continue
		}

		topkArray, ok := cellValue.([]any)
		if !ok {
			return nil, fmt.Errorf("expected topk array but got %T", cellValue)
		}

				// Process each topk entry
				for _, entry := range topkArray {
			entryMap, ok := entry.(map[string]any)
			if !ok {
				// Append nil values for malformed entries to maintain structure
				keyField.Append((*string)(nil))
				countField.Append((*float64)(nil))
				errorField.Append((*float64)(nil))
				continue
			}

						// Extract key
						if key, exists := entryMap["key"]; exists {
							if keyStr, ok := key.(string); ok {
								keyField.Append(&keyStr)
							} else {
								keyStr := fmt.Sprintf("%v", key)
								keyField.Append(&keyStr)
							}
						} else {
							keyField.Append((*string)(nil))
						}

						// Extract count
						if count, exists := entryMap["count"]; exists {
							if countFloat, ok := count.(float64); ok {
								countField.Append(&countFloat)
							} else {
								countField.Append((*float64)(nil))
							}
						} else {
							countField.Append((*float64)(nil))
						}

						// Extract error
						if errVal, exists := entryMap["error"]; exists {
							if errFloat, ok := errVal.(float64); ok {
								errorField.Append(&errFloat)
							} else {
								errorField.Append((*float64)(nil))
							}
						} else {
							errorField.Append((*float64)(nil))
						}
					}
				}

	return []*data.Field{keyField, countField, errorField}, nil
}
			} else {
				// Single row with nil values
				keyField.Append((*string)(nil))
				countField.Append((*float64)(nil))
				errorField.Append((*float64)(nil))
			}
		} else {
			// Single row with nil values
			keyField.Append((*string)(nil))
			countField.Append((*float64)(nil))
			errorField.Append((*float64)(nil))
		}
	}

	return []*data.Field{keyField, countField, errorField}, nil
}

