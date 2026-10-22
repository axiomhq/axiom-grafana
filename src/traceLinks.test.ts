import { createDataFrame, DataSourceInstanceSettings, FieldType } from '@grafana/data';
import { addTraceLinks } from './traceLinks';
import { AxiomDataSourceOptions } from './types';

const settings = { uid: 'axiom', name: 'Axiom' } as DataSourceInstanceSettings<AxiomDataSourceOptions>;

it('links trace IDs in log labels to a trace lookup with dataset context', () => {
  const frame = createDataFrame({
    meta: { preferredVisualisationType: 'logs' },
    fields: [
      { name: 'labels', type: FieldType.other, values: [{ trace_id: 'abc' }, {}, '{"trace_id":"def"}', 'invalid'] },
    ],
  });
  const result = addTraceLinks(frame, settings, 'spans', 'trace_id');
  const field = result.fields.find((item) => item.name === 'traceID')!;
  expect(field.values).toEqual(['abc', null, 'def', null]);
  expect(field.config.links?.[0].internal).toMatchObject({
    datasourceUid: 'axiom',
    query: {
      kind: 'trace',
      dataset: 'spans',
      traceIdField: 'trace_id',
      traceId: '${__value.raw}',
    },
  });
  expect(frame.fields).toHaveLength(1);
});

it('does not add links without a configured dataset or to trace frames', () => {
  const frame = createDataFrame({ meta: { preferredVisualisationType: 'trace' }, fields: [] });
  expect(addTraceLinks(frame, settings, 'spans', 'trace_id')).toBe(frame);
  expect(addTraceLinks(frame, settings, '', 'trace_id')).toBe(frame);
});

it('preserves existing links on a custom trace ID column', () => {
  const frame = createDataFrame({
    meta: { preferredVisualisationType: 'logs' },
    fields: [
      {
        name: 'custom_id',
        type: FieldType.string,
        values: ['abc'],
        config: { links: [{ title: 'Existing', url: 'https://example.com' }] },
      },
    ],
  });
  const result = addTraceLinks(frame, settings, 'spans', 'custom_id');
  expect(result.fields).toHaveLength(1);
  expect(result.fields[0].config.links).toHaveLength(2);
  expect(result.fields[0].config.links?.[1].internal?.query).toMatchObject({ traceIdField: 'custom_id' });
  expect(addTraceLinks(frame, settings, '', 'custom_id')).toBe(frame);
});
