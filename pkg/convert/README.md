# convert

## Converting Prometheus to Grafana rules

Convert the example Prometheus alert rules file to Grafana alert rules:

```shell
grr convert examples/convert/prometheus/alerts-simple.yaml resources
```

In the `./resources` you can find the newly converted Grafana alert rules. Now you can apply the converted rules to Grafana:

``` shel
grr apply resources
```
