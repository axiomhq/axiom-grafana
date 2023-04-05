import React, { ChangeEvent } from 'react';
import { InlineField, Input } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from '../datasource';
import { AxiomDataSourceOptions, AxiomQuery } from '../types';

type Props = QueryEditorProps<DataSource, AxiomQuery, AxiomDataSourceOptions>;

export function QueryEditor({ query, onChange, onRunQuery }: Props) {
  const onQueryTextChange = (event: ChangeEvent<HTMLInputElement>) => {
    onChange({ ...query, apl: event.target.value });
  };

  // const onConstantChange = (event: ChangeEvent<HTMLInputElement>) => {
  //   onChange({ ...query });
  //   // executes the query
  //   onRunQuery();
  // };

  const { apl: queryText } = query;

  return (
    <div className="gf-form">
      {/* <InlineField label="Constant">
        <Input onChange={onConstantChange} value={constant} width={8} type="number" step="0.1" />
      </InlineField> */}
      <InlineField label="APL" labelWidth={8} tooltip="Axiom Processing Language query">
        <Input onChange={onQueryTextChange} value={queryText || ''} />
      </InlineField>
    </div>
  );
}
