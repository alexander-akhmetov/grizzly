package convert

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const prometheusYAML = `
groups:
- name: example
  rules:
  - alert: InstanceDown
    expr: up == 0
    for: 5m
    labels:
      severity: page
    annotations:
      summary: "Instance {{ $labels.instance }} down"
      description: "{{ $labels.instance }} of job {{ $labels.job }} has been down for more than 5 minutes."

  - alert: APIHighRequestLatency
    expr: api_http_request_latencies_second{quantile="0.5"} > 1
    for: 10m
    annotations:
      summary: "High request latency on {{ $labels.instance }}"
      description: "{{ $labels.instance }} has a median request latency above 1s (current value: {{ $value }}s)"

  - alert: AlwaysFiringAlert
    expr: vector(1) > 0
    for: 1m
    annotations:
      summary: "This alert is always firing"
      description: "This alert is always firing (current value: {{ $value }}s)"
`

func TestPrometheusToGrafanaResource(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "prometheus-rules-*.yaml")
	require.NoError(t, err, "failed to create temp file")
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(prometheusYAML)
	require.NoError(t, err, "failed to write YAML to temp file")
	require.NoError(t, tmpFile.Close(), "failed to close temp file")

	resources, err := PrometheusToGrafanaResource(tmpFile.Name())
	require.NoError(t, err, "failed to convert Prometheus rules to Grafana resources")

	require.Len(t, resources, 2, "expected 2 resources (1 folder + 1 rule group")

	folder := resources[0]
	require.Equal(t, "grizzly.grafana.com/v1alpha1", folder.APIVersion())
	require.Equal(t, "DashboardFolder", folder.Kind())

	alertGroup := resources[1]
	require.Equal(t, "grizzly.grafana.com/v1alpha1", alertGroup.APIVersion())
	require.Equal(t, "AlertRuleGroup", alertGroup.Kind())

	// Verify the rules in the alert rule group
	spec := alertGroup.Spec()
	require.NotNil(t, spec, "alert group spec should not be nil")
	require.Equal(t, "example", spec["title"], "expected alert group title to match Prometheus group name")

	rules := spec["rules"].([]interface{})
	require.Len(t, rules, 3, "expected 3 rules in the alert group")

	// Check each rule's title
	expectedTitles := []string{"InstanceDown", "APIHighRequestLatency", "AlwaysFiringAlert"}
	for i, rule := range rules {
		ruleMap := rule.(map[string]interface{})
		require.Equal(t, expectedTitles[i], ruleMap["title"], "expected rule title to match")
	}
}
