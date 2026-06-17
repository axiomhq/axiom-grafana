import React from 'react';
import { CodeEditor } from '@grafana/ui';
import { DatasetFields, mapDatasetInfosToSchema } from '../schema';
import type { DataSource } from '../datasource';
import { registerKustoLanguage } from '../monaco/registerKustoLanguage';

const workersAssets = require('@axiomhq/axiom-frontend-workers');

const isClientSide = typeof window !== 'undefined';

const placeholder = '// Enter an APL query (run with Ctrl/Cmd+Enter)';

function normalizeEditorQuery(value: string) {
  return value.trim() === placeholder ? '' : value;
}

function hasRunnableQuery(value: string) {
  return value.split('\n').some((line) => {
    const trimmed = line.trim();
    return trimmed !== '' && !trimmed.startsWith('//');
  });
}

const getWorker = (_: string, label: string) => {
  let targetLabel = label;

  // We don't control the moduleId monaco uses to request the editor worker. If it looks like the editor worker is being requested just use "editor".
  if (targetLabel.includes('editor')) {
    targetLabel = 'editor.ec8fde3e';
  } else {
    targetLabel = 'kusto.ec8fde3e';
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

export function APLQueryEdtior({
  value,
  onRunQuery,
  onChange,
  datasource,
  autoFocus = false,
}: {
  value: string;
  onChange: (value: string) => void;
  onRunQuery: () => void;
  datasource: DataSource;
  autoFocus?: boolean;
}) {
  const [aplEditorContent, setAplEditorContent] = React.useState('');
  const hasAutoFocusedRef = React.useRef(false);
  if (value !== aplEditorContent) {
    // query.apl could've changed from the outside (e.g. when a history query
    // is ran), so we need to update the state.
    setAplEditorContent(value);
  }

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

  const focusEditor = (editor: any) => {
    if (!autoFocus || hasAutoFocusedRef.current) {
      return;
    }

    hasAutoFocusedRef.current = true;
    setTimeout(() => editor.focus(), 0);
  };

  return (
    <CodeEditor
      onBlur={(apl) => {
        const query = normalizeEditorQuery(apl);
        onChange(query);
        setAplEditorContent(query);
        if (hasRunnableQuery(query)) {
          onRunQuery();
        }
      }}
      height="140px"
      width="500"
      value={aplEditorContent}
      language="kusto"
      showLineNumbers={true}
      showMiniMap={false}
      onBeforeEditorMount={(monaco) => {
        registerKustoLanguage(monaco);
      }}
      onEditorDidMount={async (editor, monaco) => {
        await registerKustoLanguage(monaco);

        const kustoLanguageId = 'kusto';
        const model = editor.getModel();
        if (model && model.getLanguageId?.() !== kustoLanguageId) {
          monaco.editor.setModelLanguage(model, kustoLanguageId);
        }

        const kustoLanguage = monaco.languages.getLanguages().find((l) => l.id === kustoLanguageId);
        const initializeEditor = () => {
          setAplEditorContent(value);
          addPlaceholder(editor, monaco);
          focusEditor(editor);
        };

        if (kustoLanguage) {
          // If the kusto language is already registered, we can proceed immediately
          setTimeout(() => {
            // using timeout to  ensure the editor is fully loaded and we can have syntax highlighting on initial render
            initializeEditor();
          }, 200);
        } else {
          // If the kusto language isn't registered, we need to wait for it to finish loading
          await new Promise((resolve) => {
            const disposable = monaco.languages.onLanguage(kustoLanguageId, () => {
              disposable.dispose();
              resolve(undefined);
              setTimeout(() => {
                initializeEditor();
              }, 200);
            });
          });
        }

        editor.addAction({
          id: 'submit-query',
          label: 'Submit query',
          keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
          run: async function (ed) {
            const query = normalizeEditorQuery(ed.getValue());
            onChange(query);
            setAplEditorContent(query);
            if (hasRunnableQuery(query)) {
              onRunQuery();
            }
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
  );
}
