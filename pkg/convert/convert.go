package convert

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/grizzly/pkg/convert/promtografana"
	"github.com/grafana/grizzly/pkg/grizzly"
)

const (
	apiVersion = "grizzly.grafana.com/v1alpha1"
	kind       = "AlertRuleGroup"
	folderKind = "DashboardFolder"
)

func PrometheusToGrafanaResource(filename string) ([]grizzly.Resource, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	namespace := filepath.Base(filename)
	grafanaGroups, err := promtografana.PrometheusRulesToGrafana(namespace, file)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Prometheus rules: %w", err)
	}

	return convertGrafanaToGrizzly(grafanaGroups)
}

// convertGrafanaToGrizzly converts Grafana alert rule groups into grizzly.Resources.
func convertGrafanaToGrizzly(grafanaGroups []models.AlertRuleGroup) ([]grizzly.Resource, error) {
	resources := make([]grizzly.Resource, 0, len(grafanaGroups)+1)

	folderUID, folderResource, err := createFolderResource(grafanaGroups)
	if err != nil {
		return nil, fmt.Errorf("failed to create folder resource: %w", err)
	}
	resources = append(resources, folderResource)

	for _, grafanaGroup := range grafanaGroups {
		spec, err := structToMap(grafanaGroup)
		if err != nil {
			return nil, fmt.Errorf("failed to convert struct to map: %w", err)
		}

		// Grizzly expects the resource name for AlertRuleGroup to be in the format of folderUID.groupTitle
		folderAndUID := fmt.Sprintf("%s.%s", folderUID, grafanaGroup.Title)

		resource, err := grizzly.NewResource(apiVersion, kind, folderAndUID, spec)
		if err != nil {
			return nil, fmt.Errorf("failed to create grizzly resource: %w", err)
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

func createFolderResource(grafanaGroups []models.AlertRuleGroup) (string, grizzly.Resource, error) {
	if len(grafanaGroups) == 0 {
		return "", grizzly.Resource{}, fmt.Errorf("no Grafana groups provided")
	}

	folderUID := grafanaGroups[0].FolderUID
	spec := map[string]interface{}{
		"uid":   folderUID,
		"title": folderUID,
	}

	resource, err := grizzly.NewResource(apiVersion, folderKind, folderUID, spec)
	if err != nil {
		return "", grizzly.Resource{}, err
	}

	return folderUID, resource, nil
}
