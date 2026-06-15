import { AxiomQuery, AxiomQueryKind, QUERY_MODEL_VERSION } from './types';

type MigratedAxiomQuery<T extends Partial<AxiomQuery>> = Omit<T, 'apl'> & {
  version: typeof QUERY_MODEL_VERSION;
  kind: AxiomQueryKind;
  query: string;
};

export function shouldMigrateAxiomQuery(query?: Partial<AxiomQuery> | null): boolean {
  if (!query) {
    return false;
  }

  return query.version !== QUERY_MODEL_VERSION || !query.kind || 'apl' in query;
}

export function migrateAxiomQuery<T extends Partial<AxiomQuery>>(query: T): MigratedAxiomQuery<T> {
  const { apl, ...queryWithoutAPL } = query;
  const queryText = query.query || apl || '';
  const kind = query.kind || inferLegacyQueryKind(query, queryText);

  return {
    ...queryWithoutAPL,
    version: QUERY_MODEL_VERSION,
    kind,
    query: queryText,
    totals: query.totals ?? false,
  } as MigratedAxiomQuery<T>;
}

function inferLegacyQueryKind(query: Partial<AxiomQuery>, queryText: string): AxiomQueryKind {
  if (query.apl !== undefined) {
    return 'apl';
  }

  if (queryText) {
    return 'mpl';
  }

  return 'apl';
}
