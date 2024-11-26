package promtografana

import (
	"fmt"
	"io"
	"strings"

	"github.com/go-openapi/strfmt"
	"github.com/grafana/grafana-openapi-client-go/models"
	"gopkg.in/yaml.v3"
)

const (
	defaultDatasourceUID = "grafanacloud-prom"
	defaultReceiver      = "grafana-default-email"
	defaultTimeRange     = 600
	defaultExecErrState  = "OK"
	defaultNoDataState   = "NoData"
	defaultInterval      = 60
)

// PrometheusRulesToGrafana converts a Prometheus rules file into Grafana alert rule groups.
func PrometheusRulesToGrafana(namespace string, reader io.Reader) ([]models.AlertRuleGroup, error) {
	promFile, err := readPrometheusRules(reader)
	if err != nil {
		return nil, err
	}

	for _, group := range promFile.Groups {
		for _, rule := range group.Rules {
			err := validatePrometheusRule(rule)
			if err != nil {
				return nil, fmt.Errorf("invalid Prometheus rule '%s': %w", rule.Alert, err)
			}
		}
	}

	var grafanaGroups []models.AlertRuleGroup

	for _, group := range promFile.Groups {
		folderUID := getFolderUID(namespace)
		grafanaGroup, err := convertPrometheusToGrafanaRuleGroup(folderUID, group)
		if err != nil {
			return nil, fmt.Errorf("failed to convert rule group '%s': %w", group.Name, err)
		}
		grafanaGroups = append(grafanaGroups, grafanaGroup)
	}

	return grafanaGroups, nil
}

func readPrometheusRules(reader io.Reader) (*PrometheusRulesFile, error) {
	var ruleFile PrometheusRulesFile
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&ruleFile); err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	return &ruleFile, nil
}

func validatePrometheusRule(rule PrometheusRule) error {
	if rule.KeepFiringFor != "" {
		return fmt.Errorf("keep_firing_for is not supported")
	}

	return nil
}

func convertPrometheusToGrafanaRuleGroup(folderUID string, promGroup PrometheusRuleGroup) (models.AlertRuleGroup, error) {
	rules := make([]*models.ProvisionedAlertRule, 0, len(promGroup.Rules))
	for _, rule := range promGroup.Rules {
		gr, err := prometheusToGrafanaRule(folderUID, promGroup.Name, rule)
		if err != nil {
			return models.AlertRuleGroup{}, fmt.Errorf("failed to convert Prometheus rule '%s' to Grafana rule: %w", rule.Alert, err)
		}
		rules = append(rules, gr)
	}

	result := models.AlertRuleGroup{
		FolderUID: folderUID,
		Interval:  defaultInterval,
		Rules:     rules,
		Title:     promGroup.Name,
	}

	return result, nil
}

func prometheusToGrafanaRule(folderUID string, group string, rule PrometheusRule) (*models.ProvisionedAlertRule, error) {
	var duration strfmt.Duration
	if rule.For != "" {
		err := duration.UnmarshalText([]byte(rule.For))
		if err != nil {
			return nil, fmt.Errorf("invalid duration '%s': %w", rule.For, err)
		}
	}

	result := &models.ProvisionedAlertRule{
		OrgID:        nil, // OrgID can be set if needed
		FolderUID:    &folderUID,
		Title:        stringPtr(rule.Alert),
		ExecErrState: stringPtr(defaultExecErrState),
		Annotations:  rule.Annotations,
		Condition:    stringPtr("B"),
		Data: []*models.AlertQuery{
			alertQueryNode(rule.Expr),
		},
		For:         &duration,
		IsPaused:    false,
		Labels:      rule.Labels,
		NoDataState: stringPtr(defaultNoDataState),
		NotificationSettings: &models.AlertRuleNotificationSettings{
			Receiver: stringPtr(defaultReceiver),
		},
		Provenance: "",
		RuleGroup:  stringPtr(group),
	}

	if rule.Record != "" {
		result.Record = &models.Record{
			From:   stringPtr("A"),
			Metric: stringPtr(rule.Record),
		}
	} else {
		result.Data = append(result.Data, alertConditionNode())
	}

	return result, nil
}

func alertQueryNode(expr string) *models.AlertQuery {
	return &models.AlertQuery{
		DatasourceUID: defaultDatasourceUID,
		Model: map[string]interface{}{
			"datasource": map[string]interface{}{
				"type": "prometheus",
				"uid":  defaultDatasourceUID,
			},
			"editorMode":    "code",
			"expr":          expr,
			"instant":       true,
			"range":         false,
			"intervalMs":    1000,
			"legendFormat":  "__auto",
			"maxDataPoints": 43200,
			"refId":         "A",
		},
		RefID: "A",
		RelativeTimeRange: &models.RelativeTimeRange{
			From: defaultTimeRange,
			To:   0,
		},
	}
}

func alertConditionNode() *models.AlertQuery {
	return &models.AlertQuery{
		DatasourceUID: "__expr__",
		Model: map[string]interface{}{
			"datasource": map[string]interface{}{
				"type": "__expr__",
				"uid":  "__expr__",
			},
			"conditions": []interface{}{
				map[string]interface{}{
					"evaluator": map[string]interface{}{
						"params": []interface{}{0},
						"type":   "gt",
					},
					"operator": map[string]interface{}{
						"type": "and",
					},
					"query": map[string]interface{}{
						"params": []interface{}{"B"},
					},
					"reducer": map[string]interface{}{
						"params": []interface{}{},
						"type":   "last",
					},
					"type": "query",
				},
			},
			"intervalMs":    1000,
			"expression":    "A",
			"legendFormat":  "__auto",
			"maxDataPoints": 43200,
			"refId":         "B",
			"type":          "threshold",
		},
		RefID: "B",
		RelativeTimeRange: &models.RelativeTimeRange{
			From: defaultTimeRange,
			To:   0,
		},
	}
}

func stringPtr(s string) *string {
	return &s
}

func getFolderUID(namespace string) string {
	folderUID := strings.ReplaceAll(namespace, ".", "_")
	return folderUID
}
