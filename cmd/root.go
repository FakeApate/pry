// Copyright (C) 2026 fakeapate <fakeapate@pm.me>
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/BurntSushi/toml"
	charmlog "github.com/charmbracelet/log"
	"github.com/fakeapate/pry/config"
	"github.com/fakeapate/pry/internal/orchestrator"
	"github.com/fakeapate/pry/internal/tui"
	"github.com/spf13/cobra"
)

var cfgFile string
var debugFlag bool
var rootCmd = &cobra.Command{
	Use:   "pry",
	Short: "Scan open HTTP directories and catalogue interesting files",
	Long: `pry scans open directory listings over HTTP, classifies every
finding (category + interest score), and stores results in SQLite for
later exploration and export.

Without arguments, pry launches a terminal UI for browsing past
scans, running new ones, and viewing findings as a table or tree.

Subcommands run headlessly: ` + "`scan`" + ` dispatches one or more URLs and
waits for completion, ` + "`export`" + ` writes HTML/JSON/CSV output for a scan,
and ` + "`config generate`" + ` prints the default configuration.`,
	Example: `  # launch the TUI
  pry

  # scan one or more URLs headlessly
  pry scan https://example.com/files/

  # export the most recent scan as interactive HTML
  pry export --last

  # dump the default config
  pry config generate > pry.toml
  pry --config pry.toml`,
	Run: func(cmd *cobra.Command, args []string) {
		log.SetOutput(io.Discard)
		charmlog.Default().SetOutput(io.Discard)

		orch := orchestrator.GetInstance(nil)
		defer orch.Close()

		p := tea.NewProgram(tui.NewModel(orch.GetDB(), orch.Dispatch))
		orch.SetProgram(p)
		if _, err := p.Run(); err != nil {
			fmt.Println("Error starting TUI:", err)
			os.Exit(1)
		}
	},
	Args: func(cmd *cobra.Command, args []string) error {
		for _, v := range args {
			if _, err := url.ParseRequestURI(v); err != nil {
				return err
			}
		}
		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "path to a TOML config file (see `pry config generate`)")
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "enable verbose logging")
	err := rootCmd.MarkPersistentFlagFilename("config", "toml")
	cobra.CheckErr(err)
}

func initConfig() {
	var cfg config.AppConfig
	if cfgFile != "" {
		_, err := toml.DecodeFile(cfgFile, &cfg)
		cobra.CheckErr(err)
	} else {
		cfg = config.DefaultAppConfig()
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
		os.Exit(1)
	}
	config.SetConfig(cfg)
}
