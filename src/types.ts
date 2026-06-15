import { DataQuery, DataSourceJsonData } from '@grafana/schema';

export const QUERY_MODEL_VERSION = '2.0';

export type QueryModelVersion = typeof QUERY_MODEL_VERSION;
export type AxiomQueryKind = 'apl' | 'mpl';

export interface AxiomQuery extends DataQuery {
  version?: QueryModelVersion;
  kind?: AxiomQueryKind | null;
  query: string;
  apl?: string;
  totals: boolean;
  dataset?: string;
  metric?: string;
  tag?: string;
  includeTotalsTableFrame?: boolean;
  includeLogsVolumeFrame?: boolean;
  supportingQueryType?: 'LogsVolume';
  startTime?: string;
  endTime?: string;
}

export const DEFAULT_QUERY: Partial<AxiomQuery> = {
  version: QUERY_MODEL_VERSION,
  kind: 'apl',
  query: '',
  totals: false,
};

/**
 * These are options configured for each DataSource instance
 */
export interface AxiomDataSourceOptions extends DataSourceJsonData {
  apiHost: string;
  orgID: string;
  /**
   * Optional regional edge domain for ingest and query operations
   * (e.g., "eu-central-1.aws.edge.axiom.co").
   * When set, queries are routed to https://{edge}/v1/query/_apl.
   * All other API calls (schema lookup, health checks) continue to use apiHost.
   */
  edge?: string;
  /**
   * Optional explicit edge URL for ingest and query operations
   * (e.g., "https://custom-edge.example.com").
   * If a path is provided, the URL is used as-is.
   * If no path is provided, /v1/query/_apl is appended.
   * Takes precedence over edge if both are set.
   */
  edgeURL?: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface MySecureJsonData {
  accessToken: string;
}
