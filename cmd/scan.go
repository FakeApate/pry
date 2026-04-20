// Copyright (C) 2026 fakeapate <fakeapate@pm.me>
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"net/url"
	"os"

	"github.com/fakeapate/pry/internal/orchestrator"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan <url> [url...]",
	Short: "Scan one or more open directories headlessly",
	Long: `scan dispatches every given URL to the scanner and blocks until
every scan finishes. Findings are stored in the SQLite database; use
` + "`pry export`" + ` to produce HTML/JSON/CSV afterwards, or launch
` + "`pry`" + ` with no arguments to browse in the TUI.`,
	Example: `  pry scan https://example.com/files/
  pry scan https://a.example.com/ https://b.example.com/`,
	Run: run,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
			return err
		}
		for _, v := range args {
			if _, err := url.ParseRequestURI(v); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
}

func run(cmd *cobra.Command, args []string) {
	o := orchestrator.GetInstance(nil)
	defer o.Close()

	for _, v := range args {
		if _, err := o.Dispatch(v); err != nil {
			fmt.Fprintf(os.Stderr, "dispatch error for %s: %v\n", v, err)
		}
	}

	o.Wait()
}
