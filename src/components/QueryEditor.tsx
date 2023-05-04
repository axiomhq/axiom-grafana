import React from 'react';
import { CodeEditor } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from '../datasource';
import { AxiomDataSourceOptions, AxiomQuery } from '../types';

type Props = QueryEditorProps<DataSource, AxiomQuery, AxiomDataSourceOptions>;

export function QueryEditor({ query, onChange, onRunQuery }: Props) {
  const onQueryTextChange = (apl: string) => {
    onChange({ ...query, apl });
  };

  const { apl: queryText } = query;

  return (
    <div>
      <CodeEditor
        onBlur={onQueryTextChange}
        height="140px"
        width="500"
        value={queryText || ''}
        language="kusto"
        showLineNumbers={true}
        showMiniMap={false}
      />
    </div>
  );
}
