package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"env-doctor/internal/checker"
	"env-doctor/internal/config"
	"env-doctor/internal/detect"
	"env-doctor/internal/ui"
	"github.com/spf13/cobra"
)

var cfgFile string
var autoDetect bool

const sampleConfig = `version: "1"

# Required binaries and their expected versions.
# Supports exact ("1.21.5"), prefix ("1.21"), and wildcard ("24.x") matching.
tools:
  go: "1.21"
  node: "20.x"
  docker: "24.x"

# Required environment variables.
env:
  - DATABASE_URL
  - REDIS_URL

# Required files.
files:
  - .env
  - config.json

# Ports that should be free or occupied.
ports:
  3000: occupied
  5432: free
`

var rootCmd = &cobra.Command{
	Use:   "env-doctor",
	Short: "Developer environment health checker",
	Long: `env-doctor reads a .env-doctor.yaml file and checks that the local
environment matches the project's requirements (tools, environment variables,
files, and ports).`,
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Run all checks from the configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return err
		}

		workDir := "."
		if cfgFile != "" {
			workDir = filepath.Dir(cfgFile)
		}

		c := checker.NewWithDir(workDir)
		results := c.Run(cfg)
		ui.Render(results)

		for _, r := range results {
			if r.Status == checker.StatusFail {
				os.Exit(1)
			}
		}
		return nil
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a sample .env-doctor.yaml in the current directory",
	Long: `Create a sample .env-doctor.yaml in the current directory.

With --auto, env-doctor inspects manifest files, local binaries, environment
files, and common project files to generate a configuration without
hardcoding knowledge of specific technology stacks.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		const filename = ".env-doctor.yaml"
		if _, err := os.Stat(filename); err == nil {
			return fmt.Errorf("%s already exists", filename)
		}

		var content string
		var detected detect.Detected
		if autoDetect {
			var err error
			detected, err = detect.Detect(".")
			if err != nil {
				return err
			}
			content, err = detect.Generate(detected)
			if err != nil {
				return fmt.Errorf("generate config: %w", err)
			}
			fmt.Printf("Created %s (auto-detected from project files)\n", filename)
		} else {
			content = sampleConfig
			fmt.Printf("Created %s\n", filename)
		}

		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", filename, err)
		}

		if autoDetect && len(detected.ToolConflicts) > 0 {
			fmt.Println()
			fmt.Println("Warnings:")
			toolNames := make([]string, 0, len(detected.ToolConflicts))
			for name := range detected.ToolConflicts {
				toolNames = append(toolNames, name)
			}
			sort.Strings(toolNames)
			for _, name := range toolNames {
				fmt.Printf("  %s: version conflicts detected\n", name)
				for _, e := range detected.ToolConflicts[name] {
					sub := filepath.Dir(e.Source)
					if sub == "." {
						sub = "root"
					}
					fmt.Printf("    - %s: %s (from %s)\n", sub, e.Version, e.Source)
				}
				fmt.Printf("    selected %s (highest version)\n", detected.Config.Tools[name])
			}
		}

		return nil
	},
}

func init() {
	checkCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "config file path (default is ./.env-doctor.yaml)")
	initCmd.Flags().BoolVar(&autoDetect, "auto", false, "auto-detect project stack and generate config")
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(initCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
