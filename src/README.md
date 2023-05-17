# Instantly visualize Axiom data in Grafana

The Grafana data source plugin is the easiest way to query event data from Axiom directly in Grafana dashboards.

## Requirements

This plugin has the following requirements:

- An Axiom account
- An Axiom API token with query permissions for desired datasets

## Installation

### Install plugin on Grafana Cloud

1. Navigate to the official Grafana Plugins page
2. Select the Axiom data source plugin in the Installation tab
3. Configure the data source with a name and your Axiom API token

### Install plugin on local Grafana

#### Install with Grafana CLI

```
grafana-cli plugins install axiomhq-axiom-datasource
```

#### Install with Docker

1. Add the plugin to your `docker-compose.yml` or `Dockerfile`
2. Set the environment variable `GF_INSTALL_PLUGINS` to include the plugin

Example:

```
GF_INSTALL_PLUGINS="axiomhq-axiom-datasource"
```

#### Install for local development

```shell
$ yarn install
$ yarn dev
```

Run the following in another shell:

```shell
$ mage -v && docker-compose up
```

Open http://localhost:3000 and add Axiom as a data source.

## Visualizing data

The Axiom data source plugin provides a custom editor to query and visualize your Axiom event data.

1. Create a new panel in Grafana
2. Select the Axiom data source
3. Use the query editor to filter, transform and analyze your data
