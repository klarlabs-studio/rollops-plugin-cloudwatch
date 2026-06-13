# rollops-plugin-cloudwatch

A [Rollops](https://github.com/klarlabs-studio/rollops) metric-provider plugin
backed by AWS [CloudWatch](https://aws.amazon.com/cloudwatch/). It resolves a
CloudWatch metric query to a single scalar so rollout analysis can gate a canary
on CloudWatch metrics, the way it does on Prometheus.

## How it works

Rollops calls the plugin's `query_metric` tool with a query. Because a CloudWatch
metric is multi-field, the query is a small JSON object:

```json
{
  "namespace": "AWS/ApplicationELB",
  "metricName": "HTTPCode_Target_5XX_Count",
  "stat": "Sum",
  "period": 300,
  "dimensions": { "LoadBalancer": "app/web/abc", "TargetGroup": "targetgroup/web/def" }
}
```

The plugin runs `aws cloudwatch get-metric-statistics` over a lookback window
(default 5m) and returns the requested statistic of the most recent datapoint.
It drives AWS through the `aws` CLI using ambient credentials (env, profile, or
IAM role) — no AWS SDK or request signing is compiled in.

Wire those values into a CEL condition in the analysis block:

```yaml
analysis:
  provider: cloudwatch
  plugin: ~/.rollops/plugins/cloudwatch
  sha256: <pin>
  metrics:
    - name: errors5xx
      query: '{"namespace":"AWS/ApplicationELB","metricName":"HTTPCode_Target_5XX_Count","stat":"Sum","dimensions":{"LoadBalancer":"app/web/abc"}}'
  condition: "errors5xx < 5"
  count: 3
  interval: 60s
```

## Configuration

`aws` resolves credentials as usual; the plugin reads only its own knobs from the
environment (the Rollops target spec carries only the JSON query):

| Env var             | Required | Default | Description                       |
|---------------------|----------|---------|-----------------------------------|
| `CLOUDWATCH_AWS`    | no       | `aws`   | aws binary to use                 |
| `CLOUDWATCH_REGION` | no       | —       | `--region` override               |
| `CLOUDWATCH_WINDOW` | no       | `5m`    | lookback window (Go duration)     |

## Install

```sh
rollops plugin install cloudwatch
```

## License

MIT
