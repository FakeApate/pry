// Copyright (C) 2026 fakeapate <fakeapate@pm.me>
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// These are populated by -ldflags at release build time (see .goreleaser.yaml).
// Local `go build` falls back to values from debug.ReadBuildInfo.
var (
	version = "dev"
	commit  = ""
	date    = ""
)

func init() {
	rootCmd.Version = buildVersionString()
	rootCmd.SetVersionTemplate("{{.Version}}\n")
}

func buildVersionString() string {
	v, c, d := version, commit, date

	// Fill in anything the ldflags didn't set from Go's embedded build info.
	if info, ok := debug.ReadBuildInfo(); ok {
		if v == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			v = info.Main.Version
		}
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				if c == "" {
					c = s.Value
				}
			case "vcs.time":
				if d == "" {
					d = s.Value
				}
			}
		}
	}

	if len(c) > 7 {
		c = c[:7]
	}

	var b strings.Builder
	fmt.Fprintf(&b, "pry %s\n", v)
	if c != "" {
		fmt.Fprintf(&b, "  commit: %s\n", c)
	}
	if d != "" {
		fmt.Fprintf(&b, "  built:  %s\n", d)
	}
	fmt.Fprintf(&b, "  go:     %s\n", runtime.Version())
	return strings.TrimRight(b.String(), "\n")
}
