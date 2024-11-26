package convert

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	prometheusYAML = `
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

	prometheusYAMLWithRecordingRules = `
groups:
- name: example
  rules:
  - record: job:request_duration_seconds:avg
    expr: avg(rate(api_http_request_duration_seconds_sum[5m])) by (job)
    labels:
      severity: low
    annotations:
      summary: "Average request duration for job {{ $labels.job }}"
      description: "Average request duration in seconds for job {{ $labels.job }} over the last 5 minutes."

  - record: job:cpu_usage:avg
    expr: avg(rate(cpu_usage_seconds_total[5m])) by (job)
    labels:
      severity: low
    annotations:
      summary: "Average CPU usage for job {{ $labels.job }}"
      description: "Average CPU usage in seconds for job {{ $labels.job }} over the last 5 minutes."
`
)

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

func TestPrometheusToGrafanaRecordingRules(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "prometheus-recording-rules-*.yaml")
	require.NoError(t, err, "failed to create temp file")
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(prometheusYAMLWithRecordingRules)
	require.NoError(t, err, "failed to write YAML to temp file")
	require.NoError(t, tmpFile.Close(), "failed to close temp file")

	resources, err := PrometheusToGrafanaResource(tmpFile.Name())
	require.NoError(t, err, "failed to convert Prometheus rules to Grafana resources")

	require.Len(t, resources, 2, "expected 2 resources (1 folder + 1 rule group)")

	folder := resources[0]
	require.Equal(t, "grizzly.grafana.com/v1alpha1", folder.APIVersion())
	require.Equal(t, "DashboardFolder", folder.Kind())

	recordingRuleGroup := resources[1]
	require.Equal(t, "grizzly.grafana.com/v1alpha1", recordingRuleGroup.APIVersion())
	require.Equal(t, "AlertRuleGroup", recordingRuleGroup.Kind())

	// Verify the rules in the recording rule group
	spec := recordingRuleGroup.Spec()
	require.NotNil(t, spec, "recording rule group spec should not be nil")
	require.Equal(t, "example", spec["title"], "expected recording rule group title to match Prometheus group name")

	rules := spec["rules"].([]interface{})
	require.Len(t, rules, 2, "expected 2 recording rules in the rule group")

	// Check each recording rule's record name
	expectedRecords := []string{"job:request_duration_seconds:avg", "job:cpu_usage:avg"}
	for i, rule := range rules {
		ruleMap := rule.(map[string]interface{})
		ruleMapRecord := ruleMap["record"].(map[string]interface{})
		require.Equal(t, expectedRecords[i], ruleMapRecord["metric"], "expected record name to match")
		require.Equal(t, "A", ruleMapRecord["from"], "expected from to be A")
	}
}
