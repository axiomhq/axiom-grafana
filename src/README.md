# Instantly visualize Axiom data in Grafana

The Grafana data source plugin is the easiest way to query event data from Axiom directly in Grafana dashboards.

## Requirements

This plugin has the following requirements:

- An Axiom account
- An Axiom [personal access token](https://axiom.co/docs/reference/settings#personal-token)

## Configuration

1. Add a new data source in Grafana
2. Select the Axiom data source plugin
3. Enter your Axiom personal access token and organization ID
4. Save and test the data source

## Visualizing data

The Axiom data source plugin provides a custom editor to query and visualize event data from Axiom.

1. Create a new panel in Grafana
2. Select the Axiom data source
3. Use the query editor to filter, transform and analyze your data

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
