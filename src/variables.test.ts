import { CoreApp, DataQueryRequest, DataQueryResponse, FieldType } from '@grafana/data';
import { lastValueFrom, of } from 'rxjs';

import type { DataSource } from './datasource';
import { AxiomQuery } from './types';
import { AxiomVariableSupport, VARIABLE_QUERY_REF_ID } from './variables';

describe('AxiomVariableSupport', () => {
  it('assigns a refId so Grafana can keep variable query frames', async () => {
    const seen: AxiomQuery[][] = [];
    const datasource = {
      query: (request: DataQueryRequest<AxiomQuery>) => {
        seen.push(request.targets);
        return of<DataQueryResponse>({
          data: [
            {
              fields: [{ name: 'environment', type: FieldType.string, config: {}, values: ['dev', 'prod'] }],
              length: 2,
            },
          ],
        });
      },
    } as unknown as DataSource;

    const support = new AxiomVariableSupport(datasource);
    const response = await lastValueFrom(
      support.query({
        app: CoreApp.Dashboard,
        targets: [
          {
            apl: "['deployments'] | distinct ['environment']",
            totals: true,
          } as AxiomQuery,
        ],
      } as DataQueryRequest<AxiomQuery>)
    );

    expect(seen[0][0].refId).toBe(VARIABLE_QUERY_REF_ID);
    expect(response.data[0].fields[0].values).toEqual(['dev', 'prod']);
  });

  it('preserves an existing refId', async () => {
    const seen: AxiomQuery[][] = [];
    const datasource = {
      query: (request: DataQueryRequest<AxiomQuery>) => {
        seen.push(request.targets);
        return of<DataQueryResponse>({ data: [] });
      },
    } as unknown as DataSource;

    const support = new AxiomVariableSupport(datasource);
    await lastValueFrom(
      support.query({
        targets: [
          {
            refId: 'A',
            query: "['deployments'] | distinct ['environment']",
            kind: 'apl',
            totals: true,
          } as AxiomQuery,
        ],
      } as DataQueryRequest<AxiomQuery>)
    );

    expect(seen[0][0].refId).toBe('A');
  });
});
