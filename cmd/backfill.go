// Copyright (C) 2026 fakeapate <fakeapate@pm.me>
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/fakeapate/pry/config"
	"github.com/fakeapate/pry/internal/classify"
	"github.com/fakeapate/pry/internal/store"
	storedb "github.com/fakeapate/pry/internal/store/db"
	"github.com/spf13/cobra"
)

var backfillCmd = &cobra.Command{
	Use:   "backfill",
	Short: "Re-classify findings from scans taken before the tagging feature",
	Long: `backfill reads existing scan_findings that have no category and runs
the classifier over each one to populate category, interest score, and
tags. Safe to run multiple times: scans already classified are skipped.`,
	Example: `  pry backfill`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.GetConfig()
		database, err := store.OpenDB(cfg.Database.DBPath)
		cobra.CheckErr(err)
		defer database.Close()

		if err := store.MigrateUp(database); err != nil {
			cobra.CheckErr(err)
		}

		queries := storedb.New(database)
		scans, err := queries.ListScans(context.Background())
		cobra.CheckErr(err)

		for _, scan := range scans {
			findings, err := queries.ListScanFindings(context.Background(), scan.ScanID)
			cobra.CheckErr(err)

			if len(findings) == 0 {
				continue
			}

			// Skip if already classified
			if findings[0].Category != "" {
				fmt.Printf("Scan %s: already classified, skipping\n", scan.ScanID[:8])
				continue
			}

			tx, err := database.Begin()
			cobra.CheckErr(err)

			for _, f := range findings {
				cl := classify.Classify(f.Url, f.ContentType, f.ContentLength)
				_, err := tx.Exec(
					"UPDATE scan_findings SET category = ?, interest_score = ?, tags = ? WHERE id = ?",
					cl.Category, cl.InterestScore, strings.Join(cl.Tags, ","), f.ID,
				)
				if err != nil {
					tx.Rollback()
					cobra.CheckErr(err)
				}
			}

			cobra.CheckErr(tx.Commit())
			fmt.Printf("Scan %s: classified %d findings\n", scan.ScanID[:8], len(findings))
		}

		fmt.Println("Backfill complete.")
	},
}

func init() {
	rootCmd.AddCommand(backfillCmd)
}
