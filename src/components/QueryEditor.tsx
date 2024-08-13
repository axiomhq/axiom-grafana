import React, { FormEvent } from 'react';
import { CodeEditor, FieldSet, Field, InlineField, InlineFieldRow, InlineSwitch } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from '../datasource';
import { AxiomDataSourceOptions, AxiomQuery } from '../types';
import { DatasetFields, mapDatasetInfosToSchema } from '../schema';

const workersAssets = require('@axiomhq/axiom-frontend-workers');

const isClientSide = typeof window !== 'undefined';

const placeholder = '// Enter an APL query (run with Ctrl/Cmd+Enter)';

const getWorker = (_: string, label: string) => {
  let targetLabel = label;

  // We don't control the moduleId monaco uses to request the editor worker. If it looks like the editor worker is being requested just use "editor".
  if (targetLabel.includes('editor')) {
    targetLabel = 'editor.4a133207';
  } else {
    targetLabel = 'kusto.4a133207';
  }

  const filename = `${targetLabel}.js`;

  const hashedFileName = workersAssets[filename]; // Retrieve the outputted filename for the worker

  // TODO: replace the plugin hard coded name with a dynamic one if possbile
  const url = `/public/plugins/axiomhq-axiom-datasource/workers/${hashedFileName || filename}`;
  return { url };
};

if (isClientSide) {
  (window as any).MonacoEnvironment = {
    // `editor.api.ts` now checks for the `MonacoEnvironment.globalAPI` variable to see if it should
    // set `window.monaco`, and since `@kusto/monaco-kusto` expects `window.monaco` to be set,
    // we need to set `MonacoEnvironment` before `monaco-editor` loads.
    globalAPI: true,
    // Browsers won't load a worker when served from a different origin but should work if constructed using a blob url.
    // See: https://stackoverflow.com/questions/58099138/getting-failed-to-construct-worker-with-js-worker-file-located-on-other-domain
    getWorker: function (moduleId: any, label: any) {
      const worker = getWorker(moduleId, label);

      if (worker) {
        const base = worker.url.startsWith('http') ? undefined : window.location.origin;
        const url = new URL(worker.url, base).href;
        const iss = `importScripts("${url}");`;

        return new Worker(URL.createObjectURL(new Blob([iss])));
      }

      return undefined;
    },
  };
}

type Props = QueryEditorProps<DataSource, AxiomQuery, AxiomDataSourceOptions>;

export function QueryEditor({ query, onChange, onRunQuery, datasource }: Props) {
  // We need to use a ref for the totals because the function that is called on
  // Cmd/Ctrl+Enter only has access to a reference of the first render because
  // it runs when Monaco is initialized.
  const totals = React.useRef(query.totals);

  const [queryStr, setQueryStr] = React.useState('');
  const { apl: queryText } = query;

  if (query.apl !== queryStr) {
    // query.apl could've changed from the outside (e.g. when a history query
    // is ran), so we need to update the state.
    setQueryStr(query.apl);
  }

  const onQueryTextChange = (apl: string) => {
    onChange({ ...query, apl });
    setQueryStr(apl);
  };

  const onTotalsChange = (e: FormEvent<HTMLInputElement>) => {
    totals.current = e.currentTarget.checked;
    onChange({
      ...query,
      totals: e.currentTarget.checked,
    });
  };

  const addPlaceholder = (editor: any, monaco: any) => {
    if (editor.getValue() === '') {
      editor.executeEdits(null, [{ range: new monaco.Range(1, 1, 1, 1), text: placeholder }]);
    }
    editor.onDidFocusEditorText(() => {
      if (editor.getValue() === placeholder) {
        editor.executeEdits(null, [{ range: new monaco.Range(1, 1, placeholder.length, 1), text: '' }]);
      }
    });
    editor.onDidBlurEditorText(() => {
      if (editor.getValue() === '') {
        editor.executeEdits(null, [{ range: new monaco.Range(1, 1, 1, 1), text: placeholder }]);
      }
    });
  };

  return (
    <FieldSet>
      <Field>
        <CodeEditor
          onBlur={onQueryTextChange}
          height="140px"
          width="500"
          value={queryStr}
          language="kusto"
          showLineNumbers={true}
          showMiniMap={false}
          onEditorDidMount={async (editor, monaco) => {
            const kustoLanguageId = 'kusto';
            const kustoLanguage = monaco.languages.getLanguages().find((l) => l.id === kustoLanguageId);
            if (kustoLanguage) {
              // If the kusto language is already registered, we can proceed immediately
              setTimeout(() => {
                // using timeout to  ensure the editor is fully loaded and we can have syntax highlighting on initial render
                setQueryStr(queryText);
                addPlaceholder(editor, monaco);
              }, 200);
            } else {
              // If the kusto language isn't registered, we need to wait for it to finish loading
              await new Promise((resolve) => {
                const disposable = monaco.languages.onLanguage(kustoLanguageId, () => {
                  disposable.dispose();
                  resolve(undefined);
                  setTimeout(() => {
                    setQueryStr(queryText);
                    addPlaceholder(editor, monaco);
                  }, 200);
                });
              });
            }

            editor.addAction({
              id: 'submit-query',
              label: 'Submit query',
              keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
              run: async function (ed) {
                const apl = ed.getValue();
                onChange({ ...query, apl, totals: totals.current });
                setQueryStr(apl);
                onRunQuery();
              },
            });

            // Should have awaited until the lang was registered so safe to access kusto?
            try {
              let res = await datasource.lookupSchema();
              let schema = mapDatasetInfosToSchema(res as DatasetFields[]);

              const workerAccessor = await (window as any).monaco.languages.kusto.getKustoWorker();

              const model = editor.getModel();
              if (model && model.uri) {
                const worker = await workerAccessor(model.uri);
                worker.setSchemaFromShowSchema(
                  schema,
                  JSON.stringify(schema), // Not really sure what to put here - it's the database connection string
                  'db', // Should be the name of the database in the schema
                  []
                );
              }
            } catch (e) {
              console.warn(e);
            }
          }}
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
