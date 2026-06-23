import { AxiomQuery } from './types';

const GROUP_FUNCTION = 'sum';

export function buildMplVariableQuery(query: Pick<AxiomQuery, 'dataset' | 'metric' | 'tag'>): string {
  if (!query.dataset || !query.metric) {
    return '';
  }

  const source = `${escapeMplIdentifier(query.dataset)}:${escapeMplIdentifier(query.metric)}`;

  if (!query.tag) {
    return source;
  }

  return `${source} | group by ${escapeMplIdentifier(query.tag)} using ${GROUP_FUNCTION}`;
}

function escapeMplIdentifier(value: string): string {
  if (/^[A-Za-z_][A-Za-z0-9_]*$/.test(value)) {
    return value;
  }

  return `\`${value.replace(/`/g, '\\`')}\``;
}
