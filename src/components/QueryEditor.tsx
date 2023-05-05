import React, { FormEvent } from 'react';
import { FieldSet, InlineFieldRow, Field, InlineField, InlineSwitch, CodeEditor } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from '../datasource';
import { AxiomDataSourceOptions, AxiomQuery } from '../types';

type Props = QueryEditorProps<DataSource, AxiomQuery, AxiomDataSourceOptions>;

export function QueryEditor({ query, onChange, onRunQuery }: Props) {
  const onQueryTextChange = (apl: string) => {
    onChange({ ...query, apl });
  };

  const onTotalsChange = (e: FormEvent<HTMLInputElement>) => {
        onChange({
            ...query,
            totals: e.currentTarget.checked,
        });
    };

  const { apl: queryText } = query;



  return (
    <FieldSet>
        <Field>
          <CodeEditor
            onBlur={onQueryTextChange}
            height="140px"
            width="500"
            value={queryText || ''}
            language="kusto"
            showLineNumbers={true}
            showMiniMap={false}
          />
        </Field>
        <InlineFieldRow>
            <InlineField label="Query type" grow>
                <InlineSwitch
                    label="Return Totals Table"
                    showLabel={true}
                    defaultChecked={query.totals}
                    value={query.totals}
                    onChange={onTotalsChange}
                />
            </InlineField>
        </InlineFieldRow>
    </FieldSet>
  );
}
