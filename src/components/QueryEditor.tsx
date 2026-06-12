import React, { FormEvent } from 'react';
import { FieldSet, Field, InlineField, InlineFieldRow, InlineSwitch, FilterPill, Stack } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from '../datasource';
import { AxiomDataSourceOptions, AxiomQuery } from '../types';
import { MplQueryCodeMirror } from './MplQueryCodeMirror';
import { APLQueryEdtior } from './AplQueryEditor';

type Props = QueryEditorProps<DataSource, AxiomQuery, AxiomDataSourceOptions>;

export function QueryEditor({ query, onChange, onRunQuery, datasource }: Props) {
  // const [queryStr, setQueryStr] = React.useState(!query.query ? query.apl : query.query);
  // if query.query is still empty, fallback to the deprecated .apl value
  const queryText = !query.query ? query.apl : query.query;

  const onTotalsChange = (e: FormEvent<HTMLInputElement>) => {
    onChange({
      ...query,
      totals: e.currentTarget.checked,
    });
  };

  const runMplQuery = (mpl: string) => {
    onChange({
      ...query,
      kind: 'mpl',
      query: mpl,
    });
    onRunQuery();
  };

  return (
    <Stack direction={'column'}>
      <Stack>
        <FilterPill
          label="APL"
          onClick={() => onChange({ ...query, kind: 'apl' })}
          selected={query.kind === 'apl' || !query.kind}
        />
        <FilterPill label="MPL" onClick={() => onChange({ ...query, kind: 'mpl' })} selected={query.kind === 'mpl'} />
      </Stack>
      <FieldSet>
        <Field>
          {query.kind === 'mpl' ? (
            <MplQueryCodeMirror
              value={queryText}
              onBlur={runMplQuery}
              onRunQuery={runMplQuery}
              onChange={(mpl) => {
                onChange({ ...query, query: mpl });
              }}
              datasource={datasource}
            />
          ) : (
            <APLQueryEdtior
              onChange={(apl) => {
                onChange({ ...query, query: apl });
              }}
              value={queryText}
              datasource={datasource}
              onRunQuery={onRunQuery}
            />
          )}
        </Field>
        {query.kind !== 'mpl' && (
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
        )}
      </FieldSet>
    </Stack>
  );
}
