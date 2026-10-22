import { DataFrame, DataSourceInstanceSettings, FieldType } from '@grafana/data';
import { AxiomDataSourceOptions } from './types';

const traceAliases = ['traceID', 'traceId', 'trace_id', 'trace.id', 'traceid'];

export function addTraceLinks(
  frame: DataFrame,
  settings: DataSourceInstanceSettings<AxiomDataSourceOptions>,
  dataset: string,
  traceIdField: string
): DataFrame {
  if (!dataset || frame.meta?.preferredVisualisationType !== 'logs') {
    return frame;
  }
  const aliases = [traceIdField, ...traceAliases];
  const source = frame.fields.find((field) => aliases.includes(field.name));
  const labels = frame.fields.find((field) => field.name === 'labels');
  const values =
    source?.values ??
    Array.from({ length: frame.length }, (_, row) => {
      const raw = labels?.values[row];
      let parsed: Record<string, unknown>;
      try {
        parsed = typeof raw === 'string' ? JSON.parse(raw) : raw;
      } catch {
        return null;
      }
      for (const name of aliases) {
        const value = parsed?.[name];
        if (typeof value === 'string' && value) {
          return value;
        }
      }
      return null;
    });
  if (!values.some((value) => typeof value === 'string' && value.length > 0)) {
    return frame;
  }
  const field = {
    ...(source || { name: 'traceID', type: FieldType.string, values }),
    config: {
      ...source?.config,
      links: [
        ...(source?.config.links || []),
        {
          title: 'View trace',
          url: '',
          internal: {
            datasourceUid: settings.uid,
            datasourceName: settings.name,
            query: {
              refId: 'A',
              kind: 'trace',
              query: '',
              totals: false,
              dataset,
              traceIdField,
              traceId: '${__value.raw}',
            },
          },
        },
      ],
    },
  };
  return { ...frame, fields: [...frame.fields.filter((item) => item !== source), field] };
}
