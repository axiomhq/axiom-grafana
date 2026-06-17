import React, { useEffect, useRef } from 'react';
import { css } from '@emotion/css';
import { useTheme2 } from '@grafana/ui';
import { GrafanaTheme2 } from '@grafana/data';
import { EditorView, basicSetup } from 'codemirror';
import { EditorState, Prec } from '@codemirror/state';
import { keymap } from '@codemirror/view';
import { acceptCompletion } from '@codemirror/autocomplete';
import { indentWithTab } from '@codemirror/commands';
import { indentUnit } from '@codemirror/language';
import { oneDark } from '@codemirror/theme-one-dark';
import {
  mplHighlighter,
  createMplCompletion,
  mplLinter,
  mplSignatureHelp,
  mplHover,
  mplSystemParams,
} from '@axiomhq/mpl-codemirror';
import { ensureMplInit } from '../mpl/ensureMplInit';
import type { DataSource } from '../datasource';
import { MPL_SYSTEM_PARAMS } from '../mpl/constants';

const editorHeight = 140;
const editorTabSize = 2;
const editorIndentUnit = '  ';

function getMplTokenStyles(theme: GrafanaTheme2) {
  const isDark = theme.isDark;
  const editorVerticalPadding = `${0.5 * theme.spacing.gridSize}px`;
  return css({
    '& .cm-editor': {
      height: '100%',
    },
    '& .cm-scroller': {
      overflow: 'auto',
    },
    '& .cm-content': {
      padding: `${editorVerticalPadding} 0`,
    },
    '& .cm-lineNumbers .cm-gutterElement': {
      minWidth: '4ch',
    },
    // Syntax highlighting
    '& .mpl-keyword': { color: isDark ? '#c678dd' : '#7c3aed', fontWeight: 500 },
    '& .mpl-variable': { color: isDark ? '#e06c75' : '#0550ae' },
    '& .mpl-string': { color: isDark ? '#98c379' : '#0a3069' },
    '& .mpl-number': { color: isDark ? '#d19a66' : '#0550ae' },
    '& .mpl-bool': { color: isDark ? '#d19a66' : '#cf222e' },
    '& .mpl-regexp': { color: isDark ? '#56b6c2' : '#116329' },
    '& .mpl-operator': { color: isDark ? '#56b6c2' : '#cf222e' },
    '& .mpl-punctuation': { color: isDark ? '#abb2bf' : '#cf222e' },
    '& .mpl-type': { color: isDark ? '#56b6c2' : '#0550ae', fontStyle: 'italic' },
    '& .mpl-comment': { color: isDark ? '#5c6370' : '#6e7781', fontStyle: 'italic' },
    // Signature help tooltip
    '& .mpl-signature-help': {
      fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, monospace',
      fontSize: 13,
      padding: '6px 10px',
      maxWidth: 500,
    },
    '& .mpl-signature-sig': { whiteSpace: 'nowrap' },
    '& .mpl-signature-fn': { color: isDark ? '#c678dd' : '#7c3aed', fontWeight: 600 },
    '& .mpl-signature-param.active': {
      fontWeight: 700,
      textDecoration: 'underline',
      color: isDark ? '#61afef' : '#0550ae',
    },
    '& .mpl-signature-doc': {
      marginTop: 4,
      fontSize: 12,
      whiteSpace: 'pre-wrap',
    },
    // Hover tooltip
    '& .mpl-hover-tooltip': {
      fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, monospace',
      fontSize: 13,
      padding: '6px 10px',
      maxWidth: 500,
    },
    '& .mpl-hover-sig': { whiteSpace: 'nowrap' },
    '& .mpl-hover-fn': { color: isDark ? '#c678dd' : '#7c3aed', fontWeight: 600 },
    '& .mpl-hover-keyword': { color: isDark ? '#c678dd' : '#7c3aed', fontWeight: 600 },
    '& .mpl-hover-doc': {
      marginTop: 4,
      fontSize: 12,
      whiteSpace: 'pre-wrap',
    },
    '& .mpl-hover-syntax': {
      marginTop: 4,
      fontSize: 12,
      padding: '2px 6px',
      borderRadius: 3,
    },
  });
}

interface Props {
  value: string;
  onChange: (value: string) => void;
  onBlur?: (value: string) => void;
  onRunQuery?: (value: string) => void;
  datasource: DataSource;
  autoFocus?: boolean;
}

export function MplQueryCodeMirror({ value, onChange, onBlur, onRunQuery, datasource, autoFocus = false }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const onChangeRef = useRef(onChange);
  const onBlurRef = useRef(onBlur);
  const onRunQueryRef = useRef(onRunQuery);
  const valueRef = useRef(value);
  const datasourceRef = useRef(datasource);
  const hasAutoFocusedRef = useRef(false);
  const theme = useTheme2();
  const tokenStyles = getMplTokenStyles(theme);

  onChangeRef.current = onChange;
  onBlurRef.current = onBlur;
  onRunQueryRef.current = onRunQuery;
  valueRef.current = value;
  datasourceRef.current = datasource;

  useEffect(() => {
    if (!containerRef.current) {
      return;
    }

    let view: EditorView | null = null;
    let cancelled = false;

    ensureMplInit()
      .then(() => {
        if (cancelled || !containerRef.current) {
          return;
        }
        const completionExt = createMplCompletion({
          datasets: () => datasourceRef.current.getMetricsDatasets(),
          metrics: (dataset: string) => datasourceRef.current.getMetrics(dataset),
          tags: (dataset: string, metric: string) => datasourceRef.current.getTags(dataset, metric),
        });

        const extensions = [
          basicSetup,
          EditorState.tabSize.of(editorTabSize),
          indentUnit.of(editorIndentUnit),
          Prec.highest(
            keymap.of([
              { key: 'Tab', run: acceptCompletion },
              indentWithTab,
              {
                key: 'Mod-Enter',
                run: (view) => {
                  onRunQueryRef.current?.(view.state.doc.toString());
                  return true;
                },
              },
            ])
          ),
          EditorView.lineWrapping,
          mplSystemParams.of(MPL_SYSTEM_PARAMS),
          mplHighlighter,
          completionExt,
          mplLinter,
          mplSignatureHelp,
          mplHover,
          EditorView.updateListener.of((update) => {
            if (update.docChanged) {
              onChangeRef.current(update.state.doc.toString());
            }
          }),
          EditorView.domEventHandlers({
            blur: (_, view) => {
              onBlurRef.current?.(view.state.doc.toString());
              return false;
            },
          }),
        ];

        if (theme.isDark) {
          extensions.push(oneDark);
        }

        view = new EditorView({
          state: EditorState.create({
            doc: valueRef.current,
            extensions,
          }),
          parent: containerRef.current,
        });
        viewRef.current = view;

        if (autoFocus && !hasAutoFocusedRef.current) {
          hasAutoFocusedRef.current = true;
          requestAnimationFrame(() => {
            if (!cancelled) {
              view?.focus();
            }
          });
        }
      })
      .catch((err) => console.error);

    return () => {
      cancelled = true;
      if (view) {
        view.destroy();
        viewRef.current = null;
      }
    };
    // Re-create the editor when theme changes to apply dark/light mode
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [theme.isDark]);

  // Sync external value changes into the editor
  useEffect(() => {
    const view = viewRef.current;
    if (!view) {
      return;
    }
    const current = view.state.doc.toString();
    if (value !== current) {
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: value },
      });
    }
  }, [value]);

  return <div ref={containerRef} className={tokenStyles} style={{ height: editorHeight }} />;
}
