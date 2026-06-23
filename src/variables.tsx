import { CustomVariableSupport, DataQueryRequest } from '@grafana/data';
import { from, map, of } from 'rxjs';

import { VariableQueryEditor } from './components/VariableQueryEditor';
import type { DataSource } from './datasource';
import { AxiomDataSourceOptions, AxiomQuery, DEFAULT_QUERY } from './types';
import { migrateAxiomQuery } from './queryMigration';
import { getMetricFindValues, metricFindValuesToDataQueryResponse, textValuesToMetricFindValues } from './variableValues';

export class AxiomVariableSupport extends CustomVariableSupport<
  DataSource,
  AxiomQuery,
  AxiomQuery,
  AxiomDataSourceOptions
> {
  editor = VariableQueryEditor;

  constructor(private readonly datasource: DataSource) {
    super();
  }

  getDefaultQuery(): Partial<AxiomQuery> {
    return DEFAULT_QUERY;
  }

  query(request: DataQueryRequest<AxiomQuery>) {
    const query = request.targets[0] ? migrateAxiomQuery(request.targets[0]) : undefined;
    if (query?.kind === 'mpl') {
      if (!query.dataset || !query.tag) {
        return of(metricFindValuesToDataQueryResponse([]));
      }

      return from(this.datasource.getMetricTagValues(query.dataset, query.tag, query.metric)).pipe(
        map((values: string[]) => metricFindValuesToDataQueryResponse(textValuesToMetricFindValues(values)))
      );
    }

    return this.datasource
      .query({
        ...request,
        targets: request.targets.map((target) => migrateAxiomQuery(target)),
      })
      .pipe(map((res) => metricFindValuesToDataQueryResponse(getMetricFindValues(res, query?.kind))));
  }
}
