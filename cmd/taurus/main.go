// taurus generates and validates orion configs from kube-burner metrics profiles.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	kbapi "github.com/kube-burner/kube-burner/v2/pkg/prometheus/api"

	"github.com/cloud-bulldozer/orion/pkg/catalog"
	"github.com/cloud-bulldozer/orion/pkg/config"
	"github.com/cloud-bulldozer/orion/pkg/validator"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "taurus",
		Short: "Generate and validate orion configs from kube-burner metrics profiles",
	}
	root.AddCommand(generateCmd(), validateCmd(), listCmd())
	return root
}

// validateCmd checks an orion config against a metrics profile.
func validateCmd() *cobra.Command {
	var profilePath string

	cmd := &cobra.Command{
		Use:   "validate <config>",
		Short: "Validate an orion config against a metrics profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := args[0]

			cfg, err := config.LoadFile(configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			var profile []kbapi.MetricDefinition
			if profilePath != "" {
				profile, err = catalog.LoadMetricsProfile(profilePath)
				if err != nil {
					return fmt.Errorf("loading profile: %w", err)
				}
			}

			errs := validator.Validate(cfg, profile)
			if len(errs) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "OK: %s\n", configPath)
				return nil
			}
			for _, e := range errs {
				fmt.Fprintf(cmd.ErrOrStderr(), "ERROR: %s\n", e)
			}
			return fmt.Errorf("%d validation error(s)", len(errs))
		},
	}
	cmd.Flags().StringVarP(&profilePath, "profile", "p", "", "kube-burner metrics profile to validate against (optional)")
	return cmd
}

// generateCmd produces a scaffold orion config from a kube-burner metrics profile.
// Every metric in the profile becomes one orion Metric entry with no label filters,
// giving the user a starting point to customize in code or via the TUI.
func generateCmd() *cobra.Command {
	var testName string
	var output string
	var metadata []string

	cmd := &cobra.Command{
		Use:   "generate <metrics-profile>",
		Short: "Generate a scaffold orion config from a kube-burner metrics profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := catalog.LoadMetricsProfile(args[0])
			if err != nil {
				return fmt.Errorf("loading profile: %w", err)
			}

			// Build metadata from --metadata key=value flags.
			meta := config.Metadata{}
			for _, kv := range metadata {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid metadata %q: expected key=value", kv)
				}
				meta.Set(parts[0], parts[1])
			}

			// One MetricSpec per profile metric, no label filters.
			specs := make([]config.MetricSpec, 0, len(profile))
			for _, m := range profile {
				specs = append(specs, config.MetricSpec{
					Name:              m.MetricName,
					ProfileMetricName: m.MetricName,
				})
			}

			cfg, err := config.GenerateConfig(profile, []config.TestSpec{
				{
					Name:     testName,
					Metadata: meta,
					Metrics:  specs,
				},
			})
			if err != nil {
				return fmt.Errorf("generating config: %w", err)
			}

			data, err := config.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("marshaling config: %w", err)
			}

			if output == "" || output == "-" {
				_, err = cmd.OutOrStdout().Write(data)
				return err
			}
			return os.WriteFile(output, data, 0644)
		},
	}

	cmd.Flags().StringVarP(&testName, "test-name", "t", "my-test", "test name in the generated config")
	cmd.Flags().StringVarP(&output, "output", "o", "-", "output file (default: stdout)")
	cmd.Flags().StringArrayVarP(&metadata, "metadata", "m", nil, "metadata key=value pairs (repeatable)")
	return cmd
}

// listCmd prints all metric names in a metrics profile.
func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <metrics-profile>",
		Short: "List all metric names in a kube-burner metrics profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			metrics, err := catalog.LoadMetricsProfile(args[0])
			if err != nil {
				return err
			}
			for _, m := range metrics {
				labels := catalog.ExtractLabels(m.Query)
				if len(labels) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "%-40s labels: %v\n", m.MetricName, labels)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\n", m.MetricName)
				}
			}
			return nil
		},
	}
}
