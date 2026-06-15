import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { InlineField, Input, Combobox, Stack, FilterPill, FieldSet } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';

import { APLQueryEdtior } from './AplQueryEditor';
import type { DataSource } from '../datasource';
import { buildMplVariableQuery } from '../mplVariableQuery';
import { AxiomDataSourceOptions, AxiomQuery, DEFAULT_QUERY } from '../types';

type Props = QueryEditorProps<DataSource, AxiomQuery, AxiomDataSourceOptions>;

export function VariableQueryEditor({ query, onChange, onRunQuery, datasource }: Props) {
  const variableQuery = useMemo(() => ({ ...DEFAULT_QUERY, ...query } as AxiomQuery), [query]);
  const queryText = variableQuery.query || variableQuery.apl || '';
  const aplText = variableQuery.kind === 'mpl' ? variableQuery.apl || '' : queryText;
  const [metrics, setMetrics] = useState<string[]>([]);
  const [tags, setTags] = useState<string[]>([]);
  const [isMetricsLoading, setIsMetricsLoading] = useState(false);
  const [isTagsLoading, setIsTagsLoading] = useState(false);

  const loadMetrics = useCallback(async () => {
    if (!variableQuery.dataset) {
      setMetrics([]);
      return;
    }

    try {
      setIsMetricsLoading(true);
      setMetrics(await datasource.getMetrics(variableQuery.dataset));
    } catch (error) {
      console.error('Failed to load metrics:', error);
      setMetrics([]);
    } finally {
      setIsMetricsLoading(false);
    }
  }, [variableQuery.dataset, datasource]);

  const loadTags = useCallback(async () => {
    if (!variableQuery.dataset) {
      setTags([]);
      return;
    }

    try {
      setIsTagsLoading(true);
      setTags(await datasource.getTags(variableQuery.dataset, variableQuery.metric));
    } catch (error) {
      console.error('Failed to load tags:', error);
      setTags([]);
    } finally {
      setIsTagsLoading(false);
    }
  }, [variableQuery.dataset, variableQuery.metric, datasource]);

  const debouncedLoadMetrics = useDebouncedCallback(loadMetrics, 300);
  const debouncedLoadTags = useDebouncedCallback(loadTags, 300);

  useEffect(() => {
    debouncedLoadMetrics();
  }, [variableQuery.dataset, debouncedLoadMetrics]);

  useEffect(() => {
    debouncedLoadTags();
  }, [variableQuery.dataset, variableQuery.metric, debouncedLoadTags]);

  const updateQuery = (patch: Partial<AxiomQuery>) => {
    const next = { ...variableQuery, ...patch } as AxiomQuery;
    onChange(next);
  };

  const updateMplQuery = (patch: Partial<AxiomQuery>) => {
    const next = { ...variableQuery, ...patch, kind: 'mpl', totals: false } as AxiomQuery;
    onChange({
      ...next,
      query: buildMplVariableQuery(next),
      apl: variableQuery.kind === 'apl' ? queryText : next.apl || '',
    });
  };

  return (
    <Stack direction="column">
      <Stack>
        <FilterPill
          label="APL"
          onClick={() => updateQuery({ kind: 'apl', query: aplText, apl: aplText })}
          selected={variableQuery.kind === 'apl' || !variableQuery.kind}
        />
        <FilterPill
          label="MPL"
          onClick={() => updateMplQuery({})}
          selected={variableQuery.kind === 'mpl'}
        />
      </Stack>
      {variableQuery.kind === 'mpl' ? (
        <FieldSet>
          <InlineField label="Dataset" labelWidth={12}>
            <Input
              id="variables-editor-dataset"
              name="dataset"
              value={variableQuery.dataset || ''}
              onChange={(event) =>
                updateMplQuery({
                  dataset: event.currentTarget.value,
                  metric: '',
                  tag: '',
                })
              }
              placeholder="Enter dataset name"
              width={40}
            />
          </InlineField>
          <InlineField label="Metric" labelWidth={12}>
            <Combobox
              id="variables-editor-metric"
              options={metrics.map((metric) => ({ label: metric, value: metric }))}
              value={variableQuery.metric || ''}
              onChange={(event) =>
                updateMplQuery({
                  metric: event?.value || '',
                  tag: '',
                })
              }
              placeholder="Select metric"
              width={40}
              createCustomValue
              disabled={!variableQuery.dataset}
              isClearable={true}
              loading={isMetricsLoading}
            />
          </InlineField>
          <InlineField label="Tag" labelWidth={12}>
            <Combobox
              id="variables-editor-tag"
              options={tags.map((tag) => ({ label: tag, value: tag }))}
              value={variableQuery.tag || ''}
              onChange={(event) =>
                updateMplQuery({
                  tag: event?.value || '',
                })
              }
              placeholder="Select tag"
              width={40}
              createCustomValue
              disabled={!variableQuery.dataset}
              isClearable={true}
              loading={isTagsLoading}
            />
          </InlineField>
        </FieldSet>
      ) : (
        <APLQueryEdtior
          value={aplText}
          onChange={(apl) => updateQuery({ kind: 'apl', query: apl, apl })}
          datasource={datasource}
          onRunQuery={onRunQuery}
        />
      )}
    </Stack>
  );
}

function useDebouncedCallback(callback: () => void, delay: number) {
  const callbackRef = useRef(callback);
  const timeoutRef = useRef<ReturnType<typeof setTimeout>>();

  callbackRef.current = callback;

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  return useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }

    timeoutRef.current = setTimeout(() => callbackRef.current(), delay);
  }, [delay]);
}
