import { CoreApp, DataFrame, DataQueryRequest, DataQueryResponse, DataSourceInstanceSettings, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';

import { AxiomQuery, AxiomDataSourceOptions } from './types';
import { lastValueFrom } from 'rxjs';

const SUPPLEMENTARY_QUERY_TYPE_LOGS_VOLUME = 'LogsVolume';
const LOGS_VOLUME_QUERY_TYPE = 'logs-volume';

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
      apl: query.query ? templateSrv.replace(query.query, scopedVars) : '',
    };
  }

  query(request: DataQueryRequest<AxiomQuery>) {
    const includeTotalsTableFrame = request.app === CoreApp.Explore;

    return super.query({
      ...request,
      targets: request.targets.map((query) => ({
        ...query,
        includeTotalsTableFrame: includeTotalsTableFrame && query.kind !== 'mpl' && !query.totals,
      })),
    });
  }

  private timeRangeParams(): Record<string, string> {
    const from = getTemplateSrv().replace('$__from');
    const to = getTemplateSrv().replace('$__to');
    return {
      start: new Date(parseInt(from, 10)).toISOString(),
      end: new Date(parseInt(to, 10)).toISOString(),
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
      res = await lastValueFrom(this.query(request));
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
    return query.query;
  }

  getSupportedSupplementaryQueryTypes() {
    return [SUPPLEMENTARY_QUERY_TYPE_LOGS_VOLUME];
  }

  getSupplementaryQuery(options: { type: string }, originalQuery: AxiomQuery): AxiomQuery | undefined {
    if (options.type !== SUPPLEMENTARY_QUERY_TYPE_LOGS_VOLUME || originalQuery.hide || originalQuery.kind === 'mpl') {
      return undefined;
    }

    return {
      ...originalQuery,
      refId: `log-volume-${originalQuery.refId}`,
      queryType: LOGS_VOLUME_QUERY_TYPE,
      supportingQueryType: SUPPLEMENTARY_QUERY_TYPE_LOGS_VOLUME,
      totals: false,
    };
  }

  getSupplementaryRequest(type: string, request: DataQueryRequest<AxiomQuery>, options?: { type: string }) {
    if (type !== SUPPLEMENTARY_QUERY_TYPE_LOGS_VOLUME) {
      return undefined;
    }

    const targets = request.targets
      .map((query) => this.getSupplementaryQuery(options || { type }, query))
      .filter((query): query is AxiomQuery => Boolean(query));

    if (!targets.length) {
      return undefined;
    }

    return {
      ...request,
      targets,
    };
  }

  // metrics
  async getMetricsDatasets() {
    return this.getResource('metricsdatasets');
  }

  async getMetrics(dataset: string) {
    const timeParams = this.timeRangeParams();
    const params = new URLSearchParams();
    if (timeParams.start) {
      params.set('start', timeParams.start);
    }
    if (timeParams.end) {
      params.set('end', timeParams.end);
    }

    return this.getResource(`datasets/${encodeURIComponent(dataset)}/metrics?${params}`);
  }

  getTags(dataset: string, metric: string) {
    const timeParams = this.timeRangeParams();
    const params = new URLSearchParams();
    if (timeParams.start) {
      params.set('start', timeParams.start);
    }
    if (timeParams.end) {
      params.set('end', timeParams.end);
    }

    return this.getResource(
      `datasets/${encodeURIComponent(dataset)}/metrics/${encodeURIComponent(metric)}/tags?${params}`
    );
  }
}
