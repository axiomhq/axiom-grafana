import {
  CoreApp,
  DataQueryRequest,
  DataQueryResponse,
  DataSourceInstanceSettings,
  MetricFindValue,
  ScopedVars,
} from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';

import { AxiomQuery, AxiomDataSourceOptions } from './types';
import { AxiomVariableSupport } from './variables';
import { getMetricFindValues, textValuesToMetricFindValues } from './variableValues';
import { lastValueFrom } from 'rxjs';

const SUPPLEMENTARY_QUERY_TYPE_LOGS_VOLUME = 'LogsVolume';
const LOGS_VOLUME_QUERY_TYPE = 'logs-volume';

const PANEL_APPS = new Set<CoreApp | string>([CoreApp.Dashboard, CoreApp.PanelEditor, CoreApp.PanelViewer]);

export class DataSource extends DataSourceWithBackend<AxiomQuery, AxiomDataSourceOptions> {
  url?: string;

  constructor(instanceSettings: DataSourceInstanceSettings<AxiomDataSourceOptions>) {
    super(instanceSettings);
    this.url = instanceSettings.url;
    this.variables = new AxiomVariableSupport(this);
  }

  applyTemplateVariables(query: AxiomQuery, scopedVars: ScopedVars) {
    const templateSrv = getTemplateSrv();
    const queryText = query.query || query.apl || '';
    const interpolatedQuery = templateSrv.replace(queryText, scopedVars);

    return {
      ...query,
      query: interpolatedQuery,
      apl: interpolatedQuery,
    };
  }

  query(request: DataQueryRequest<AxiomQuery>) {
    const includeTotalsTableFrame = request.app === CoreApp.Explore;
    // Dashboard panels default to Time series, which needs a numeric frame.
    // When an APL query returns logs, the backend uses this flag to prepend a
    // logs-volume frame while preserving the raw log lines as a secondary frame.
    const includeLogsVolumeFrame = request.app ? PANEL_APPS.has(request.app) : false;

    return super.query({
      ...request,
      targets: request.targets.map((query) => ({
        ...query,
        includeTotalsTableFrame: includeTotalsTableFrame && query.kind !== 'mpl' && !query.totals,
        includeLogsVolumeFrame: includeLogsVolumeFrame && query.kind !== 'mpl' && !query.totals,
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

  async metricFindQuery(query: AxiomQuery, options?: any): Promise<MetricFindValue[]> {
    const scopedVars = options?.scopedVars ?? {};
    const interpolatedQuery = this.applyTemplateVariables(query, scopedVars);
    if (interpolatedQuery.kind === 'mpl') {
      if (!interpolatedQuery.dataset || !interpolatedQuery.tag) {
        return [];
      }

      return textValuesToMetricFindValues(
        await this.getMetricTagValues(interpolatedQuery.dataset, interpolatedQuery.tag, interpolatedQuery.metric)
      );
    }

    const request = {
      targets: [
        {
          ...interpolatedQuery,
          refId: 'metricFindQuery',
        },
      ],
      range: options?.range,
      rangeRaw: options?.rangeRaw,
    } as DataQueryRequest<AxiomQuery>;

    let res: DataQueryResponse | undefined;

    try {
      res = await lastValueFrom(this.query(request));
    } catch (err) {
      return Promise.reject(err);
    }

    if (res && (!res.data.length || !res.data.some((frame) => frame.fields?.length))) {
      return [];
    }

    return res ? getMetricFindValues(res, interpolatedQuery.kind) : [];
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

  getTags(dataset: string, metric?: string) {
    const timeParams = this.timeRangeParams();
    const params = new URLSearchParams();
    if (timeParams.start) {
      params.set('start', timeParams.start);
    }
    if (timeParams.end) {
      params.set('end', timeParams.end);
    }

    const encodedDataset = encodeURIComponent(dataset);
    if (!metric) {
      return this.getResource(`datasets/${encodedDataset}/tags?${params}`);
    }

    return this.getResource(`datasets/${encodedDataset}/metrics/${encodeURIComponent(metric)}/tags?${params}`);
  }

  getMetricTagValues(dataset: string, tag: string, metric?: string) {
    const timeParams = this.timeRangeParams();
    const params = new URLSearchParams();
    if (timeParams.start) {
      params.set('start', timeParams.start);
    }
    if (timeParams.end) {
      params.set('end', timeParams.end);
    }

    const encodedDataset = encodeURIComponent(dataset);
    const encodedTag = encodeURIComponent(tag);
    if (!metric) {
      return this.getResource(`datasets/${encodedDataset}/tags/${encodedTag}/values?${params}`);
    }

    return this.getResource(
      `datasets/${encodedDataset}/metrics/${encodeURIComponent(metric)}/tags/${encodedTag}/values?${params}`
    );
  }
}
