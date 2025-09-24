# Changelog

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
