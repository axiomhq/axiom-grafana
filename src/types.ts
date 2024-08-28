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
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface MySecureJsonData {
  accessToken: string;
}
