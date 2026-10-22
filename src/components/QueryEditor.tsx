import React, { FormEvent, useEffect } from 'react';
import { FieldSet, Field, InlineField, InlineFieldRow, InlineSwitch, FilterPill, Stack, Input } from '@grafana/ui';
import { CoreApp, QueryEditorProps } from '@grafana/data';
import type { DataSource } from '../datasource';
import { AxiomDataSourceOptions, AxiomQuery } from '../types';
import { migrateAxiomQuery, shouldMigrateAxiomQuery } from '../queryMigration';
import { MplQueryCodeMirror } from './MplQueryCodeMirror';
import { APLQueryEdtior } from './AplQueryEditor';

type Props = QueryEditorProps<DataSource, AxiomQuery, AxiomDataSourceOptions>;

function hasRunnableQuery(value: string) {
  return value.split('\n').some((line) => {
    const trimmed = line.trim();
    return trimmed !== '' && !trimmed.startsWith('//');
  });
}

export function QueryEditor({ query, onChange, onRunQuery, datasource, app }: Props) {
  const migratedQuery = migrateAxiomQuery(query);
  const queryText = migratedQuery.query;
  const autoFocusEditor = app === CoreApp.Explore;

  useEffect(() => {
    if (shouldMigrateAxiomQuery(query)) {
      onChange(migrateAxiomQuery(query));
    }
  }, [query, onChange]);

  const onTotalsChange = (e: FormEvent<HTMLInputElement>) => {
    onChange({
      ...migratedQuery,
      totals: e.currentTarget.checked,
    });
  };

  const runMplQuery = (mpl: string) => {
    onChange({
      ...migratedQuery,
      kind: 'mpl',
      query: mpl,
    });
    if (hasRunnableQuery(mpl)) {
      onRunQuery();
    }
  };

  return (
    <Stack direction={'column'}>
      <Stack>
        <FilterPill
          label="APL"
          onClick={() => onChange({ ...migratedQuery, kind: 'apl' })}
          selected={migratedQuery.kind === 'apl'}
        />
        <FilterPill
          label="MPL"
          onClick={() => onChange({ ...migratedQuery, kind: 'mpl' })}
          selected={migratedQuery.kind === 'mpl'}
        />
        <FilterPill
          label="Traces"
          selected={migratedQuery.kind === 'trace'}
          onClick={() => onChange({ ...migratedQuery, kind: 'trace', totals: false })}
        />
      </Stack>
      <FieldSet>
        <Field>
          {migratedQuery.kind === 'trace' ? (
            <Stack direction="column">
              <InlineField label="Trace dataset">
                <Input
                  aria-label="Trace dataset"
                  value={migratedQuery.dataset || datasource.traceDataset}
                  onChange={(e) => onChange({ ...migratedQuery, dataset: e.currentTarget.value })}
                />
              </InlineField>
              <InlineField label="Trace ID field">
                <Input
                  aria-label="Trace ID field"
                  value={migratedQuery.traceIdField || datasource.traceIdField}
                  onChange={(e) => onChange({ ...migratedQuery, traceIdField: e.currentTarget.value })}
                />
              </InlineField>
              <InlineField
                label="Trace ID"
                tooltip="Spans are fetched within the selected time range. Expand it if spans are missing."
              >
                <Input
                  aria-label="Trace ID"
                  value={migratedQuery.traceId || ''}
                  onChange={(e) => onChange({ ...migratedQuery, traceId: e.currentTarget.value })}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      onRunQuery();
                    }
                  }}
                />
              </InlineField>
            </Stack>
          ) : migratedQuery.kind === 'mpl' ? (
            <MplQueryCodeMirror
              value={queryText}
              onBlur={runMplQuery}
              onRunQuery={runMplQuery}
              onChange={(mpl) => {
                onChange({ ...migratedQuery, query: mpl });
              }}
              datasource={datasource}
              autoFocus={autoFocusEditor}
            />
          ) : (
            <APLQueryEdtior
              onChange={(apl) => {
                onChange({ ...migratedQuery, query: apl });
              }}
              value={queryText}
              datasource={datasource}
              onRunQuery={onRunQuery}
              autoFocus={autoFocusEditor}
            />
          )}
        </Field>
        {migratedQuery.kind === 'apl' && (
          <InlineFieldRow>
            <InlineField label="Query type" grow>
              <InlineSwitch
                label="Return Totals Table"
                showLabel={true}
                defaultChecked={migratedQuery.totals}
                value={migratedQuery.totals}
                onChange={onTotalsChange}
              />
            </InlineField>
          </InlineFieldRow>
        )}
      </FieldSet>
    </Stack>
  );
}
