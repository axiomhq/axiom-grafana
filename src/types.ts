import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface AxiomQuery extends DataQuery {
  apl: string;
  startTime?: string;
  endTime?: string;
}

export const DEFAULT_QUERY: Partial<AxiomQuery> = {
  apl: '',
};

/**
 * These are options configured for each DataSource instance
 */
export interface AxiomDataSourceOptions extends DataSourceJsonData {
  apiHost: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface MySecureJsonData {
  accessToken: string;
}

export interface APLResponse {
  format: string;
  tables: APLTable[];
  datasetNames: string[];

}

export interface APLTable {
  name: string;
  buckets: any;
  fields: APLTableField[];
  columns: any[];
  groups: Array<{name: string}>
}

export interface APLTableField {
  name: string;
  type: string;
  agg: {name: string};
}
