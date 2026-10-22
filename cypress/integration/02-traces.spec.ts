import { e2e } from '@grafana/e2e';

// Mock Grafana's query response so Explore coverage needs no Axiom credentials.
e2e.scenario({
  describeName: 'Trace lookup',
  itName: 'opens a trace from a log link and renders its child span',
  scenario: () => {
    cy.request('/api/datasources/name/Axiom').then(({ body: datasource }) => {
      cy.intercept('GET', '**/resources/schema-lookup', { body: [] });
      cy.request('PUT', `/api/datasources/uid/${datasource.uid}`, {
        ...datasource,
        jsonData: { ...datasource.jsonData, traceDataset: 'spans', traceIdField: 'trace_id' },
      });
      cy.intercept('POST', '/api/ds/query*', (req) => {
        if (req.body.queries[0].supportingQueryType === 'LogsVolume') {
          req.reply({ results: { [req.body.queries[0].refId]: { status: 200, frames: [] } } });
          return;
        }
        const trace = req.body.queries[0].kind === 'trace';
        if (trace) {
          expect(req.body.queries[0].traceId).to.equal('abc');
          expect(req.body.queries[0].dataset).to.equal('spans');
        }
        const names = trace
          ? [
              'traceID',
              'spanID',
              'parentSpanID',
              'operationName',
              'serviceName',
              'serviceTags',
              'startTime',
              'duration',
              'tags',
            ]
          : ['timestamp', 'body', 'labels'];
        const types = trace
          ? ['string', 'string', 'string', 'string', 'string', 'other', 'number', 'number', 'other']
          : ['time', 'string', 'other'];
        req.reply({
          results: {
            A: {
              status: 200,
              frames: [
                {
                  schema: {
                    name: trace ? 'Trace' : 'Logs',
                    refId: 'A',
                    meta: { preferredVisualisationType: trace ? 'trace' : 'logs' },
                    fields: names.map((name, index) => ({
                      name,
                      type: types[index],
                      typeInfo: { frame: types[index] },
                    })),
                  },
                  data: {
                    values: trace
                      ? [
                          ['abc', 'abc'],
                          ['root', 'child'],
                          [null, 'root'],
                          ['request', 'database child'],
                          ['api', 'db'],
                          [[], []],
                          [Date.now(), Date.now()],
                          [10, 5],
                          [[], []],
                        ]
                      : [[Date.now()], ['request completed'], [{ trace_id: 'abc' }]],
                  },
                },
              ],
            },
          },
        });
      }).as('query');
      const panes = {
        left: {
          datasource: datasource.uid,
          queries: [{ refId: 'A', kind: 'apl', query: "['logs']", totals: false }],
          range: { from: 'now-1h', to: 'now' },
        },
      };
      cy.visit(`/explore?schemaVersion=1&panes=${encodeURIComponent(JSON.stringify(panes))}`);
      cy.wait('@query');
      cy.contains('request completed').click();
      cy.contains('View trace').click();
      cy.contains('database child').should('be.visible');
      cy.get('input[aria-label="Trace ID"]').should('have.value', 'abc');
      cy.request('PUT', `/api/datasources/uid/${datasource.uid}`, datasource);
    });
  },
});
