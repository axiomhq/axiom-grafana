import { DataFrame, DataQueryResponse, FieldType, MetricFindValue, toDataFrame } from '@grafana/data';

import { AxiomQuery } from './types';

export function getMetricFindValues(res: DataQueryResponse, kind: AxiomQuery['kind']): MetricFindValue[] {
  const values = kind === 'mpl' ? getMplMetricFindValues(res) : getAplMetricFindValues(res);
  return textValuesToMetricFindValues(values);
}

export function textValuesToMetricFindValues(values: unknown[]): MetricFindValue[] {
  const seen = new Set<string>();

  return values.reduce<MetricFindValue[]>((acc, value) => {
    if (value == null) {
      return acc;
    }

    const text = value.toString();
    if (!seen.has(text)) {
      seen.add(text);
      acc.push({ text });
    }

    return acc;
  }, []);
}

export function metricFindValuesToDataQueryResponse(values: MetricFindValue[]): DataQueryResponse {
  return {
    data: [
      toDataFrame({
        fields: [
          {
            name: 'text',
            type: FieldType.string,
            values: values.map((value) => value.text),
          },
        ],
      }),
    ],
  };
}

function getAplMetricFindValues(res: DataQueryResponse): unknown[] {
  const frame = getDataFrames(res)[0];
  return frame?.fields[0]?.values ?? [];
}

function getMplMetricFindValues(res: DataQueryResponse): unknown[] {
  const frames = getDataFrames(res);
  const labelValues = frames.flatMap((frame) => frame.fields.flatMap((field) => Object.values(field.labels ?? {})));

  if (labelValues.length) {
    return labelValues;
  }

  return frames.flatMap((frame) => frame.fields.find((field) => field.type !== FieldType.time)?.values ?? []);
}

function getDataFrames(res: DataQueryResponse): DataFrame[] {
  return res.data.filter((frame): frame is DataFrame => Array.isArray((frame as DataFrame).fields));
}
