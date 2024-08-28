import { DataFrame, DataQueryRequest, DataQueryResponse, DataSourceInstanceSettings, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';

import { AxiomQuery, AxiomDataSourceOptions } from './types';

export class DataSource extends DataSourceWithBackend<AxiomQuery, AxiomDataSourceOptions> {
  url?: string;

  constructor(instanceSettings: DataSourceInstanceSettings<AxiomDataSourceOptions>) {
    super(instanceSettings);
    this.url = instanceSettings.url;
  }

  applyTemplateVariables(query: AxiomQuery, scopedVars: ScopedVars) {
    const templateSrv = getTemplateSrv();

    return {
      ...query,
      apl: query.apl ? templateSrv.replace(query.apl, scopedVars) : '',
    };
  }

  async metricFindQuery(query: AxiomQuery, options?: any) {
    const request = {
      targets: [
        {
          ...query,
          refId: 'metricFindQuery',
        },
      ],
      range: options.range,
      rangeRaw: options.rangeRaw,
    } as DataQueryRequest<AxiomQuery>;

    let res: DataQueryResponse | undefined;

    try {
      res = await this.query(request).toPromise();
    } catch (err) {
      return Promise.reject(err);
    }

    if (res && (!res.data.length || !res.data[0].fields.length)) {
      return [];
    }

    return res
      ? (res.data[0] as DataFrame).fields[0].values.map((v) => ({ text: v != null ? v.toString() : null }))
      : [];
  }

  async lookupSchema() {
    return this.getResource('/schema-lookup');
  }

  getQueryDisplayText(query: AxiomQuery) {
    return query.apl;
  }
}
