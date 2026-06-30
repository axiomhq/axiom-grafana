# Changelog

## 0.7.0

- Add support for MPL queries in panels and variables, including the MPL editor, metric selectors, tag lookup resources, and chart-width forwarding.
- Migrate queries to the v2 query model with explicit `kind` and `query` fields while preserving runtime migration for legacy saved APL queries.
- Return Grafana-native frames for APL time series, logs, traces, generic tables, and logs-volume results.
- Improve metric series display names with tag-derived labels and `__label` legend overrides.
- Upgrade Grafana frontend dependencies and plugin SDK support for newer Grafana versions and React 19.
- Fix edge URL handling and datasource credentials validation.
- Fix provisioned dashboard query shape, APL/MPL variable query editing, Kusto/MPL editor initialization, empty-query execution, and trace/log frame normalization.

### Breaking changes

- Personal tokens are no longer allowed; datasource authentication now requires API tokens.
- `Edge URL` is now required in the plugin config and is used for all query operations. Existing v0.6.x data sources that only have `apiHost` and an API token must be edited after upgrade to set **Edge URL**. For provisioned data sources, add `jsonData.edgeURL`; legacy `jsonData.edge` values continue to be migrated automatically.

## 0.6.4

- Add support for regional edge query endpoints and explicit edge URLs.
- Add smart edge URL path handling for custom query endpoints.
- Bump `axiom-go` to v0.29.0, including support for `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment variables.
- Upgrade the Grafana Go SDK.
- Update release automation for the current Grafana plugin build, signing, and attestation flow.

## 0.6.3

- Update Grafana plugin SDK to v12
- Fix datasource settings to expect injected Grafana settings that are not strings

## 0.5.1

- fix(repeat panels): use scopedVars for query by @schehata in #69

## 0.5.0

- feat: bump frontend-workers/apl by @kevinehosford in #64
- fix(monaco): update hash and re-add copy step by @kevinehosford in #67
- use tabular-row format by @schehata in #66
- Bump fast-loops from 1.1.3 to 1.1.4 by @dependabot in #65
- Bump micromatch from 4.0.5 to 4.0.8 by @dependabot in #68
- Bump braces from 3.0.2 to 3.0.3 by @dependabot in #63

## 0.4.0

- Added support for advanced API tokens by @schehata (#59)
- Deprecate usage of Personal Tokens by @schehata (#59)
- Upgraded axiom-go to version v0.17.8

## 0.3.1

- Show the URL config field in plugin settings by @schehata in #58

## 0.3.0

- fix: Run queries concurrently to speed up panels with many queries by @jahands in #40
- testing: provide a datasource & dashboard for quick testing by @schehata in #57


## 0.2.0

- feat: replace plugin screenshots by @dominicchapman in #29
- Add placeholder, query shortcut to QueryEditor by @bahlo in #30
- Fix totals switch when submitting via keybinding by @bahlo in #31
- Fix history bugs by @bahlo in #32

## 0.1.9

- switch to using call resource functionality by @mschoch in #23
- handle totals table groups same as series by @mschoch in #24
- fix: mismatch between values and fields arrays and longToWide on … by @a-khaledf in #21
- docs: update README by @dominicchapman in #25
- feat: revise Axiom logo for light background by @dominicchapman in #26
- fix handling of fields with escaped dots by @mschoch in #28
