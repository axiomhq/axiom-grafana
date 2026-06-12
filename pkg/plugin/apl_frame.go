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

type aplResponseFrameBuilder struct {
	totals bool
}

func newAPLResponseFrameBuilder(totals bool) aplResponseFrameBuilder {
	return aplResponseFrameBuilder{
		totals: totals,
	}
}

func (b aplResponseFrameBuilder) Build(ctx context.Context, result axiomapi.APLQueryResponse, opts aplFrameOptions) (*data.Frame, error) {
	if len(result.Tables) == 0 {
		return nil, fmt.Errorf("query returned no tables")
	}

	if b.shouldBuildTimeSeries(result) {
		return aplTimeSeriesFrameBuilder{}.Build(ctx, &result.Tables[0], opts)
	}

	table := &result.Tables[0]
	if b.totals && len(result.Tables) > 1 {
		table = &result.Tables[1]
	}

	return newAPLEventFrameBuilder(ctx, table.Fields).Build(ctx, table, opts)
}

func (b aplResponseFrameBuilder) shouldBuildTimeSeries(result axiomapi.APLQueryResponse) bool {
	return !b.totals && len(result.Tables) > 1
}

func buildAPLFrame(ctx context.Context, result *axiQuery.Table, opts ...aplFrameOptions) (*data.Frame, error) {
	frameOptions := aplFrameOptions{}
	if len(opts) > 0 {
		frameOptions = opts[0]
	}

	builder := newAPLEventFrameBuilder(ctx, result.Fields)
	return builder.Build(ctx, result, frameOptions)
}

func newAPLEventFrameBuilder(ctx context.Context, fields []axiQuery.Field) aplFrameBuilder {
	if fieldsMatchTrace(ctx, fields) {
		return aplTraceFrameBuilder{}
	}
	if fieldsMatchLogs(fields) {
		return aplLogsFrameBuilder{}
	}

	return aplEventsFrameBuilder{}
}

type aplTimeSeriesFrameBuilder struct{}

func (aplTimeSeriesFrameBuilder) Build(ctx context.Context, result *axiQuery.Table, opts aplFrameOptions) (*data.Frame, error) {
	logger := log.DefaultLogger.FromContext(ctx)

	frame, err := aplTableFrameBuilder{}.Build(ctx, result, opts)
	if err != nil {
		return nil, err
	}

	frame = prepareAPLTimeSeriesFrame(frame)
	wideFrame, err := aplWideFrameBuilder{}.Build(frame)
	if err != nil {
		logger.Error("transformation from long to wide failed", "error", err.Error())
		return frame, nil
	}

	applyPreferredVisualization(wideFrame, data.VisTypeGraph)
	return wideFrame, nil
}

func prepareAPLTimeSeriesFrame(frame *data.Frame) *data.Frame {
	primaryTimeIndex := preferredAPLTimeFieldIndex(frame.Fields)
	if primaryTimeIndex < 0 {
		return frame
	}

	fields := make([]*data.Field, 0, len(frame.Fields))
	fields = append(fields, frame.Fields[primaryTimeIndex])
	for i, field := range frame.Fields {
		if i == primaryTimeIndex {
			continue
		}
		if _, ok := aplDataFrameTimeFieldPriority(field); ok {
			continue
		}
		fields = append(fields, field)
	}

	if len(fields) == len(frame.Fields) && primaryTimeIndex == 0 {
		return frame
	}

	timeSeriesFrame := *frame
	timeSeriesFrame.Fields = fields
	return &timeSeriesFrame
}

type aplEventsFrameBuilder struct{}

func (aplEventsFrameBuilder) Build(ctx context.Context, result *axiQuery.Table, opts aplFrameOptions) (*data.Frame, error) {
	frame, err := aplTableFrameBuilder{}.Build(ctx, result, opts)
	if err != nil {
		return nil, err
	}

	applyPreferredVisualization(frame, data.VisTypeTable)
	return frame, nil
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
				fieldValues = []*time.Time{}
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

				switch fieldTypes[colIndex] {
				case "datetime":
					if col[i] == nil {
						fields[colIndex].Append(nil)
						return
					}

					timestamp, ok := col[i].(time.Time)
					if ok {
						fields[colIndex].Append(&timestamp)
						return
					}

					t, err := time.Parse(time.RFC3339Nano, col[i].(string))
					if err != nil {
						logger.Warn("Failed to parse time", "time", col[i])
						fields[colIndex].Append(nil)
						return
					}
					fields[colIndex].Append(&t)
				case "integer":
					if col[i] == nil {
						fields[colIndex].Append(nil)
						return
					}
					num := col[i].(float64)
					fields[colIndex].Append(&num)
				case "float":
					if col[i] == nil {
						fields[colIndex].Append(nil)
						return
					}
					num := col[i].(float64)
					fields[colIndex].Append(&num)
				case "string", "unknown":
					if col[i] == nil {
						fields[colIndex].Append(nil)
						return
					}
					txt, ok := col[i].(string)
					if !ok {
						txt = stringifyFrameValue(col[i])
					}
					fields[colIndex].Append(&txt)
				case "bool":
					if col[i] == nil {
						fields[colIndex].Append(nil)
						return
					}
					b := col[i].(bool)
					fields[colIndex].Append(&b)
				case "timespan":
					if col[i] == nil {
						fields[colIndex].Append(nil)
						return
					}
					num := col[i].(string)
					fields[colIndex].Append(&num)
				case "array":
					if col[i] == nil {
						fields[colIndex].Append(nil)
						return
					}
					txt := stringifyFrameValue(col[i])
					fields[colIndex].Append(&txt)
				default:
					if col[i] == nil {
						fields[colIndex].Append(nil)
						return
					}
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
