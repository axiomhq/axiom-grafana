import type { MplSystemParam } from '@axiomhq/mpl-codemirror';

/**
 * System params the datasource injects at query time. Shared by the
 * editor (via `mplSystemParams` facet) and the wasm `diagnostics()` calls
 * so the language server, lint pass, and backend stay in sync.
 */
export const MPL_SYSTEM_PARAMS: MplSystemParam[] = [
  { name: '__interval', type: 'Duration' },
];

export const timeAggrOpts = ['Count', 'Sum', 'Avg', 'Min', 'Max'].map((value) => ({ label: value, value }));
export const tagAggrOpts = ['Count', 'Sum', 'Avg', 'Min', 'Max'].map((value) => ({ label: value, value }));
export const filterOperators = ['=~', '!~', '==', '!=', '>', '>=', '<', '<='].map((op) => ({ label: op, value: op }));
export const mapAggrOpts = ['Rate', 'Increase', 'Min', 'Max', 'Add', 'Sub', 'Mul', 'Div', 'Abs', 'FillConst', 'FillPrev'].map((value) => ({ label: value, value }));
