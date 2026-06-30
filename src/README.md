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
4. Confirm the Edge URL for your Axiom deployment. The default is `https://us-east-1.aws.edge.axiom.co`
5. Save and test the data source

## Upgrading from v0.6.x to v0.7.0

v0.7.0 no longer supports Personal token plus Org ID authentication. If an existing data source still uses a Personal token and Org ID, update it to use an Axiom API token before upgrading so dashboards using that data source continue to authenticate.

v0.7.0 uses Edge URL for query operations. Existing v0.6.x data sources that only have an API URL and API token continue to work: when neither `edge` nor `edgeURL` is configured, the plugin defaults `edgeURL` to `https://us-east-1.aws.edge.axiom.co` at config load.

No manual migration is required for data sources that should use the default edge endpoint. If your Axiom deployment uses a different edge endpoint, update the data source:

1. In Grafana, open **Connections > Data sources > Axiom**
2. Set **Edge URL** to the edge endpoint for your Axiom deployment
3. Save and test the data source

If the data source is provisioned and should use a non-default edge endpoint, add `edgeURL` under `jsonData`:

```yaml
jsonData:
  apiHost: https://api.axiom.co
  edgeURL: https://your-edge.example.com
```

Data sources that already used the legacy `edge` setting continue to be migrated automatically to `edgeURL`.

## Visualizing data

The Axiom data source plugin provides a custom editor to query and visualize event data from Axiom.

1. Create a new panel in Grafana
2. Select the Axiom data source
3. Use the query editor to filter, transform and analyze your data

### Metric series labels

For MPL metric queries, Grafana legends use the metric name plus the tag set by default. To make legends easier to scan, the plugin uses tag values as the leading label when tags are present, for example `200 | POST | /checkout {status=200, method=POST, route=/checkout}`.

You can control the full legend dynamically from your query by adding a `__label` tag. The plugin uses the `__label` value as the series label and hides the normal `{tag=value}` suffix from the legend, while still leaving those labels available in Grafana:

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
