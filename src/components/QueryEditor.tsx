import React, { FormEvent } from 'react';
import { FieldSet, Field, InlineField, InlineFieldRow, InlineSwitch, FilterPill, Stack } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from '../datasource';
import { AxiomDataSourceOptions, AxiomQuery } from '../types';
import { MplQueryCodeMirror } from './MplQueryCodeMirror';
import { APLQueryEdtior } from './AplQueryEditor';

type Props = QueryEditorProps<DataSource, AxiomQuery, AxiomDataSourceOptions>;

export function QueryEditor({ query, onChange, onRunQuery, datasource }: Props) {
  // We need to use a ref for the totals because the function that is called on
  // Cmd/Ctrl+Enter only has access to a reference of the first render because
  // it runs when Monaco is initialized.
  const totals = React.useRef(query.totals);

  // const [queryStr, setQueryStr] = React.useState(!query.query ? query.apl : query.query);
  // if query.query is still empty, fallback to the deprecated .apl value
  const queryText = !query.query ? query.apl : query.query;
  console.log({ stored: query.apl, query: queryText });

  const onTotalsChange = (e: FormEvent<HTMLInputElement>) => {
    totals.current = e.currentTarget.checked;
    onChange({
      ...query,
      totals: e.currentTarget.checked,
    });
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
              onBlur={() => {}}
              onRunQuery={() => {
                onChange({
                  ...query,
                  kind: 'mpl',
                  query: queryText,
                });
                onRunQuery();
              }}
              onChange={(mpl) => {
                onChange({ ...query, query: mpl });
              }}
              datasource={datasource}
            />
          ) : (
            <APLQueryEdtior
              onChange={(apl) => {
                console.log('onBlur: changing apl', apl);
                onChange({ ...query, query: apl });
              }}
              value={queryText}
              totals={totals.current}
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
