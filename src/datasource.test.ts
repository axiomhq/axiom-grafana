import { DataQueryResponse, FieldType } from '@grafana/data';
import { buildMplVariableQuery } from './mplVariableQuery';
import { getMetricFindValues } from './variableValues';

describe('getMetricFindValues', () => {
  it('returns unique values from the first APL field', () => {
    const response: DataQueryResponse = {
      data: [
        {
          fields: [
            {
              name: 'service',
              type: FieldType.string,
              config: {},
              values: ['api', 'worker', 'api', null],
            },
          ],
          length: 4,
        },
      ],
    };

    expect(getMetricFindValues(response, 'apl')).toEqual([{ text: 'api' }, { text: 'worker' }]);
  });

  it('returns unique values from MPL series labels', () => {
    const response: DataQueryResponse = {
      data: [
        {
          fields: [
            { name: 'Time', type: FieldType.time, config: {}, values: [] },
            {
              name: 'http.requests',
              type: FieldType.number,
              config: {},
              values: [1],
              labels: { service: 'api' },
            },
          ],
          length: 1,
        },
        {
          fields: [
            { name: 'Time', type: FieldType.time, config: {}, values: [] },
            {
              name: 'http.requests',
              type: FieldType.number,
              config: {},
              values: [2],
              labels: { service: 'worker' },
            },
          ],
          length: 1,
        },
      ],
    };

    expect(getMetricFindValues(response, 'mpl')).toEqual([{ text: 'api' }, { text: 'worker' }]);
  });

  it('falls back to MPL non-time field values when labels are absent', () => {
    const response: DataQueryResponse = {
      data: [
        {
          fields: [
            { name: 'Time', type: FieldType.time, config: {}, values: [] },
            { name: 'value', type: FieldType.number, config: {}, values: [1, 2] },
          ],
          length: 2,
        },
      ],
    };

    expect(getMetricFindValues(response, 'mpl')).toEqual([{ text: '1' }, { text: '2' }]);
  });
});

describe('buildMplVariableQuery', () => {
  it('builds an MPL metric source from dataset and metric', () => {
    expect(buildMplVariableQuery({ dataset: 'metrics', metric: 'http_requests' })).toBe('metrics:http_requests');
  });

  it('groups by the selected tag to expose label values', () => {
    expect(buildMplVariableQuery({ dataset: 'metrics', metric: 'http_requests', tag: 'service_name' })).toBe(
      'metrics:http_requests | group by service_name using sum'
    );
  });

  it('escapes MPL identifiers that include punctuation', () => {
    expect(buildMplVariableQuery({ dataset: 'team/prod', metric: 'http.requests/total', tag: 'service.name' })).toBe(
      '`team/prod`:`http.requests/total` | group by `service.name` using sum'
    );
  });
});
