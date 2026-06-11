package plugin

import (
	"context"
	"fmt"
	"time"

	axiQuery "github.com/axiomhq/axiom-go/axiom/query"
	"github.com/axiomhq/axiom-grafana/pkg/axiomapi"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type aplFrameOptions struct {
	FieldMetaByName map[string]axiomapi.APLFieldMetaMap
	Status          *axiomapi.APLQueryStatus
	Query           string
}

type aplFrameBuilder interface {
	Build(context.Context, *axiQuery.Table, aplFrameOptions) (*data.Frame, error)
}

func buildAPLFrame(ctx context.Context, result *axiQuery.Table, opts ...aplFrameOptions) (*data.Frame, error) {
	frameOptions := aplFrameOptions{}
	if len(opts) > 0 {
		frameOptions = opts[0]
	}

	builder := newAPLFrameBuilder(ctx, result.Fields)
	return builder.Build(ctx, result, frameOptions)
}

func newAPLFrameBuilder(ctx context.Context, fields []axiQuery.Field) aplFrameBuilder {
	if fieldsMatchTrace(ctx, fields) {
		return traceFrameBuilder{}
	}

	return aplTableFrameBuilder{}
}

type aplTableFrameBuilder struct{}

func (aplTableFrameBuilder) Build(ctx context.Context, result *axiQuery.Table, opts aplFrameOptions) (*data.Frame, error) {
	logger := log.DefaultLogger.FromContext(ctx)
	frame := data.NewFrame("response")

	fields := make([]*data.Field, 0, len(result.Fields))
	fieldTypes := make([]string, 0, len(result.Fields))

	for i, f := range result.Fields {
		f := f
		i := i
		fieldType := f.Type
		var sampleValue any
		sampleRow := -1
		columnLen := 0
		if i < len(result.Columns) {
			columnLen = len(result.Columns[i])
			sampleValue, sampleRow, _ = firstDebugValue(result.Columns[i])
		}
		if f.Name == "_time" {
			fieldType = "datetime"
		} else if fieldType == "unknown" && i < len(result.Columns) {
			fieldType = inferUnknownFieldType(f.Name, result.Columns[i])
			logger.Debug("inferred unknown APL field type", "field", f.Name, "type", fieldType)
		}

		var field *data.Field
		func() {
			var fieldValues any
			defer func() {
				if r := recover(); r != nil {
					logger.Error(
						"panic creating APL data frame field",
						"field", f.Name,
						"fieldIndex", i,
						"declaredType", f.Type,
						"resolvedType", fieldType,
						"columnLength", columnLen,
						"sampleRow", sampleRow,
						"sampleValueType", debugValueType(sampleValue),
						"sampleValue", debugValuePreview(sampleValue),
						"fieldValuesType", debugValueType(fieldValues),
						"panic", fmt.Sprintf("%v", r),
					)
					panic(r)
				}
			}()

			switch fieldType {
			case "datetime":
				fieldValues = []time.Time{}
			case "integer":
				fieldValues = []*float64{}
			case "float":
				fieldValues = []*float64{}
			case "bool":
				fieldValues = []*bool{}
			case "timespan":
				fieldValues = []*string{}
			case "array":
				fieldValues = []*string{}
			default:
				fieldValues = []*string{}
			}

			field = data.NewField(f.Name, nil, fieldValues)
		}()
		applyAPLFieldMetadata(field, f, opts.FieldMetaByName)

		fields = append(fields, field)
		fieldTypes = append(fieldTypes, fieldType)
	}

	for colIndex, col := range result.Columns {
		if colIndex >= len(result.Fields) || colIndex >= len(fields) {
			return nil, fmt.Errorf("table column %d has no matching field metadata", colIndex)
		}

		for i := 0; i < len(col); i++ {
			colIndex := colIndex
			i := i

			func() {
				defer func() {
					if r := recover(); r != nil {
						logger.Error(
							"panic appending APL data frame value",
							"field", result.Fields[colIndex].Name,
							"fieldIndex", colIndex,
							"rowIndex", i,
							"declaredType", result.Fields[colIndex].Type,
							"resolvedType", fieldTypes[colIndex],
							"valueType", debugValueType(col[i]),
							"value", debugValuePreview(col[i]),
							"panic", fmt.Sprintf("%v", r),
						)
						panic(r)
					}
				}()

				if col[i] == nil {
					fields[colIndex].Append(nil)
					return
				}

				switch fieldTypes[colIndex] {
				case "datetime":
					t, err := time.Parse(time.RFC3339, col[i].(string))
					if err != nil {
						logger.Warn("Failed to parse time", "time", col[i])
						fields[colIndex].Append(time.Time{})
						return
					}
					fields[colIndex].Append(t)
				case "integer":
					num := col[i].(float64)
					fields[colIndex].Append(&num)
				case "float":
					num := col[i].(float64)
					fields[colIndex].Append(&num)
				case "string", "unknown":
					txt, ok := col[i].(string)
					if !ok {
						txt = stringifyFrameValue(col[i])
					}
					fields[colIndex].Append(&txt)
				case "bool":
					b := col[i].(bool)
					fields[colIndex].Append(&b)
				case "timespan":
					num := col[i].(string)
					fields[colIndex].Append(&num)
				case "array":
					txt := stringifyFrameValue(col[i])
					fields[colIndex].Append(&txt)
				default:
					txt := stringifyFrameValue(col[i])
					fields[colIndex].Append(&txt)
				}
			}()
		}
	}
	frame.Fields = fields
	applyAPLFrameMetadata(frame, opts)

	return frame, nil
}
