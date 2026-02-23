import { DataQuery, DataSourceJsonData } from '@grafana/schema';

export interface AxiomQuery extends DataQuery {
  apl: string;
  totals: boolean;
  startTime?: string;
  endTime?: string;
}

export const DEFAULT_QUERY: Partial<AxiomQuery> = {
  apl: '',
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
