import { DataSourcePlugin } from '@grafana/data';
import { DataSource } from './datasource';
import { ConfigEditor } from './components/ConfigEditor';
import { QueryEditor } from './components/QueryEditor';
import { AxiomQuery, AxiomDataSourceOptions, MySecureJsonData } from './types';

export const plugin = new DataSourcePlugin<DataSource, AxiomQuery, AxiomDataSourceOptions, MySecureJsonData>(DataSource)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor);
