package main

import (
	"fmt"
	"os"

	"github.com/go-clix/cli"
	"gopkg.in/yaml.v3"

	"github.com/grafana/grizzly/pkg/convert"
	"github.com/grafana/grizzly/pkg/grizzly/notifier"
)

func convertCmd() *cli.Command {
	cmd := &cli.Command{
		Use:   "convert <filename> <output-folder>",
		Short: "Convert resource to Grafana",
		Args:  cli.ArgsExact(2),
	}
	var opts LoggingOpts

	cmd.Run = func(cmd *cli.Command, args []string) error {
		filename := args[0]
		outputFolder := args[1]
		resources, err := convert.PrometheusToGrafanaResource(filename)

		err = os.MkdirAll(outputFolder, 0755)
		if err != nil {
			return err
		}

		// write each resource to folder/file
		for _, resource := range resources {
			resourceYaml, err := yaml.Marshal(resource.Body)
			if err != nil {
				return err
			}

			resourcePath := fmt.Sprintf("%s/%s.yaml", outputFolder, resource.String())
			err = os.WriteFile(resourcePath, resourceYaml, 0644)
			if err != nil {
				notifier.Error(nil, "Failed to write resource to file")
				return err
			}
			notifier.Info(nil, fmt.Sprintf("Resource %s is written to %s", resource.Kind(), resourcePath))
		}

		notifier.Info(nil, "")
		notifier.Info(nil, fmt.Sprintf("Successfully converted %s to Grafana resources", filename))
		notifier.Info(nil, fmt.Sprintf("To apply the resources, use the `grr apply %[1]s` command, or `grr show %[1]s` to preview the resources", outputFolder))

		return nil
	}

	return initialiseLogging(cmd, &opts)
}
