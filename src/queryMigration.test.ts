import { migrateAxiomQuery, shouldMigrateAxiomQuery } from './queryMigration';

describe('migrateAxiomQuery', () => {
  it('migrates legacy APL queries to the v2 query model', () => {
    const migrated = migrateAxiomQuery({ refId: 'A', apl: "['logs'] | limit 10", totals: true });

    expect(migrated).toEqual({
      refId: 'A',
      version: '2.0',
      kind: 'apl',
      query: "['logs'] | limit 10",
      totals: true,
    });
    expect(migrated).not.toHaveProperty('apl');
  });

  it('migrates legacy MPL queries to the v2 query model', () => {
    const migrated = migrateAxiomQuery({ refId: 'A', query: 'metrics:cpu | group by host using avg' });

    expect(migrated).toEqual({
      refId: 'A',
      version: '2.0',
      kind: 'mpl',
      query: 'metrics:cpu | group by host using avg',
      totals: false,
    });
  });

  it('preserves an existing kind when adding the v2 version', () => {
    const migrated = migrateAxiomQuery({ refId: 'A', kind: 'apl', query: "['logs']" });

    expect(migrated).toMatchObject({
      version: '2.0',
      kind: 'apl',
      query: "['logs']",
    });
  });

  it('uses query over apl when both are present', () => {
    const migrated = migrateAxiomQuery({ refId: 'A', kind: 'apl', query: "['new']", apl: "['old']" });

    expect(migrated.query).toBe("['new']");
    expect(migrated).not.toHaveProperty('apl');
  });

  it('does not require migration after the query is v2 and apl is absent', () => {
    expect(shouldMigrateAxiomQuery({ refId: 'A', version: '2.0', kind: 'mpl', query: 'fetch cpu' })).toBe(false);
  });
});
