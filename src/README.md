# Instantly visualize Axiom data in Grafana

The Grafana data source plugin is the easiest way to query event data from Axiom directly in Grafana dashboards.

## Requirements

This plugin has the following requirements:

- An Axiom account
- An Axiom [advanced API token with read permission](https://axiom.co/docs/reference/tokens#api-tokens)

## Configuration

1. Add a new data source in Grafana
2. Select the Axiom data source plugin
3. Enter your Axiom advanced API token
4. Save and test the data source

## Visualizing data

The Axiom data source plugin provides a custom editor to query and visualize event data from Axiom.

1. Create a new panel in Grafana
2. Select the Axiom data source
3. Use the query editor to filter, transform and analyze your data

### Metric series labels

For MPL metric queries, Grafana legends use the metric name plus the tag set by default. To make legends easier to scan, the plugin uses tag values as the leading label when tags are present, for example `200 | POST | /checkout {status=200, method=POST, route=/checkout}`.

You can control this leading label dynamically from your query by adding a `__label` tag. The plugin uses the `__label` value as the series label and still leaves the normal `{tag=value}` labels available in Grafana:

```mpl
extend __label = "${`k8s.pod.name`}"
```

## Installation

### Installation on Grafana Cloud

For more information, visit the docs on [plugin installation](https://grafana.com/docs/grafana/latest/plugins/installation/).

### Installation with Grafana CLI

```
grafana-cli plugins install axiomhq-axiom-datasource
```

### Installation with Docker

1. Add the plugin to your `docker-compose.yml` or `Dockerfile`
2. Set the environment variable `GF_INSTALL_PLUGINS` to include the plugin

```
GF_INSTALL_PLUGINS="axiomhq-axiom-datasource"
```
