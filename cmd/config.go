// Copyright (C) 2026 fakeapate <fakeapate@pm.me>
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/fakeapate/pry/config"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Print the default configuration as TOML",
	Long: `generate writes the default configuration to stdout. Redirect to a
file and edit to tune scanning parallelism, retry behaviour, Mullvad
proxy settings, and database path.`,
	Example: `  pry config generate > pry.toml
  pry --config pry.toml scan https://example.com/`,
	Run: func(cmd *cobra.Command, args []string) {
		configBytes, err := toml.Marshal(config.DefaultAppConfig())
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal default config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(configBytes))
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage pry configuration",
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(generateCmd)
}
