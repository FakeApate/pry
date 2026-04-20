// Copyright (C) 2026 fakeapate <fakeapate@pm.me>
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fakeapate/pry/config"
	"github.com/fakeapate/pry/internal/export"
	"github.com/fakeapate/pry/internal/store"
	storedb "github.com/fakeapate/pry/internal/store/db"
	"github.com/fakeapate/pry/internal/tree"
	"github.com/spf13/cobra"
)

var (
	exportFormat string
	exportOutput string
	exportLast   bool
)

var exportCmd = &cobra.Command{
	Use:   "export [scan-id]",
	Short: "Export scan results as HTML, JSON, or CSV",
	Long: `export renders a completed scan to a file. HTML is self-contained
with an interactive collapsible tree, search, and category filters.
JSON preserves the tree hierarchy; CSV is a flat table of findings.

Use --last to export the most recent scan without typing its ID.`,
	Example: `  pry export --last
  pry export 3a0f73b0-c444-4dc2-80ca-15f821198e71
  pry export --last --format json --output results.json
  pry export --last --format csv --output results.csv`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.GetConfig()
		database, err := store.OpenDB(cfg.Database.DBPath)
		cobra.CheckErr(err)
		defer database.Close()
		cobra.CheckErr(store.MigrateUp(database))

		queries := storedb.New(database)
		ctx := context.Background()

		// Resolve scan ID
		var scanID string
		if exportLast {
			scans, err := queries.ListScans(ctx)
			cobra.CheckErr(err)
			if len(scans) == 0 {
				fmt.Println("No scans found.")
				os.Exit(1)
			}
			scanID = scans[0].ScanID
		} else if len(args) > 0 {
			scanID = args[0]
		} else {
			fmt.Println("Specify a scan ID or use --last")
			os.Exit(1)
		}

		scan, err := queries.GetScan(ctx, scanID)
		cobra.CheckErr(err)

		// Load all findings
		fs := store.NewFindingsStore(database)
		result, err := fs.QueryFindings(ctx, store.FindingsFilter{
			ScanID:   scanID,
			Page:     1,
			PageSize: 1000000,
			SortBy:   "url",
		})
		cobra.CheckErr(err)

		// Build tree
		treeFindings := make([]tree.Finding, len(result.Findings))
		exportFindings := make([]export.Finding, len(result.Findings))
		for i, f := range result.Findings {
			treeFindings[i] = tree.Finding{
				URL:           f.URL,
				ContentType:   f.ContentType,
				ContentLength: f.ContentLength,
				Category:      f.Category,
				Interest:      f.InterestScore,
				LastModified:  f.LastModified,
			}
			exportFindings[i] = export.Finding{
				URL:           f.URL,
				ContentType:   f.ContentType,
				ContentLength: f.ContentLength,
				Category:      f.Category,
				InterestScore: f.InterestScore,
				Tags:          f.Tags,
				LastModified:  f.LastModified,
			}
		}
		root := tree.Build(scan.Url, treeFindings)

		scanDate, _ := time.Parse("2006-01-02 15:04:05", scan.CreatedAt)
		data := export.ScanData{
			ScanID:   scanID,
			URL:      scan.Url,
			ScanDate: scanDate,
			Tree:     root,
			Findings: exportFindings,
			Total:    result.Total,
		}

		// Pick exporter
		var exporter export.Exporter
		switch exportFormat {
		case "json":
			exporter = export.JSONExporter{}
		case "csv":
			exporter = export.CSVExporter{}
		default:
			exporter = export.HTMLExporter{}
		}

		// Determine output path
		outPath := exportOutput
		if outPath == "" {
			ext := exportFormat
			if ext == "" {
				ext = "html"
			}
			prefix := scanID
			if len(prefix) > 8 {
				prefix = prefix[:8]
			}
			outPath = fmt.Sprintf("pry-%s.%s", prefix, ext)
		}

		f, err := os.Create(outPath)
		cobra.CheckErr(err)
		defer f.Close()

		cobra.CheckErr(exporter.Export(f, data))
		fmt.Printf("Exported %d findings to %s\n", result.Total, outPath)
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringVarP(&exportFormat, "format", "f", "html", "export format: html, json, csv")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file path")
	exportCmd.Flags().BoolVar(&exportLast, "last", false, "export the most recent scan")
}
